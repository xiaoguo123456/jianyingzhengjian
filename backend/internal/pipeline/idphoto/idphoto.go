// Package idphoto implements the ID photo pipeline (docs/GENERATION_PIPELINE.md §7.1): the gen model redraws
// the user photo as an ID photo from the idphoto_prompt instruction, then it is centre-cropped to the spec size.
package idphoto

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	"yingji/backend/internal/domain"
	"yingji/backend/internal/engine/local"
	"yingji/backend/internal/pipeline/steps"
	"yingji/backend/internal/pkg/apperr"
	gm "yingji/backend/internal/provider/genmodel"
)

func Run(ctx context.Context, d *steps.Deps, st *steps.State) (*steps.Output, error) {
	if st.Spec == nil {
		return nil, steps.Fail(domain.ErrProvider, fmt.Errorf("spec missing"))
	}
	st.Stage(domain.StageQueued)
	if err := steps.Prepare(st, 2048); err != nil {
		return nil, err
	}
	bg := st.Params.Bg
	if bg == "" {
		bg = st.Spec.BgDefault
	}

	st.Stage(domain.StageProcessing)
	req := gm.Request{Mode: gm.ModeEdit, Source: st.Raw, Prompt: Instruction(d.Cfg.String(ctx, "idphoto_prompt"), st.Spec, st.Params, bg),
		Width: st.Spec.WidthPx, Height: st.Spec.HeightPx, Seed: rand.Int63()}
	res, provider, err := d.Gen.Run(ctx, "", "", req)
	if err != nil {
		return nil, mapGenErr(err)
	}
	st.UsedGen, st.Cost, st.Provider, st.ProviderRef, st.Seed = true, st.Cost+res.CostCents, provider, res.ProviderRef, res.Seed
	if err := steps.ReplaceSource(st, res.Image); err != nil {
		return nil, err
	}

	st.Stage(domain.StageFinishing)
	// PNG keeps the spec DPI in pHYs; JPEG output carries no DPI.
	st.Out, st.OutFormat = local.Fill(st.Img, st.Spec.WidthPx, st.Spec.HeightPx), "png"
	meta := map[string]any{
		"bg": strings.ToUpper(bg), "clothing": st.Params.Clothing, "beauty": st.Params.Beauty,
		"width_mm": st.Spec.WidthMM, "height_mm": st.Spec.HeightMM, "dpi": st.Spec.DPI,
	}
	return steps.Finish(d, st, st.Spec.DPI, meta)
}

var bgNames = map[string]string{"#FFFFFF": "白色", "#438EDB": "蓝色", "#FF0000": "红色", "#808080": "灰色"}

// Instruction fills the idphoto_prompt template for one spec and parameter set.
func Instruction(tpl string, sp *domain.Spec, p domain.IDPhotoParams, bg string) string {
	bg = strings.ToUpper(bg)
	name := bgNames[bg]
	if name == "" {
		name = "背景色"
	}
	clothing := "保持原照片中的服装"
	if c := domain.ClothingByID(p.Clothing); c != nil && c.Prompt != "" {
		clothing = "服装换成：" + c.Prompt
	}
	beauty := "不做美颜，保留真实肤质"
	if p.Beauty == "light" {
		beauty = "轻度自然美颜：均匀肤色、淡化瑕疵，不改变五官"
	}
	ratio := strconv.FormatFloat(sp.WidthMM, 'f', -1, 64) + ":" + strconv.FormatFloat(sp.HeightMM, 'f', -1, 64)
	return strings.NewReplacer("{bg_name}", name, "{bg_hex}", bg, "{clothing}", clothing, "{beauty}", beauty, "{ratio}", ratio).Replace(tpl)
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
