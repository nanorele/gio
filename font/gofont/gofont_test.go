package gofont

import (
	"testing"
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
