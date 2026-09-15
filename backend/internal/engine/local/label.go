package local

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"hash/crc32"
	"image"
	"image/color"
	"image/draw"
	"os"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// AIGCMetadata is the implicit label written into every generated file (docs/COMPLIANCE.md §3.1).
type AIGCMetadata struct {
	Label      string `json:"Label"`
	Producer   string `json:"ContentProducer"`
	ProduceID  string `json:"ProduceID"`
	ContentID  string `json:"ContentID"`
	Propagator string `json:"ContentPropagator,omitempty"`
}

// AddMetadata embeds the AIGC JSON: a PNG tEXt chunk or a JPEG COM segment.
// For PNG it also writes a pHYs chunk when dpi > 0.
func AddMetadata(data []byte, format string, meta AIGCMetadata, dpi int) []byte {
	js, _ := json.Marshal(meta)
	switch format {
	case "png":
		chunks := [][]byte{pngChunk("tEXt", append([]byte("AIGC\x00"), js...))}
		if dpi > 0 {
			ppm := uint32(float64(dpi) / 0.0254)
			b := make([]byte, 9)
			binary.BigEndian.PutUint32(b[0:], ppm)
			binary.BigEndian.PutUint32(b[4:], ppm)
			b[8] = 1
			chunks = append(chunks, pngChunk("pHYs", b))
		}
		return insertPNGChunks(data, chunks)
	case "jpeg", "jpg":
		return insertJPEGComment(data, append([]byte("AIGC:"), js...))
	}
	return data
}

func pngChunk(typ string, payload []byte) []byte {
	buf := make([]byte, 0, 12+len(payload))
	l := make([]byte, 4)
	binary.BigEndian.PutUint32(l, uint32(len(payload)))
	buf = append(buf, l...)
	buf = append(buf, typ...)
	buf = append(buf, payload...)
	crc := crc32.NewIEEE()
	crc.Write([]byte(typ))
	crc.Write(payload)
	c := make([]byte, 4)
	binary.BigEndian.PutUint32(c, crc.Sum32())
	return append(buf, c...)
}

// insertPNGChunks places chunks right after IHDR.
func insertPNGChunks(data []byte, chunks [][]byte) []byte {
	if len(data) < 33 || !bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")) {
		return data
	}
	ihdrEnd := 8 + 4 + 4 + 13 + 4
	out := make([]byte, 0, len(data)+256)
	out = append(out, data[:ihdrEnd]...)
	for _, c := range chunks {
		out = append(out, c...)
	}
	return append(out, data[ihdrEnd:]...)
}

// insertJPEGComment adds a COM segment right after SOI.
func insertJPEGComment(data []byte, comment []byte) []byte {
	if len(data) < 2 || data[0] != 0xFF || data[1] != 0xD8 {
		return data
	}
	if len(comment) > 65000 {
		comment = comment[:65000]
	}
	seg := []byte{0xFF, 0xFE, byte((len(comment) + 2) >> 8), byte((len(comment) + 2) & 0xFF)}
	seg = append(seg, comment...)
	out := make([]byte, 0, len(data)+len(seg))
	out = append(out, data[:2]...)
	out = append(out, seg...)
	return append(out, data[2:]...)
}

// DrawBadge paints the explicit "AI生成" label bottom-right (docs/COMPLIANCE.md §3.2).
// With a font it renders text; without one it draws a translucent badge with a glyph-less mark.
func DrawBadge(img *image.NRGBA, fontPath string) {
	b := img.Bounds()
	short := b.Dx()
	if b.Dy() < short {
		short = b.Dy()
	}
	h := int(float64(short) * 0.045)
	if h < 14 {
		h = 14
	}
	pad := h / 3
	w := h * 3
	face := loadFace(fontPath, float64(h)*0.7)
	if face != nil {
		w = int(font.MeasureString(face, "AI生成")>>6) + pad*2
	}
	rect := image.Rect(b.Max.X-w-pad*2, b.Max.Y-h-pad*2, b.Max.X-pad, b.Max.Y-pad)
	draw.Draw(img, rect, &image.Uniform{color.NRGBA{17, 24, 39, 150}}, image.Point{}, draw.Over)
	if face != nil {
		d := &font.Drawer{Dst: img, Src: image.White, Face: face,
			Dot: fixed.P(rect.Min.X+pad, rect.Max.Y-pad-int(float64(h)*0.12))}
		d.DrawString("AI生成")
		return
	}
	// no font: two small white bars as a neutral mark
	bar := image.Rect(rect.Min.X+pad, rect.Min.Y+h/3, rect.Max.X-pad, rect.Min.Y+h/3+h/8)
	draw.Draw(img, bar, image.White, image.Point{}, draw.Over)
	bar2 := bar.Add(image.Point{Y: h / 3})
	draw.Draw(img, bar2, image.White, image.Point{}, draw.Over)
}

var (
	faceMu    sync.Mutex
	faceCache = map[string]*opentype.Font{}
)

// loadFace parses a .ttf/.otf, or the first font of a .ttc/.otc collection
// (most CJK system fonts ship as collections).
func loadFace(path string, size float64) font.Face {
	if path == "" {
		return nil
	}
	faceMu.Lock()
	f, ok := faceCache[path]
	if !ok {
		data, err := os.ReadFile(path)
		if err != nil {
			faceMu.Unlock()
			return nil
		}
		f, err = opentype.Parse(data)
		if err != nil {
			coll, cerr := opentype.ParseCollection(data)
			if cerr != nil || coll.NumFonts() == 0 {
				faceMu.Unlock()
				return nil
			}
			if f, err = coll.Font(0); err != nil {
				faceMu.Unlock()
				return nil
			}
		}
		faceCache[path] = f
	}
	faceMu.Unlock()
	face, err := opentype.NewFace(f, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return nil
	}
	return face
}

// Face exposes font loading for the poster renderer.
func Face(path string, size float64) font.Face { return loadFace(path, size) }
