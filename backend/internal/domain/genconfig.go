package domain

// GenConfig is the template recipe (docs/GENERATION_PIPELINE.md §7.3).
type GenConfig struct {
	Engine            string         `json:"engine,omitempty"`
	Provider          string         `json:"provider,omitempty"`
	Model             string         `json:"model,omitempty"`
	Mode              string         `json:"mode"` // img2img | reference | edit
	Prompt            string         `json:"prompt"`
	NegativePrompt    string         `json:"negative_prompt,omitempty"`
	Strength          float64        `json:"strength,omitempty"`
	Output            *GenOutput     `json:"output,omitempty"`
	IdentityCheck     bool           `json:"identity_check"`
	IdentityThreshold float64        `json:"identity_threshold,omitempty"`
	Post              []PostOp       `json:"post,omitempty"`
	Style             string         `json:"style,omitempty"` // photo | illustration
	FallbackProvider  string         `json:"fallback_provider,omitempty"`
	Extra             map[string]any `json:"extra,omitempty"`
}

type GenOutput struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type PostOp struct {
	Op     string `json:"op"` // square_crop | resize | matte_solid_bg
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
	Color  string `json:"color,omitempty"`
}

// CropRule overrides for a spec (docs/GENERATION_PIPELINE.md §7.1).
type CropRule struct {
	HeadRatio float64 `json:"head_ratio,omitempty"`
	TopMargin float64 `json:"top_margin,omitempty"`
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
	FaceBox [4]int   `json:"face_box"`
	Reasons []string `json:"reasons"`
	Blur    float64  `json:"blur"`
	Luma    float64  `json:"luma"`
	Gender  string   `json:"gender,omitempty"`
}
