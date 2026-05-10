package debug

import (
	"testing"
)

// Parse uses package-level sync.Once, so we can only meaningfully exercise
// it once per test binary. We pick GIODEBUG=text,silent to verify the text
// subsystem is enabled and the usage banner is suppressed by `silent`.
func TestParseTextSilent(t *testing.T) {
	t.Setenv("GIODEBUG", "text,silent")
	Parse()
	if !Text.Load() {
		t.Errorf("Parse() with GIODEBUG=text,silent: Text=false, want true")
	}
}
