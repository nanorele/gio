package gesture

import (
	"image"
	"math"
	"time"

	"github.com/nanorele/gio/f32"
	"github.com/nanorele/gio/internal/fling"
	"github.com/nanorele/gio/io/event"
	"github.com/nanorele/gio/io/input"
	"github.com/nanorele/gio/io/key"
	"github.com/nanorele/gio/io/pointer"
	"github.com/nanorele/gio/op"
	"github.com/nanorele/gio/unit"
)

const doubleClickDuration = 200 * time.Millisecond

type Hover struct {
	entered bool

	pid pointer.ID
}

func (h *Hover) Add(ops *op.Ops) {
	event.Op(ops, h)
}

func (h *Hover) Update(q input.Source) bool {
	for {
		ev, ok := q.Event(pointer.Filter{
			Target: h,
			Kinds:  pointer.Enter | pointer.Leave | pointer.Cancel,
		})
		if !ok {
			break
		}
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		switch e.Kind {
		case pointer.Leave, pointer.Cancel:
			if h.entered && h.pid == e.PointerID {
				h.entered = false
			}
		case pointer.Enter:
			if !h.entered {
				h.pid = e.PointerID
			}
			if h.pid == e.PointerID {
				h.entered = true
			}
		}
	}
	return h.entered
}

type Click struct {
	clickedAt time.Duration

	clicks int

	pressed bool

	hovered bool

	entered bool

	pid pointer.ID
}

type ClickEvent struct {
	Kind      ClickKind
	Position  image.Point
	Source    pointer.Source
	Modifiers key.Modifiers

	NumClicks int
}

type ClickKind uint8

type Drag struct {
	dragging bool
	pressed  bool
	pid      pointer.ID
	start    f32.Point
}

type Scroll struct {
	dragging  bool
	estimator fling.Extrapolation
	flinger   fling.Animation
	pid       pointer.ID
	last      int

	scroll float32
}

type ScrollState uint8

type Axis uint8

const (
	Horizontal Axis = iota
	Vertical
	Both
)

const (
	KindPress ClickKind = iota

	KindClick

	KindCancel
)

const (
	StateIdle ScrollState = iota

	StateDragging

	StateFlinging
)

const touchSlop = unit.Dp(3)

func (c *Click) Add(ops *op.Ops) {
	event.Op(ops, c)
}

func (c *Click) Hovered() bool {
	return c.hovered
}

func (c *Click) Pressed() bool {
	return c.pressed
}

func (c *Click) Update(q input.Source) (ClickEvent, bool) {
	for {
		evt, ok := q.Event(pointer.Filter{
			Target: c,
			Kinds:  pointer.Press | pointer.Release | pointer.Enter | pointer.Leave | pointer.Cancel,
		})
		if !ok {
			break
		}
		e, ok := evt.(pointer.Event)
		if !ok {
			continue
		}
		switch e.Kind {
		case pointer.Release:
			if !c.pressed || c.pid != e.PointerID {
				break
			}
			c.pressed = false
			if !c.entered || c.hovered {
				return ClickEvent{
					Kind:      KindClick,
					Position:  e.Position.Round(),
					Source:    e.Source,
					Modifiers: e.Modifiers,
					NumClicks: c.clicks,
				}, true
			} else {
				return ClickEvent{Kind: KindCancel}, true
			}
		case pointer.Cancel:
			wasPressed := c.pressed
			c.pressed = false
			c.hovered = false
			c.entered = false
			if wasPressed {
				return ClickEvent{Kind: KindCancel}, true
			}
		case pointer.Press:
			if c.pressed {
				break
			}
			if e.Source == pointer.Mouse && e.Buttons != pointer.ButtonPrimary {
				break
			}
			if !c.hovered {
				c.pid = e.PointerID
			}
			if c.pid != e.PointerID {
				break
			}
			c.pressed = true
			if e.Time-c.clickedAt < doubleClickDuration {
				c.clicks++
			} else {
				c.clicks = 1
			}
			c.clickedAt = e.Time
			return ClickEvent{Kind: KindPress, Position: e.Position.Round(), Source: e.Source, Modifiers: e.Modifiers, NumClicks: c.clicks}, true
		case pointer.Leave:
			if !c.pressed {
				c.pid = e.PointerID
			}
			if c.pid == e.PointerID {
				c.hovered = false
			}
		case pointer.Enter:
			if !c.pressed {
				c.pid = e.PointerID
			}
			if c.pid == e.PointerID {
				c.hovered = true
				c.entered = true
			}
		}
	}
	return ClickEvent{}, false
}

func (ClickEvent) ImplementsEvent() {}

func (s *Scroll) Add(ops *op.Ops) {
	event.Op(ops, s)
}

func (s *Scroll) Stop() {
	s.flinger = fling.Animation{}
}

