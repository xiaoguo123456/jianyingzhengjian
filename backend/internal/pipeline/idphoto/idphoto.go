// Package idphoto implements the ID photo pipeline (docs/GENERATION_PIPELINE.md §7.1).
package idphoto

import (
	"context"
	"fmt"
	"math/rand"
	"strings"

	"yingji/backend/internal/domain"
	"yingji/backend/internal/engine/local"
	"yingji/backend/internal/pipeline/steps"
	"yingji/backend/internal/pkg/apperr"
	gm "yingji/backend/internal/provider/genmodel"
)

// UsesGen reports whether the parameters require the gen model (D-21).
func UsesGen(p domain.IDPhotoParams) bool {
	return (p.Clothing != "" && p.Clothing != "keep") || p.Beauty == "light"
}

func Run(ctx context.Context, d *steps.Deps, st *steps.State) (*steps.Output, error) {
	if st.Spec == nil {
		return nil, steps.Fail(domain.ErrProvider, fmt.Errorf("spec missing"))
	}
	st.Stage(domain.StageQueued)
	if err := steps.Prepare(st, 3000); err != nil {
		return nil, err
	}
	if err := steps.Detect(ctx, d, st); err != nil {
		return nil, err
	}

	if UsesGen(st.Params) {
		st.Stage(domain.StageProcessing)
		original := st.Raw
		instruction := buildInstruction(st.Params, st.Face.Gender)
		req := gm.Request{Mode: gm.ModeEdit, Source: st.Raw, Prompt: instruction, Seed: rand.Int63()}
		res, provider, err := d.Gen.Run(ctx, "", "", req)
		if err != nil {
			return nil, mapGenErr(err)
		}
		st.UsedGen, st.Cost, st.Provider, st.ProviderRef, st.Seed = true, st.Cost+res.CostCents, provider, res.ProviderRef, res.Seed
		if err := steps.ReplaceSource(st, res.Image); err != nil {
			return nil, err
		}
		if err := steps.Detect(ctx, d, st); err != nil {
			return nil, err
		}
		threshold := d.Cfg.Float(ctx, "identity_threshold")
		ok, score, err := steps.Identity(ctx, d, original, st.Raw, threshold)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, steps.Fail(domain.ErrIdentityMismatch, fmt.Errorf("score %.2f < %.2f", score, threshold))
		}
	}

	st.Stage(domain.StageFinishing)
	var rule domain.CropRule
	_ = st.Spec.CropRule.Into(&rule)
	cropped, _ := local.CropSpec(st.Img, st.Face.Box, st.Spec.WidthPx, st.Spec.HeightPx, rule)
	croppedRaw, err := local.EncodeJPEG(cropped, 95)
	if err != nil {
		return nil, steps.Fail(domain.ErrStorage, err)
	}
	// re-detect on the crop so the matting provider gets a face box in crop coordinates
	f, err := d.Vision.Detect(ctx, cropped, croppedRaw)
	if err != nil || f.Faces == 0 {
		f = st.Face
	}
	alpha, err := d.Vision.Matte(ctx, cropped, croppedRaw, f)
	if err != nil {
		return nil, steps.Fail(domain.ErrVision, err)
	}
	bg := st.Params.Bg
	if bg == "" {
		bg = st.Spec.BgDefault
	}
	out, err := local.CompositeSolid(cropped, alpha, bg)
	if err != nil {
		return nil, steps.Fail(domain.ErrProvider, err)
	}
	st.Out, st.OutFormat, st.Alpha = out, "png", alpha
	meta := map[string]any{
		"bg": strings.ToUpper(bg), "clothing": st.Params.Clothing, "beauty": st.Params.Beauty,
		"width_mm": st.Spec.WidthMM, "height_mm": st.Spec.HeightMM, "dpi": st.Spec.DPI,
	}
	return steps.Finish(d, st, st.Spec.DPI, meta)
}

func buildInstruction(p domain.IDPhotoParams, gender string) string {
	var parts []string
	if c := domain.ClothingByID(p.Clothing); c != nil && c.Prompt != "" {
		parts = append(parts, c.Prompt)
	}
	if p.Beauty == "light" {
		parts = append(parts, "light natural skin retouch, keep facial features unchanged")
	}
	parts = append(parts, "keep the same person, same face, frontal, neutral expression, plain background")
	if gender != "" {
		parts = append(parts, gender)
	}
	return strings.Join(parts, ", ")
}

func mapGenErr(err error) error {
	if apperr.Is(err, "CONTENT_REJECTED") {
		return steps.Fail(domain.ErrContentRejected, err)
	}
	if apperr.Is(err, "GENERATION_UNAVAILABLE") {
		return apperr.Transient(domain.ErrProvider, err)
	}
	return steps.Fail(domain.ErrProvider, err)
}

// Recolor re-composites a stored ID photo over a new background using its alpha (free op, D-05).
func Recolor(workImage, alphaPNG []byte, bg string) ([]byte, []byte, error) {
	img, _, err := local.Decode(workImage)
	if err != nil {
		return nil, nil, err
	}
	alpha, err := local.DecodeAlphaPNG(alphaPNG)
	if err != nil {
		return nil, nil, err
	}
	out, err := local.CompositeSolid(img, alpha, bg)
	if err != nil {
		return nil, nil, err
	}
	data, err := local.EncodePNG(out)
	if err != nil {
		return nil, nil, err
	}
	thumb, err := local.Thumb(out, 600)
	return data, thumb, err
}
