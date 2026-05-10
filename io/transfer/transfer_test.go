package transfer

import (
	"io"
	"testing"

	"github.com/nanorele/gio/io/event"
)

// commandMarker mirrors io/input.Command without importing it (avoiding cycles).
type commandMarker interface {
	ImplementsCommand()
}

func TestEventMarkers(t *testing.T) {
	var _ event.Event = RequestEvent{}
	var _ event.Event = InitiateEvent{}
	var _ event.Event = CancelEvent{}
	var _ event.Event = DataEvent{}
}

func TestCommandMarkers(t *testing.T) {
	var _ commandMarker = OfferCmd{}
}

func TestFilterMarkers(t *testing.T) {
	var _ event.Filter = SourceFilter{}
	var _ event.Filter = TargetFilter{}
}

func TestOfferCmdFields(t *testing.T) {
	tag := new(int)
	c := OfferCmd{Tag: tag, Type: "text/plain"}
	if c.Tag != tag {
		t.Errorf("OfferCmd.Tag should round-trip")
	}
	if c.Type != "text/plain" {
		t.Errorf("OfferCmd.Type = %q, want %q", c.Type, "text/plain")
	}
	if c.Data != nil {
		t.Errorf("OfferCmd.Data should default to nil")
	}
}

func TestDataEventOpen(t *testing.T) {
	called := false
	d := DataEvent{
		Type: "text/plain",
		Open: func() io.ReadCloser {
			called = true
			return nil
		},
	}
	if d.Open == nil {
		t.Fatalf("DataEvent.Open should not be nil")
	}
	d.Open()
	if !called {
		t.Errorf("DataEvent.Open() should invoke the closure")
	}
}

func TestSourceTargetFilterFields(t *testing.T) {
	tag := new(int)
	s := SourceFilter{Target: tag, Type: "text/plain"}
	if s.Target != tag || s.Type != "text/plain" {
		t.Errorf("SourceFilter fields should round-trip")
	}
	t2 := TargetFilter{Target: tag, Type: "text/html"}
	if t2.Target != tag || t2.Type != "text/html" {
		t.Errorf("TargetFilter fields should round-trip")
	}
}
