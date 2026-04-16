package widget

import (
	"io"

	"github.com/nanorele/gio/f32"
	"github.com/nanorele/gio/gesture"
	"github.com/nanorele/gio/io/event"
	"github.com/nanorele/gio/io/pointer"
	"github.com/nanorele/gio/io/transfer"
	"github.com/nanorele/gio/layout"
	"github.com/nanorele/gio/op"
	"github.com/nanorele/gio/op/clip"
)

type Draggable struct {
	Type string

	drag  gesture.Drag
	click f32.Point
	pos   f32.Point
}

func (d *Draggable) Layout(gtx layout.Context, w, drag layout.Widget) layout.Dimensions {
	if !gtx.Enabled() {
		return w(gtx)
	}
	dims := w(gtx)

	stack := clip.Rect{Max: dims.Size}.Push(gtx.Ops)
	d.drag.Add(gtx.Ops)
	event.Op(gtx.Ops, d)
	stack.Pop()

	if drag != nil && d.drag.Pressed() {
		rec := op.Record(gtx.Ops)
		op.Offset(d.pos.Round()).Add(gtx.Ops)
		drag(gtx)
		op.Defer(gtx.Ops, rec.Stop())
	}

	return dims
}

func (d *Draggable) Dragging() bool {
	return d.drag.Dragging()
}

func (d *Draggable) Update(gtx layout.Context) (mime string, requested bool) {
	pos := d.pos
	for {
		ev, ok := d.drag.Update(gtx.Metric, gtx.Source, gesture.Both)
		if !ok {
			break
		}
		switch ev.Kind {
		case pointer.Press:
			d.click = ev.Position
			pos = f32.Point{}
		case pointer.Drag, pointer.Release:
			pos = ev.Position.Sub(d.click)
		}
	}
	d.pos = pos

	for {
		e, ok := gtx.Event(transfer.SourceFilter{Target: d, Type: d.Type})
		if !ok {
			break
		}
		if e, ok := e.(transfer.RequestEvent); ok {
			return e.Type, true
		}
	}
	return "", false
}

func (d *Draggable) Offer(gtx layout.Context, mime string, data io.ReadCloser) {
	gtx.Execute(transfer.OfferCmd{Tag: d, Type: mime, Data: data})
}

func (d *Draggable) Pos() f32.Point {
	return d.pos
}
