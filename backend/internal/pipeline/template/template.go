// Package template implements the gen-model pipeline for pro / portrait / avatar (docs/GENERATION_PIPELINE.md §7.2):
// the template prompt, the user photo and the template's reference images go to the gen model in one call,
// then the result is centre-cropped to the output size.
package template

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

func Run(ctx context.Context, d *steps.Deps, st *steps.State) (*steps.Output, error) {
	if st.Template == nil {
		return nil, steps.Fail(domain.ErrProvider, fmt.Errorf("template missing"))
	}
	var cfg domain.GenConfig
	if err := st.Template.GenConfig.Into(&cfg); err != nil {
		return nil, steps.Fail(domain.ErrProvider, fmt.Errorf("gen_config: %w", err))
	}
	st.Stage(domain.StageQueued)
	if err := steps.Prepare(st, 2048); err != nil {
		return nil, err
	}
	var refs [][]byte
	for _, key := range cfg.ReferenceKeys {
		b, err := steps.Asset(ctx, d, key)
		if err != nil {
			return nil, err
		}
		refs = append(refs, b)
	}

	st.Stage(domain.StageProcessing)
	mode := gm.Mode(cfg.Mode)
	switch {
	case len(refs) > 0:
		mode = gm.ModeReference
	case mode == "":
		mode = gm.ModeEdit
	}
	req := gm.Request{
		Mode: mode, Source: st.Raw, References: refs, Prompt: strings.ReplaceAll(cfg.Prompt, "{gender}", genderWord(st.Gender)),
		NegativePrompt: cfg.NegativePrompt, Strength: cfg.Strength, Model: cfg.Model, Seed: rand.Int63(), Extra: cfg.Extra,
	}
	if cfg.Output != nil {
		req.Width, req.Height = cfg.Output.Width, cfg.Output.Height
	}
	res, provider, err := d.Gen.Run(ctx, cfg.Provider, cfg.FallbackProvider, req)
	if err != nil {
		return nil, mapGenErr(err)
	}
	st.UsedGen, st.Provider, st.ProviderRef, st.Seed = true, provider, res.ProviderRef, res.Seed
	st.Cost += res.CostCents
	if err := steps.ReplaceSource(st, res.Image); err != nil {
		return nil, err
	}

	st.Stage(domain.StageFinishing)
	out := local.ToNRGBA(st.Img)
	for _, op := range cfg.Post {
		switch op.Op {
		case "square_crop":
			side := op.Width
			if side == 0 {
				side = 1024
			}
			out = local.Fill(out, side, side)
		case "resize":
			if op.Width > 0 && op.Height > 0 {
				out = local.Fill(out, op.Width, op.Height)
			}
		}
	}
	if st.Template.Module == domain.ModuleAvatar && out.Bounds().Dx() != out.Bounds().Dy() {
		out = local.Fill(out, 1024, 1024)
	}
	if cfg.Output != nil && cfg.Output.Width > 0 && cfg.Output.Height > 0 {
		out = local.Fill(out, cfg.Output.Width, cfg.Output.Height)
	}
	st.Out = out
	st.OutFormat = "jpeg"
	if cfg.Style == "illustration" && res.HasAlpha {
		st.OutFormat = "png"
	}
	meta := map[string]any{"template_id": st.Template.ID, "seed": st.Seed, "provider": provider, "mode": string(mode)}
	return steps.Finish(d, st, 0, meta)
}

// genderWord fills {gender} in template prompts; empty when the upload check could not tell.
func genderWord(g string) string {
	switch g {
	case "male":
		return "男性"
	case "female":
		return "女性"
	}
	return ""
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
