package input

import (
	"testing"
	"github.com/nanorele/gio/io/key"
)

func TestRouter_Extra(t *testing.T) {
	var r Router
	SystemEvent{}.ImplementsEvent()
	
	src := r.Source()
	_ = src.Enabled()
	dsrc := src.Disabled()
	if dsrc.Enabled() {
		t.Error("expected disabled source")
	}
	
	if s := ClickGesture.String(); s != "Click" {
		t.Errorf("got %q", s)
	}
	if s := ScrollGesture.String(); s != "Scroll" {
		t.Errorf("ScrollGesture.String()=%q want %q", s, "Scroll")
	}
	if s := (ClickGesture | ScrollGesture).String(); s != "Click,Scroll" {
		t.Errorf("(Click|Scroll).String()=%q want %q", s, "Click,Scroll")
	}
}

func TestKey_Extra(t *testing.T) {
	var k keyFilter
	k.Add(key.Filter{})
	k.Merge(keyFilter{key.Filter{Name: "A"}})
	
	if TextInputOpen.String() != "Open" {
		t.Error("TextInputState.String failed")
	}
}
