package widget_test

import (
	"image"
	"image/color"
	"testing"

	"github.com/nanorele/gio/f32"
	"github.com/nanorele/gio/io/input"
	"github.com/nanorele/gio/io/pointer"
	"github.com/nanorele/gio/io/semantic"
	"github.com/nanorele/gio/layout"
	"github.com/nanorele/gio/op"
	"github.com/nanorele/gio/widget"
)

func TestBorder(t *testing.T) {
	gtx := layout.Context{
		Ops: new(op.Ops),
		Constraints: layout.Exact(image.Pt(100, 100)),
	}
	border := widget.Border{
		Color: color.NRGBA{A: 255},
		Width: 2,
	}
	dims := border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Dimensions{Size: image.Pt(50, 50)}
	})
	if dims.Size != image.Pt(50, 50) {
		t.Errorf("expected size (50, 50), got %v", dims.Size)
	}
}

func TestBool(t *testing.T) {
	var (
		r input.Router
		b widget.Bool
	)
	gtx := layout.Context{
		Ops:    new(op.Ops),
		Source: r.Source(),
	}
	layout := func() {
		b.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			semantic.CheckBox.Add(gtx.Ops)
			semantic.DescriptionOp("description").Add(gtx.Ops)
			return layout.Dimensions{Size: image.Pt(100, 100)}
		})
	}
	if b.Hovered() || b.Pressed() {
		t.Error("Bool should not be hovered or pressed initially")
	}
	layout()
	r.Frame(gtx.Ops)
	r.Queue(
		pointer.Event{
			Source:   pointer.Touch,
			Kind:     pointer.Press,
			Position: f32.Pt(50, 50),
		},
		pointer.Event{
			Source:   pointer.Touch,
			Kind:     pointer.Release,
			Position: f32.Pt(50, 50),
		},
	)
	gtx.Reset()
	layout()
	r.Frame(gtx.Ops)
	tree := r.AppendSemantics(nil)
	n := tree[0].Children[0].Desc
	if n.Description != "description" {
		t.Errorf("unexpected semantic description: %s", n.Description)
	}
	if n.Class != semantic.CheckBox {
		t.Errorf("unexpected semantic class: %v", n.Class)
	}
	if !b.Value || !n.Selected {
		t.Error("click did not select")
	}
	_ = b.History()
}