func (s *Scroll) Update(cfg unit.Metric, q input.Source, t time.Time, axis Axis, scrollx, scrolly pointer.ScrollRange) int {
	total := 0
	f := pointer.Filter{
		Target:  s,
		Kinds:   pointer.Press | pointer.Drag | pointer.Release | pointer.Scroll | pointer.Cancel,
		ScrollX: scrollx,
		ScrollY: scrolly,
	}
	for {
		evt, ok := q.Event(f)
		if !ok {
			break
		}
		e, ok := evt.(pointer.Event)
		if !ok {
			continue
		}
		switch e.Kind {
		case pointer.Press:
			if s.dragging {
				break
			}

			if e.Source != pointer.Touch {
				break
			}
			s.Stop()
			s.estimator = fling.Extrapolation{}
			v := s.val(axis, e.Position)
			s.last = int(math.Round(float64(v)))
			s.estimator.Sample(e.Time, v)
			s.dragging = true
			s.pid = e.PointerID
		case pointer.Release:
			if s.pid != e.PointerID {
				break
			}
			fling := s.estimator.Estimate()
			if slop, d := float32(cfg.Dp(touchSlop)), fling.Distance; d < -slop || d > slop {
				s.flinger.Start(cfg, t, fling.Velocity)
			}
			fallthrough
		case pointer.Cancel:
			s.dragging = false
		case pointer.Scroll:
			switch axis {
			case Horizontal:
				s.scroll += e.Scroll.X
			case Vertical:
				s.scroll += e.Scroll.Y
			case Both:
				s.scroll += e.Scroll.X + e.Scroll.Y
			}
			iscroll := int(s.scroll)
			s.scroll -= float32(iscroll)
			total += iscroll
		case pointer.Drag:
			if !s.dragging || s.pid != e.PointerID {
				continue
			}
			val := s.val(axis, e.Position)
			s.estimator.Sample(e.Time, val)
			v := int(math.Round(float64(val)))
			dist := s.last - v
			if e.Priority < pointer.Grabbed {
				slop := cfg.Dp(touchSlop)
				if dist := dist; dist >= slop || -slop >= dist {
					q.Execute(pointer.GrabCmd{Tag: s, ID: e.PointerID})
				}
			} else {
				s.last = v
				total += dist
			}
		}
	}
	total += s.flinger.Tick(t)
	if s.flinger.Active() {
		q.Execute(op.InvalidateCmd{})
	}
	return total
}

func (s *Scroll) val(axis Axis, p f32.Point) float32 {
	switch axis {
	case Horizontal:
		return p.X
	case Vertical:
		return p.Y
	case Both:
		return p.X + p.Y
	default:
		return 0
	}
}

func (s *Scroll) State() ScrollState {
	switch {
	case s.flinger.Active():
		return StateFlinging
	case s.dragging:
		return StateDragging
	default:
		return StateIdle
	}
}

func (d *Drag) Add(ops *op.Ops) {
	event.Op(ops, d)
}

func (d *Drag) Update(cfg unit.Metric, q input.Source, axis Axis) (pointer.Event, bool) {
	for {
		ev, ok := q.Event(pointer.Filter{
			Target: d,
			Kinds:  pointer.Press | pointer.Drag | pointer.Release | pointer.Cancel,
		})
		if !ok {
			break
		}
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}

		switch e.Kind {
		case pointer.Press:
			if !(e.Buttons == pointer.ButtonPrimary || e.Source == pointer.Touch) {
				continue
			}
			d.pressed = true
			if d.dragging {
				continue
			}
			d.dragging = true
			d.pid = e.PointerID
			d.start = e.Position
		case pointer.Drag:
			if !d.dragging || e.PointerID != d.pid {
				continue
			}
			switch axis {
			case Horizontal:
				e.Position.Y = d.start.Y
			case Vertical:
				e.Position.X = d.start.X
			case Both:

			}
			if e.Priority < pointer.Grabbed {
				diff := e.Position.Sub(d.start)
				slop := cfg.Dp(touchSlop)
				if diff.X*diff.X+diff.Y*diff.Y > float32(slop*slop) {
					q.Execute(pointer.GrabCmd{Tag: d, ID: e.PointerID})
				}
			}
		case pointer.Release, pointer.Cancel:
			d.pressed = false
			if !d.dragging || e.PointerID != d.pid {
				continue
			}
			d.dragging = false
		}

		return e, true
	}

	return pointer.Event{}, false
}

func (d *Drag) Dragging() bool { return d.dragging }

func (d *Drag) Pressed() bool { return d.pressed }

func (a Axis) String() string {
	switch a {
	case Horizontal:
		return "Horizontal"
	case Vertical:
		return "Vertical"
	default:
		panic("invalid Axis")
	}
}

func (ct ClickKind) String() string {
	switch ct {
	case KindPress:
		return "KindPress"
	case KindClick:
		return "KindClick"
	case KindCancel:
		return "KindCancel"
	default:
		panic("invalid ClickKind")
	}
}

func (s ScrollState) String() string {
	switch s {
	case StateIdle:
		return "StateIdle"
	case StateDragging:
		return "StateDragging"
	case StateFlinging:
		return "StateFlinging"
	default:
		panic("unreachable")
	}
}
