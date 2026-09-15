// Package genmodel defines the image-generation large-model providers.
package genmodel

import "context"

type Mode string

const (
	ModeImg2Img   Mode = "img2img"
	ModeReference Mode = "reference"
	ModeEdit      Mode = "edit"
)

type Caps struct {
	Img2Img, Reference, Edit, MaskEdit bool
	MaxSide                            int
	ReturnsAlpha                       bool
}

func (c Caps) Supports(m Mode) bool {
	switch m {
	case ModeImg2Img:
		return c.Img2Img
	case ModeReference:
		return c.Reference
	case ModeEdit:
		return c.Edit
	}
	return false
}

type Request struct {
	Mode           Mode
	Source         []byte
	References     [][]byte
	Mask           []byte
	Prompt         string
	NegativePrompt string
	Strength       float64
	Width, Height  int
	Seed           int64
	Model          string
	Extra          map[string]any
}

type Result struct {
	Image       []byte
	Format      string // jpeg | png
	CostCents   int
	ProviderRef string
	Seed        int64
	HasAlpha    bool
}

type Model interface {
	Name() string
	Capabilities() Caps
	Run(ctx context.Context, req Request) (Result, error)
}
