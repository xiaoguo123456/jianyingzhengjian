// Package steps holds the shared pipeline state and reusable steps.
package steps

import (
	"context"
	"fmt"
	"image"
	"log/slog"
	"net/http"

	"yingji/backend/internal/config"
	"yingji/backend/internal/domain"
	"yingji/backend/internal/engine/genmodel"
	"yingji/backend/internal/engine/local"
	"yingji/backend/internal/engine/vision"
	"yingji/backend/internal/pkg/apperr"
	"yingji/backend/internal/provider/face"
	"yingji/backend/internal/provider/storage"
)

type Deps struct {
	Store    storage.ObjectStore
	Vision   *vision.Engine
	Gen      *genmodel.Router
	Cfg      *config.Runtime
	Log      *slog.Logger
	FontPath string
	AppID    string
}

type State struct {
	Task     *domain.Task
	Spec     *domain.Spec
	Template *domain.Template
	Params   domain.IDPhotoParams

	Raw  []byte
	Img  image.Image
	Face face.Result

	Out       *image.NRGBA
	OutFormat string
	Alpha     *image.Alpha

	UsedGen     bool
	Cost        int
	Provider    string
	ProviderRef string
	Seed        int64

	OnStage func(domain.TaskStage)
}

func (s *State) Stage(st domain.TaskStage) {
	if s.OnStage != nil {
		s.OnStage(st)
	}
}

type Output struct {
	Image       []byte
	Format      string
	Thumb       []byte
	Alpha       []byte
	Width       int
	Height      int
	Cost        int
	Provider    string
	ProviderRef string
	Seed        int64
	AILabel     bool
	Meta        map[string]any
}

// Fail wraps a pipeline failure with a task error code.
func Fail(code string, err error) error {
	msg := domain.TaskErrorMessages[code]
	e := apperr.New(code, http.StatusInternalServerError, msg)
	if err != nil {
		e = e.WithCause(err)
	}
	return e
}

// Prepare decodes and downscales the source photo.
func Prepare(st *State, maxSide int) error {
	img, _, err := local.Decode(st.Raw)
	if err != nil {
		return Fail(domain.ErrProvider, fmt.Errorf("decode source: %w", err))
	}
	st.Img = local.Downscale(img, maxSide)
	if st.Img != img {
		// keep Raw in sync with the working image for providers that take bytes
		if b, err := local.EncodeJPEG(st.Img, 95); err == nil {
			st.Raw = b
		}
	}
	return nil
}

// Detect requires exactly one face.
func Detect(ctx context.Context, d *Deps, st *State) error {
	f, err := d.Vision.Detect(ctx, st.Img, st.Raw)
	if err != nil {
		return Fail(domain.ErrVision, err)
	}
	if f.Faces != 1 {
		return Fail(domain.ErrNoFace, fmt.Errorf("faces=%d", f.Faces))
	}
	st.Face = f
	return nil
}

// ReplaceSource swaps the working image after a gen step.
func ReplaceSource(st *State, raw []byte) error {
	img, _, err := local.Decode(raw)
	if err != nil {
		return Fail(domain.ErrProvider, fmt.Errorf("decode generated: %w", err))
	}
	st.Raw = raw
	st.Img = img
	return nil
}

// Identity compares the original and the generated image; below threshold returns false.
func Identity(ctx context.Context, d *Deps, original, generated []byte, threshold float64) (bool, float64, error) {
	score, err := d.Vision.Compare(ctx, original, generated)
	if err != nil {
		return false, 0, Fail(domain.ErrVision, err)
	}
	return score >= threshold, score, nil
}

// Finish encodes the output, thumbnail, alpha and labels (docs/COMPLIANCE.md §3).
func Finish(d *Deps, st *State, dpi int, meta map[string]any) (*Output, error) {
	if st.Out == nil {
		return nil, Fail(domain.ErrProvider, fmt.Errorf("no output image"))
	}
	format := st.OutFormat
	if format == "" {
		format = "jpeg"
	}
	out := &Output{Format: format, Width: st.Out.Bounds().Dx(), Height: st.Out.Bounds().Dy(),
		Cost: st.Cost, Provider: st.Provider, ProviderRef: st.ProviderRef, Seed: st.Seed, AILabel: st.UsedGen, Meta: meta}
	if st.UsedGen {
		local.DrawBadge(st.Out, d.FontPath)
	}
	var data []byte
	var err error
	if format == "png" {
		data, err = local.EncodePNG(st.Out)
	} else {
		data, err = local.EncodeJPEG(st.Out, 92)
	}
	if err != nil {
		return nil, Fail(domain.ErrStorage, err)
	}
	label := local.AIGCMetadata{Label: "AIGC", Producer: "yingji", ProduceID: st.Task.ID, ContentID: st.Task.ID}
	if !st.UsedGen {
		label.Label = "edited"
	}
	out.Image = local.AddMetadata(data, format, label, dpi)
	if out.Thumb, err = local.Thumb(st.Out, 600); err != nil {
		return nil, Fail(domain.ErrStorage, err)
	}
	if st.Alpha != nil {
		if out.Alpha, err = local.EncodeAlphaPNG(st.Alpha); err != nil {
			return nil, Fail(domain.ErrStorage, err)
		}
	}
	return out, nil
}
