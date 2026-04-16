package widget

import (
	"image"
	"time"

	"github.com/nanorele/gio/gesture"
	"github.com/nanorele/gio/io/event"
	"github.com/nanorele/gio/io/key"
	"github.com/nanorele/gio/io/semantic"
	"github.com/nanorele/gio/layout"
	"github.com/nanorele/gio/op"
	"github.com/nanorele/gio/op/clip"
)

type Clickable struct {
	click   gesture.Click
	history []Press

	requestClicks int
	pressedKey    key.Name
}

type Click struct {
	Modifiers key.Modifiers
	NumClicks int
}

type Press struct {
	Position image.Point

	Start time.Time

	End time.Time

	Cancelled bool
}

func (b *Clickable) Click() {
	b.requestClicks++
}

func (b *Clickable) Clicked(gtx layout.Context) bool {
	return b.clicked(b, gtx)
}

func (b *Clickable) clicked(t event.Tag, gtx layout.Context) bool {
	_, clicked := b.update(t, gtx)
	return clicked
}

func (b *Clickable) Hovered() bool {
	return b.click.Hovered()
}

func (b *Clickable) Pressed() bool {
	return b.click.Pressed()
}

func (b *Clickable) History() []Press {
	return b.history
}

func (b *Clickable) Layout(gtx layout.Context, w layout.Widget) layout.Dimensions {
	return b.layout(b, gtx, w)
}

func (b *Clickable) layout(t event.Tag, gtx layout.Context, w layout.Widget) layout.Dimensions {
	for {
		_, ok := b.update(t, gtx)
		if !ok {
			break
		}
	}
	m := op.Record(gtx.Ops)
	dims := w(gtx)
	c := m.Stop()
	defer clip.Rect(image.Rectangle{Max: dims.Size}).Push(gtx.Ops).Pop()
	semantic.EnabledOp(gtx.Enabled()).Add(gtx.Ops)
	b.click.Add(gtx.Ops)
	event.Op(gtx.Ops, t)
	c.Add(gtx.Ops)
	return dims
}

func (b *Clickable) Update(gtx layout.Context) (Click, bool) {
	return b.update(b, gtx)
}

func (b *Clickable) update(t event.Tag, gtx layout.Context) (Click, bool) {
	for len(b.history) > 0 {
		c := b.history[0]
		if c.End.IsZero() || gtx.Now.Sub(c.End) < 1*time.Second {
			break
		}
		n := copy(b.history, b.history[1:])
		b.history = b.history[:n]
	}
	if c := b.requestClicks; c > 0 {
		b.requestClicks = 0
		return Click{
			NumClicks: c,
		}, true
	}
	for {
		e, ok := b.click.Update(gtx.Source)
		if !ok {
			break
		}
		switch e.Kind {
		case gesture.KindClick:
			if l := len(b.history); l > 0 {
				b.history[l-1].End = gtx.Now
			}
			return Click{
				Modifiers: e.Modifiers,
				NumClicks: e.NumClicks,
			}, true
		case gesture.KindCancel:
			for i := range b.history {
				b.history[i].Cancelled = true
				if b.history[i].End.IsZero() {
					b.history[i].End = gtx.Now
				}
			}
		case gesture.KindPress:
			b.history = append(b.history, Press{
				Position: e.Position,
				Start:    gtx.Now,
			})
		}
	}
	for {
		e, ok := gtx.Event(
			key.FocusFilter{Target: t},
			key.Filter{Focus: t, Name: key.NameReturn},
			key.Filter{Focus: t, Name: key.NameSpace},
		)
		if !ok {
			break
		}
		switch e := e.(type) {
		case key.FocusEvent:
			if e.Focus {
				b.pressedKey = ""
			}
		case key.Event:
			if !gtx.Focused(t) {
				break
			}
			if e.Name != key.NameReturn && e.Name != key.NameSpace {
				break
			}
			switch e.State {
			case key.Press:
				b.pressedKey = e.Name
			case key.Release:
				if b.pressedKey != e.Name {
					break
				}

				b.pressedKey = ""
				return Click{
					Modifiers: e.Modifiers,
					NumClicks: 1,
				}, true
			}
		}
	}
	return Click{}, false
}
