package gofont

import (
	"testing"

	"github.com/nanorele/gio/font"
)

func TestRegular(t *testing.T) {
	faces := Regular()
	if len(faces) == 0 {
		t.Error("Regular() returned no font faces")
	}
}

func TestCollection(t *testing.T) {
	faces := Collection()
	if len(faces) == 0 {
		t.Error("Collection() returned no font faces")
	}
	// It should have several fonts
	if len(faces) < 5 {
		t.Errorf("Collection() returned only %d faces, expected more", len(faces))
	}
}

func TestCollection_AllFacesNonNil(t *testing.T) {
	faces := Collection()
	if len(faces) == 0 {
		t.Fatal("Collection() returned no faces")
	}
	for i, ff := range faces {
		if ff.Face == nil {
			t.Errorf("face[%d] has nil Face", i)
		}
		if ff.Face != nil && ff.Face.Face() == nil {
			t.Errorf("face[%d].Face().Face() is nil", i)
		}
		if ff.Font.Typeface == "" {
			t.Errorf("face[%d] has empty Typeface", i)
		}
	}
}

func TestCollection_StableAcrossCalls(t *testing.T) {
	a := Collection()
	b := Collection()
	if len(a) != len(b) {
		t.Fatalf("Collection length not stable: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i].Font != b[i].Font {
			t.Errorf("face[%d] Font differs across calls", i)
		}
		if a[i].Face != b[i].Face {
			t.Errorf("face[%d] Face differs across calls", i)
		}
	}
}

func TestCollection_CountAtLeast12(t *testing.T) {
	// One regular plus 11 registered fonts = 12 entries.
	faces := Collection()
	if len(faces) < 12 {
		t.Errorf("expected at least 12 faces, got %d", len(faces))
	}
}

func TestRegular_NotEmpty(t *testing.T) {
	faces := Regular()
	if len(faces) == 0 {
		t.Fatal("Regular() returned 0 faces")
	}
	for i, ff := range faces {
		if ff.Face == nil {
			t.Errorf("Regular()[%d] has nil Face", i)
		}
	}
}

func TestRegular_Idempotent(t *testing.T) {
	a := Regular()
	b := Regular()
	if len(a) != len(b) {
		t.Errorf("Regular() length not stable")
	}
}

func TestCollection_IncludesItalicAndHeavyWeight(t *testing.T) {
	faces := Collection()
	var sawItalic, sawHeavy bool
	for _, ff := range faces {
		if ff.Font.Style == font.Italic {
			sawItalic = true
		}
		// Go's "gobold" parses with the OS/2 weight class corresponding to
		// SemiBold rather than Bold, so accept any weight heavier than Normal.
		if ff.Font.Weight > font.Normal {
			sawHeavy = true
		}
	}
	if !sawItalic {
		t.Error("Collection() has no italic face")
	}
	if !sawHeavy {
		t.Error("Collection() has no face heavier than Normal")
	}
}

func TestCollection_FirstFaceIsRegular(t *testing.T) {
	faces := Collection()
	if len(faces) == 0 {
		t.Fatal("empty collection")
	}
	first := faces[0]
	if first.Font.Style != font.Regular {
		t.Errorf("first face Style = %v; want Regular", first.Font.Style)
	}
	if first.Font.Weight != font.Normal {
		t.Errorf("first face Weight = %v; want Normal", first.Font.Weight)
	}
}

func TestCollection_NoDuplicateRegular(t *testing.T) {
	// goregular.TTF is registered once via loadRegular; ensure Collection
	// does not contain a duplicate first face.
	faces := Collection()
	if len(faces) < 2 {
		t.Skip("not enough faces to check duplication")
	}
	regularCount := 0
	for _, ff := range faces {
		if ff.Font.Style == font.Regular && ff.Font.Weight == font.Normal {
			// Multiple typefaces (proportional vs mono) can both be Regular/Normal.
			if ff.Font.Typeface == faces[0].Font.Typeface {
				regularCount++
			}
		}
	}
	if regularCount > 1 {
		t.Errorf("found %d copies of the regular face; expected 1", regularCount)
	}
}
