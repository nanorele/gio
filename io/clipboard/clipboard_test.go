package clipboard

import "testing"

// commandMarker mirrors io/input.Command without importing it (avoiding cycles).
type commandMarker interface {
	ImplementsCommand()
}

func TestCommandMarkers(t *testing.T) {
	var _ commandMarker = WriteCmd{}
	var _ commandMarker = ReadCmd{}
}

func TestWriteCmdFields(t *testing.T) {
	w := WriteCmd{Type: "text/plain"}
	if w.Type != "text/plain" {
		t.Errorf("WriteCmd.Type = %q, want %q", w.Type, "text/plain")
	}
	if w.Data != nil {
		t.Errorf("WriteCmd.Data should default to nil")
	}
}

func TestReadCmdFields(t *testing.T) {
	tag := new(int)
	r := ReadCmd{Tag: tag}
	if r.Tag != tag {
		t.Errorf("ReadCmd.Tag should round-trip")
	}
}
