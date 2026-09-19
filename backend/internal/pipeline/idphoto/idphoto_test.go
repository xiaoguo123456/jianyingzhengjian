package idphoto

import (
	"strings"
	"testing"

	"yingji/backend/internal/config"
	"yingji/backend/internal/domain"
)

func TestInstructionFillsEveryPlaceholder(t *testing.T) {
	tpl := config.Defaults["idphoto_prompt"].(string)
	sp := &domain.Spec{WidthMM: 25, HeightMM: 35}
	got := Instruction(tpl, sp, domain.IDPhotoParams{Clothing: "m_black_suit", Beauty: "light"}, "#438edb")
	for _, want := range []string{"蓝色（#438EDB）", "服装换成：", "轻度自然美颜", "25:35"} {
		if !strings.Contains(got, want) {
			t.Errorf("instruction missing %q:\n%s", want, got)
		}
	}
	if strings.ContainsAny(got, "{}") {
		t.Errorf("unfilled placeholder:\n%s", got)
	}
	plain := Instruction(tpl, sp, domain.IDPhotoParams{Clothing: "keep", Beauty: "natural"}, "#FFFFFF")
	if !strings.Contains(plain, "保持原照片中的服装") || !strings.Contains(plain, "不做美颜") || !strings.Contains(plain, "白色") {
		t.Errorf("default options not described:\n%s", plain)
	}
}
