package widget

import (
	"image"

	"github.com/nanorele/gio/gesture"
	"github.com/nanorele/gio/io/pointer"
	"github.com/nanorele/gio/layout"
	"github.com/nanorele/gio/op/clip"
	"github.com/nanorele/gio/unit"
)

type Float struct {
	Value float32

	drag   gesture.Drag
	axis   layout.Axis
	length float32
}

func (f *Float) Dragging() bool { return f.drag.Dragging() }

func (f *Float) Layout(gtx layout.Context, axis layout.Axis, pointerMargin unit.Dp) layout.Dimensions {
	f.Update(gtx)
	size := gtx.Constraints.Min
	f.length = float32(axis.Convert(size).X)
	f.axis = axis

	margin := axis.Convert(image.Pt(gtx.Dp(pointerMargin), 0))
	rect := image.Rectangle{
		Min: margin.Mul(-1),
		Max: size.Add(margin),
	}
	defer clip.Rect(rect).Push(gtx.Ops).Pop()
	f.drag.Add(gtx.Ops)

	return layout.Dimensions{Size: size}
}

func (f *Float) Update(gtx layout.Context) bool {
	changed := false
	for {
		e, ok := f.drag.Update(gtx.Metric, gtx.Source, gesture.Axis(f.axis))
		if !ok {
			break
		}
		if f.length > 0 && (e.Kind == pointer.Press || e.Kind == pointer.Drag) {
			pos := e.Position.X
			if f.axis == layout.Vertical {
				pos = f.length - e.Position.Y
			}
			f.Value = pos / f.length
			if f.Value < 0 {
				f.Value = 0
			} else if f.Value > 1 {
				f.Value = 1
			}
			changed = true
		}
	}
	return changed
}
