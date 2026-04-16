package widget

import (
	"github.com/nanorele/gio/io/semantic"
	"github.com/nanorele/gio/layout"
)

type Bool struct {
	Value bool

	clk Clickable
}

func (b *Bool) Update(gtx layout.Context) bool {
	changed := false
	for b.clk.clicked(b, gtx) {
		b.Value = !b.Value
		changed = true
	}
	return changed
}

func (b *Bool) Hovered() bool {
	return b.clk.Hovered()
}

func (b *Bool) Pressed() bool {
	return b.clk.Pressed()
}

func (b *Bool) History() []Press {
	return b.clk.History()
}

func (b *Bool) Layout(gtx layout.Context, w layout.Widget) layout.Dimensions {
	b.Update(gtx)
	dims := b.clk.layout(b, gtx, func(gtx layout.Context) layout.Dimensions {
		semantic.SelectedOp(b.Value).Add(gtx.Ops)
		semantic.EnabledOp(gtx.Enabled()).Add(gtx.Ops)
		return w(gtx)
	})
	return dims
}
