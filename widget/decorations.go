package widget

import (
	"math/bits"

	"github.com/nanorele/gio/io/system"
	"github.com/nanorele/gio/layout"
	"github.com/nanorele/gio/op/clip"
)

type Decorations struct {
	Maximized bool
	clicks    map[int]*Clickable
}

func (d *Decorations) LayoutMove(gtx layout.Context, w layout.Widget) layout.Dimensions {
	dims := w(gtx)
	defer clip.Rect{Max: dims.Size}.Push(gtx.Ops).Pop()
	system.ActionInputOp(system.ActionMove).Add(gtx.Ops)
	return dims
}

func (d *Decorations) Clickable(action system.Action) *Clickable {
	if bits.OnesCount(uint(action)) != 1 {
		panic("not a single action")
	}
	idx := bits.TrailingZeros(uint(action))
	click, found := d.clicks[idx]
	if !found {
		click = new(Clickable)
		if d.clicks == nil {
			d.clicks = make(map[int]*Clickable)
		}
		d.clicks[idx] = click
	}
	return click
}

func (d *Decorations) Update(gtx layout.Context) system.Action {
	var actions system.Action
	for idx, clk := range d.clicks {
		if !clk.Clicked(gtx) {
			continue
		}
		action := system.Action(1 << idx)
		switch {
		case action == system.ActionMaximize && d.Maximized:
			action = system.ActionUnmaximize
		case action == system.ActionUnmaximize && !d.Maximized:
			action = system.ActionMaximize
		}
		actions |= action
	}
	return actions
}
