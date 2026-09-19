package local

import (
	"image"
	"image/color"
	"testing"
)

// solid returns a test image with a distinguishable gradient.
func solid(w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(x % 255), G: uint8(y % 255), B: 128, A: 255})
		}
	}
	return img
}

// TestFillHitsSpecSize: gen-model output (e.g. 1024×1536) must end at the exact spec pixels.
func TestFillHitsSpecSize(t *testing.T) {
	for _, tc := range []struct{ srcW, srcH, w, h int }{
		{1024, 1536, 295, 413}, {1024, 1536, 413, 579}, {1536, 1024, 390, 567}, {1024, 1024, 1024, 1024},
	} {
		out := Fill(solid(tc.srcW, tc.srcH), tc.w, tc.h)
		if out.Bounds().Dx() != tc.w || out.Bounds().Dy() != tc.h {
			t.Errorf("%dx%d -> %v, want %dx%d", tc.srcW, tc.srcH, out.Bounds(), tc.w, tc.h)
		}
	}
}

// TestAddMetadataKeepsImageDecodable: the AIGC label must not corrupt the file.
func TestAddMetadataKeepsImageDecodable(t *testing.T) {
	img := solid(40, 60)
	for _, format := range []string{"png", "jpeg"} {
		var data []byte
		var err error
		if format == "png" {
			data, err = EncodePNG(img)
		} else {
			data, err = EncodeJPEG(img, 90)
		}
		if err != nil {
			t.Fatalf("encode %s: %v", format, err)
		}
		labelled := AddMetadata(data, format, AIGCMetadata{Label: "AIGC", Producer: "yingji", ContentID: "x"}, 300)
		if len(labelled) <= len(data) {
			t.Errorf("%s: metadata not added (%d -> %d bytes)", format, len(data), len(labelled))
		}
		decoded, gotFormat, err := Decode(labelled)
		if err != nil {
			t.Fatalf("%s: labelled image does not decode: %v", format, err)
		}
		if decoded.Bounds() != img.Bounds() {
			t.Errorf("%s: bounds changed: %v", format, decoded.Bounds())
		}
		if gotFormat != format {
			t.Errorf("%s: format changed to %s", format, gotFormat)
		}
	}
}
