package event

import (
	"testing"

	"github.com/nanorele/gio/op"
)

func TestOp(t *testing.T) {
	var ops op.Ops
	tag := new(int)
	Op(&ops, tag)

	defer func() {
		if r := recover(); r == nil {
			t.Error("Op(nil) should panic")
		}
	}()
	Op(&ops, nil)
}

type fakeEvent struct{}

func (fakeEvent) ImplementsEvent() {}

type fakeFilter struct{}

func (fakeFilter) ImplementsFilter() {}

func TestEventInterface(t *testing.T) {
	var e Event = fakeEvent{}
	e.ImplementsEvent()
}

func TestFilterInterface(t *testing.T) {
	var f Filter = fakeFilter{}
	f.ImplementsFilter()
}

func TestTagAcceptsAny(t *testing.T) {
	tags := []Tag{
		new(int),
		new(string),
		new(struct{}),
		"string-tag",
		42,
	}
	for _, tg := range tags {
		if tg == nil {
			t.Error("tag should not be nil")
		}
	}
}

func TestOp_DistinctTags(t *testing.T) {
	var ops op.Ops
	tag1 := new(int)
	tag2 := new(int)
	Op(&ops, tag1)
	Op(&ops, tag2)
}

func TestOp_StringTag(t *testing.T) {
	var ops op.Ops
	// Tag is `any`; pointer-like values should work. Use a new struct value.
	type k struct{}
	Op(&ops, &k{})
}

func TestOp_PanicMessage(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for nil tag")
		}
		s, ok := r.(string)
		if !ok || s != "Tag must be non-nil" {
			t.Errorf("unexpected panic value: %#v", r)
		}
	}()
	var ops op.Ops
	Op(&ops, nil)
}
