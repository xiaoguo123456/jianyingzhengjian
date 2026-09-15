package local

import (
	"image"
	"image/color"
	"math"
	"testing"

	"yingji/backend/internal/domain"
)

// solid returns a test image with a distinguishable face region.
func solid(w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(x % 255), G: uint8(y % 255), B: 128, A: 255})
		}
	}
	return img
}

// TestCropSpecHeadRatio verifies the documented crop rule (docs/GENERATION_PIPELINE.md §7.1):
// the estimated head height must occupy head_ratio of the output height.
func TestCropSpecHeadRatio(t *testing.T) {
	const srcW, srcH = 2000, 2800
	src := solid(srcW, srcH)
	// face box: chin-to-brow, 400px tall, centred horizontally
	face := image.Rect(800, 700, 1200, 1100)
	specW, specH := 295, 413
	rule := domain.CropRule{HeadRatio: 0.62, TopMargin: 0.10}

	out, rect := CropSpec(src, face, specW, specH, rule)
	if out.Bounds().Dx() != specW || out.Bounds().Dy() != specH {
		t.Fatalf("output = %dx%d, want %dx%d", out.Bounds().Dx(), out.Bounds().Dy(), specW, specH)
	}
	// aspect of the crop rectangle must match the spec aspect
	gotAspect := float64(rect.Dx()) / float64(rect.Dy())
	wantAspect := float64(specW) / float64(specH)
	if math.Abs(gotAspect-wantAspect) > 0.01 {
		t.Errorf("crop aspect = %.4f, want %.4f", gotAspect, wantAspect)
	}
	// the head (1.45 x face height) must map to head_ratio of the output height
	scale := float64(specH) / float64(rect.Dy())
	headOut := 1.45 * float64(face.Dy()) * scale
	if ratio := headOut / float64(specH); math.Abs(ratio-rule.HeadRatio) > 0.02 {
		t.Errorf("head ratio = %.3f, want %.3f", ratio, rule.HeadRatio)
	}
	// top margin: distance from crop top to the crown, as a fraction of crop height
	crown := float64(face.Min.Y) - 0.30*float64(face.Dy())
	if margin := (crown - float64(rect.Min.Y)) / float64(rect.Dy()); math.Abs(margin-rule.TopMargin) > 0.02 {
		t.Errorf("top margin = %.3f, want %.3f", margin, rule.TopMargin)
	}
	// face must be horizontally centred in the crop
	faceCx := float64(face.Min.X+face.Max.X) / 2
	cropCx := float64(rect.Min.X+rect.Max.X) / 2
	if math.Abs(faceCx-cropCx) > 1 {
		t.Errorf("face centre %.1f != crop centre %.1f", faceCx, cropCx)
	}
}

// TestCropSpecOutOfFrame: a crop that leaves the source must still produce the exact spec size.
func TestCropSpecOutOfFrame(t *testing.T) {
	src := solid(400, 400)
	face := image.Rect(150, 20, 250, 120) // very close to the top edge
	out, rect := CropSpec(src, face, 295, 413, domain.CropRule{})
	if out.Bounds().Dx() != 295 || out.Bounds().Dy() != 413 {
		t.Fatalf("output = %v, want 295x413", out.Bounds())
	}
	if rect.Min.Y >= 0 {
		t.Logf("crop stayed in frame (rect=%v); edge extension not exercised", rect)
	}
}

func TestCropSpecDefaultsApplied(t *testing.T) {
	src := solid(1000, 1400)
	face := image.Rect(400, 350, 600, 550)
	_, rect := CropSpec(src, face, 295, 413, domain.CropRule{}) // zero rule -> defaults
	scale := float64(413) / float64(rect.Dy())
	headOut := 1.45 * float64(face.Dy()) * scale
	if ratio := headOut / 413; math.Abs(ratio-DefaultCropRule.HeadRatio) > 0.02 {
		t.Errorf("default head ratio = %.3f, want %.3f", ratio, DefaultCropRule.HeadRatio)
	}
}

func TestCompositeSolidUsesAlpha(t *testing.T) {
	src := solid(10, 10)
	alpha := image.NewAlpha(src.Bounds())
	// left half opaque, right half transparent
	for y := 0; y < 10; y++ {
		for x := 0; x < 5; x++ {
			alpha.SetAlpha(x, y, color.Alpha{A: 255})
		}
	}
	out, err := CompositeSolid(src, alpha, "#FF0000")
	if err != nil {
		t.Fatalf("composite: %v", err)
	}
	if r, _, _, _ := out.At(7, 5).RGBA(); r>>8 != 255 {
		t.Errorf("transparent area should be red background, got %v", out.At(7, 5))
	}
	if _, _, b, _ := out.At(2, 5).RGBA(); b>>8 != 128 {
		t.Errorf("opaque area should keep source pixels, got %v", out.At(2, 5))
	}
}

func TestParseHex(t *testing.T) {
	for _, tc := range []struct {
		in string
		ok bool
	}{
		{"#FFFFFF", true}, {"438EDB", true}, {"#ff0000", true}, {"#FFF", false}, {"nope", false},
	} {
		_, err := ParseHex(tc.in)
		if (err == nil) != tc.ok {
			t.Errorf("ParseHex(%q) err=%v, want ok=%v", tc.in, err, tc.ok)
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

func TestBlurScoreDistinguishesSharpFromFlat(t *testing.T) {
	flat := image.NewNRGBA(image.Rect(0, 0, 200, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 200; x++ {
			flat.SetNRGBA(x, y, color.NRGBA{R: 128, G: 128, B: 128, A: 255})
		}
	}
	sharp := solid(200, 200)
	if BlurScore(flat) >= BlurScore(sharp) {
		t.Errorf("flat image (%.2f) should score below detailed image (%.2f)", BlurScore(flat), BlurScore(sharp))
	}
}
