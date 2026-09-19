package domain

// GenConfig is the template recipe (docs/GENERATION_PIPELINE.md §7.3).
type GenConfig struct {
	Engine           string         `json:"engine,omitempty"`
	Provider         string         `json:"provider,omitempty"`
	Model            string         `json:"model,omitempty"`
	Mode             string         `json:"mode"` // img2img | reference | edit; reference when reference_keys are set
	Prompt           string         `json:"prompt"`
	NegativePrompt   string         `json:"negative_prompt,omitempty"`
	Strength         float64        `json:"strength,omitempty"`
	ReferenceKeys    []string       `json:"reference_keys,omitempty"` // assets/ images sent with the user photo
	Output           *GenOutput     `json:"output,omitempty"`
	Post             []PostOp       `json:"post,omitempty"`
	Style            string         `json:"style,omitempty"` // photo | illustration
	FallbackProvider string         `json:"fallback_provider,omitempty"`
	Extra            map[string]any `json:"extra,omitempty"`
}

// MaxReferenceImages caps reference_keys per template.
const MaxReferenceImages = 4

type GenOutput struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type PostOp struct {
	Op     string `json:"op"` // square_crop | resize (both centre-crop)
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

// IDPhotoParams are the user-selected parameters of an ID photo task.
type IDPhotoParams struct {
	Bg       string `json:"bg"`
	Clothing string `json:"clothing"`
	Beauty   string `json:"beauty"`
}

// ClothingOption is a clothing template for ID photos (static catalogue in V1).
type ClothingOption struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Group  string `json:"group"` // keep | male | female
	Prompt string `json:"-"`
}

var ClothingOptions = []ClothingOption{
	{ID: "keep", Name: "保持原服装", Group: "keep"},
	{ID: "m_white_shirt", Name: "白衬衫", Group: "male", Prompt: "wearing a crisp white dress shirt, formal, ID photo"},
	{ID: "m_black_suit", Name: "黑西装", Group: "male", Prompt: "wearing a black suit with white shirt and tie, formal, ID photo"},
	{ID: "m_navy_suit", Name: "深蓝西装", Group: "male", Prompt: "wearing a navy suit with white shirt, formal, ID photo"},
	{ID: "f_white_shirt", Name: "白衬衫", Group: "female", Prompt: "wearing a white blouse, formal, ID photo"},
	{ID: "f_suit", Name: "女士职业装", Group: "female", Prompt: "wearing a women's business suit, formal, ID photo"},
	{ID: "f_navy_suit", Name: "深蓝西装", Group: "female", Prompt: "wearing a navy women's blazer with white blouse, formal, ID photo"},
}

func ClothingByID(id string) *ClothingOption {
	for i := range ClothingOptions {
		if ClothingOptions[i].ID == id {
			return &ClothingOptions[i]
		}
	}
	return nil
}

// PhotoCheckResult is stored in photos.check_result.
type PhotoCheckResult struct {
	Passed  bool     `json:"passed"`
	Faces   int      `json:"faces"`
	Reasons []string `json:"reasons"`
	Gender  string   `json:"gender,omitempty"`
}
