package layout

import (
	"image"

	"github.com/nanorele/gio/op"
)

type Stack struct {
	Alignment Direction
}

type StackChild struct {
	expanded bool
	widget   Widget
}

func Stacked(w Widget) StackChild {
	return StackChild{
		widget: w,
	}
}

func Expanded(w Widget) StackChild {
	return StackChild{
		expanded: true,
		widget:   w,
	}
}

func (s Stack) Layout(gtx Context, children ...StackChild) Dimensions {
	var maxSZ image.Point

	cgtx := gtx
	cgtx.Constraints.Min = image.Point{}

	type scratchSpace struct {
		call op.CallOp
		dims Dimensions
	}
	var scratchArray [32]scratchSpace
	var scratch []scratchSpace
	if len(children) <= len(scratchArray) {
		scratch = scratchArray[:len(children)]
	} else {
		scratch = make([]scratchSpace, len(children))
	}
	for i, w := range children {
		if w.expanded {
			continue
		}
		macro := op.Record(gtx.Ops)
		dims := w.widget(cgtx)
		call := macro.Stop()
		if w := dims.Size.X; w > maxSZ.X {
			maxSZ.X = w
		}
		if h := dims.Size.Y; h > maxSZ.Y {
			maxSZ.Y = h
		}
		scratch[i].call = call
		scratch[i].dims = dims
	}

	for i, w := range children {
		if !w.expanded {
			continue
		}
		macro := op.Record(gtx.Ops)
		cgtx.Constraints.Min = maxSZ
		dims := w.widget(cgtx)
		call := macro.Stop()
		if w := dims.Size.X; w > maxSZ.X {
			maxSZ.X = w
		}
		if h := dims.Size.Y; h > maxSZ.Y {
			maxSZ.Y = h
		}
		scratch[i].call = call
		scratch[i].dims = dims
	}

	maxSZ = gtx.Constraints.Constrain(maxSZ)
	var baseline int
	for _, scratchChild := range scratch {
		sz := scratchChild.dims.Size
		var p image.Point
		switch s.Alignment {
		case N, S, Center:
			p.X = (maxSZ.X - sz.X) / 2
		case NE, SE, E:
			p.X = maxSZ.X - sz.X
		}
		switch s.Alignment {
		case W, Center, E:
			p.Y = (maxSZ.Y - sz.Y) / 2
		case SW, S, SE:
			p.Y = maxSZ.Y - sz.Y
		}
		trans := op.Offset(p).Push(gtx.Ops)
		scratchChild.call.Add(gtx.Ops)
		trans.Pop()
		if baseline == 0 {
			if b := scratchChild.dims.Baseline; b != 0 {
				baseline = b + maxSZ.Y - sz.Y - p.Y
			}
		}
	}
	return Dimensions{
		Size:     maxSZ,
		Baseline: baseline,
	}
}

type Background struct{}

func (Background) Layout(gtx Context, background, widget Widget) Dimensions {
	macro := op.Record(gtx.Ops)
	wdims := widget(gtx)
	baseline := wdims.Baseline
	call := macro.Stop()

	cgtx := gtx
	cgtx.Constraints.Min = gtx.Constraints.Constrain(wdims.Size)
	bdims := background(cgtx)

	if bdims.Size != wdims.Size {
		p := image.Point{
			X: (bdims.Size.X - wdims.Size.X) / 2,
			Y: (bdims.Size.Y - wdims.Size.Y) / 2,
		}
		baseline += (bdims.Size.Y - wdims.Size.Y) / 2
		trans := op.Offset(p).Push(gtx.Ops)
		defer trans.Pop()
	}

	call.Add(gtx.Ops)

	return Dimensions{
		Size:     bdims.Size,
		Baseline: baseline,
	}
}
