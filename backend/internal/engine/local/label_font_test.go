package local

import (
	"os"
	"testing"
)

// A .ttc collection (the common form for CJK system fonts) must load.
func TestLoadFaceHandlesCollections(t *testing.T) {
	const ttc = "/System/Library/Fonts/Supplemental/Songti.ttc"
	if _, err := os.Stat(ttc); err != nil {
		t.Skip("no .ttc font on this machine")
	}
	if Face(ttc, 24) == nil {
		t.Error("Face returned nil for a .ttc collection")
	}
}

func TestLoadFaceMissingPathIsNil(t *testing.T) {
	if Face("", 24) != nil {
		t.Error("empty path should yield no face")
	}
	if Face("/nope/does-not-exist.ttf", 24) != nil {
		t.Error("missing file should yield no face")
	}
}
