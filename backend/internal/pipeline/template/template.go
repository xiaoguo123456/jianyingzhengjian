// Package template implements the gen-model pipeline for pro / portrait / avatar (docs/GENERATION_PIPELINE.md §7.2).
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
	if err := steps.Detect(ctx, d, st); err != nil {
		return nil, err
	}
	original := st.Raw

	st.Stage(domain.StageProcessing)
	mode := gm.Mode(cfg.Mode)
	if mode == "" {
		mode = gm.ModeImg2Img
	}
	req := gm.Request{
		Mode: mode, Source: st.Raw, Prompt: strings.ReplaceAll(cfg.Prompt, "{gender}", st.Face.Gender),
		NegativePrompt: cfg.NegativePrompt, Strength: cfg.Strength, Model: cfg.Model, Seed: rand.Int63(), Extra: cfg.Extra,
	}
	if cfg.Output != nil {
		req.Width, req.Height = cfg.Output.Width, cfg.Output.Height
	}
	threshold := cfg.IdentityThreshold
	if threshold <= 0 {
		threshold = d.Cfg.Float(ctx, "identity_threshold")
	}

	var res gm.Result
	var provider string
	var err error
	for attempt := 0; attempt < 2; attempt++ {
		res, provider, err = d.Gen.Run(ctx, cfg.Provider, cfg.FallbackProvider, req)
		if err != nil {
			return nil, mapGenErr(err)
		}
		st.UsedGen, st.Provider, st.ProviderRef, st.Seed = true, provider, res.ProviderRef, res.Seed
		st.Cost += res.CostCents
		if !cfg.IdentityCheck || cfg.Style == "illustration" {
			break
		}
		ok, score, ierr := steps.Identity(ctx, d, original, res.Image, threshold)
		if ierr != nil {
			return nil, ierr
		}
		if ok {
			break
		}
		if attempt == 1 {
			return nil, steps.Fail(domain.ErrIdentityMismatch, fmt.Errorf("score %.2f < %.2f", score, threshold))
		}
		if req.Strength > 0.15 {
			req.Strength -= 0.1
		}
		req.Seed = rand.Int63()
	}
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
			f, err := d.Vision.Detect(ctx, out, nil)
			if err != nil {
				f = st.Face
			}
			out = local.SquareCropFace(out, f.Box, side)
		case "resize":
			if op.Width > 0 && op.Height > 0 {
				out = local.Fill(out, op.Width, op.Height)
			}
		case "matte_solid_bg":
			raw, _ := local.EncodeJPEG(out, 95)
			f, err := d.Vision.Detect(ctx, out, raw)
			if err != nil {
				f = st.Face
			}
			alpha, err := d.Vision.Matte(ctx, out, raw, f)
			if err != nil {
				return nil, steps.Fail(domain.ErrVision, err)
			}
			color := op.Color
			if color == "" {
				color = "#FFFFFF"
			}
			if out, err = local.CompositeSolid(out, alpha, color); err != nil {
				return nil, steps.Fail(domain.ErrProvider, err)
			}
		}
	}
	if st.Template.Module == domain.ModuleAvatar && out.Bounds().Dx() != out.Bounds().Dy() {
		out = local.SquareCropFace(out, st.Face.Box, 1024)
	}
	st.Out = out
	st.OutFormat = "jpeg"
	if cfg.Style == "illustration" && res.HasAlpha {
		st.OutFormat = "png"
	}
	meta := map[string]any{"template_id": st.Template.ID, "seed": st.Seed, "provider": provider, "mode": string(mode)}
	return steps.Finish(d, st, 0, meta)
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
