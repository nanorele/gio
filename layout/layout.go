package layout

import (
	"image"

	"github.com/nanorele/gio/f32"
	"github.com/nanorele/gio/op"
	"github.com/nanorele/gio/unit"
)

type Constraints struct {
	Min, Max image.Point
}

type Dimensions struct {
	Size     image.Point
	Baseline int
}

type Axis uint8

type Alignment uint8

type Direction uint8

type Widget func(gtx Context) Dimensions

const (
	Start Alignment = iota
	End
	Middle
	Baseline
)

const (
	NW Direction = iota
	N
	NE
	E
	SE
	S
	SW
	W
	Center
)

const (
	Horizontal Axis = iota
	Vertical
)

func Exact(size image.Point) Constraints {
	return Constraints{
		Min: size, Max: size,
	}
}

func FPt(p image.Point) f32.Point {
	return f32.Point{
		X: float32(p.X), Y: float32(p.Y),
	}
}

func (c Constraints) Constrain(size image.Point) image.Point {
	if min := c.Min.X; size.X < min {
		size.X = min
	}
	if min := c.Min.Y; size.Y < min {
		size.Y = min
	}
	if max := c.Max.X; size.X > max {
		size.X = max
	}
	if max := c.Max.Y; size.Y > max {
		size.Y = max
	}
	return size
}

func (c Constraints) AddMin(delta image.Point) Constraints {
	c.Min = c.Min.Add(delta)
	if c.Min.X < 0 {
		c.Min.X = 0
	}
	if c.Min.Y < 0 {
		c.Min.Y = 0
	}
	c.Min = c.Constrain(c.Min)
	return c
}

func (c Constraints) SubMax(delta image.Point) Constraints {
	c.Max = c.Max.Sub(delta)
	if c.Max.X < 0 {
		c.Max.X = 0
	}
	if c.Max.Y < 0 {
		c.Max.Y = 0
	}
	c.Min = c.Constrain(c.Min)
	return c
}

type Inset struct {
	Top, Bottom, Left, Right unit.Dp
}

func (in Inset) Layout(gtx Context, w Widget) Dimensions {
	top := gtx.Dp(in.Top)
	right := gtx.Dp(in.Right)
	bottom := gtx.Dp(in.Bottom)
	left := gtx.Dp(in.Left)
	mcs := gtx.Constraints
	mcs.Max.X -= left + right
	if mcs.Max.X < 0 {
		left = 0
		right = 0
		mcs.Max.X = 0
	}
	if mcs.Min.X > mcs.Max.X {
		mcs.Min.X = mcs.Max.X
	}
	mcs.Max.Y -= top + bottom
	if mcs.Max.Y < 0 {
		bottom = 0
		top = 0
		mcs.Max.Y = 0
	}
	if mcs.Min.Y > mcs.Max.Y {
		mcs.Min.Y = mcs.Max.Y
	}
	gtx.Constraints = mcs
	trans := op.Offset(image.Pt(left, top)).Push(gtx.Ops)
	dims := w(gtx)
	trans.Pop()
	return Dimensions{
		Size:     dims.Size.Add(image.Point{X: right + left, Y: top + bottom}),
		Baseline: dims.Baseline + bottom,
	}
}

func UniformInset(v unit.Dp) Inset {
	return Inset{Top: v, Right: v, Bottom: v, Left: v}
}

func (d Direction) Layout(gtx Context, w Widget) Dimensions {
	macro := op.Record(gtx.Ops)
	csn := gtx.Constraints.Min
	switch d {
	case N, S:
		gtx.Constraints.Min.Y = 0
	case E, W:
		gtx.Constraints.Min.X = 0
	default:
		gtx.Constraints.Min = image.Point{}
	}
	dims := w(gtx)
	call := macro.Stop()
	sz := dims.Size
	if sz.X < csn.X {
		sz.X = csn.X
	}
	if sz.Y < csn.Y {
		sz.Y = csn.Y
	}
	p := d.Position(dims.Size, sz)

	trans := op.Offset(p).Push(gtx.Ops)
	call.Add(gtx.Ops)
	trans.Pop()

	return Dimensions{
		Size:     sz,
		Baseline: dims.Baseline + sz.Y - dims.Size.Y - p.Y,
	}
}

func (d Direction) Position(widget, bounds image.Point) image.Point {
	var p image.Point

	switch d {
	case N, S, Center:
		p.X = (bounds.X - widget.X) / 2
	case NE, SE, E:
		p.X = bounds.X - widget.X
	}

	switch d {
	case W, Center, E:
		p.Y = (bounds.Y - widget.Y) / 2
	case SW, S, SE:
		p.Y = bounds.Y - widget.Y
	}

	return p
}

type Spacer struct {
	Width, Height unit.Dp
}

func (s Spacer) Layout(gtx Context) Dimensions {
	return Dimensions{
		Size: gtx.Constraints.Constrain(image.Point{
			X: gtx.Dp(s.Width),
			Y: gtx.Dp(s.Height),
		}),
	}
}

func (a Alignment) String() string {
	switch a {
	case Start:
		return "Start"
	case End:
		return "End"
	case Middle:
		return "Middle"
	case Baseline:
		return "Baseline"
	default:
		panic("unreachable")
	}
}

func (a Axis) Convert(pt image.Point) image.Point {
	if a == Horizontal {
		return pt
	}
	return image.Pt(pt.Y, pt.X)
}

func (a Axis) FConvert(pt f32.Point) f32.Point {
	if a == Horizontal {
		return pt
	}
	return f32.Pt(pt.Y, pt.X)
}

func (a Axis) mainConstraint(cs Constraints) (int, int) {
	if a == Horizontal {
		return cs.Min.X, cs.Max.X
	}
	return cs.Min.Y, cs.Max.Y
}

func (a Axis) crossConstraint(cs Constraints) (int, int) {
	if a == Horizontal {
		return cs.Min.Y, cs.Max.Y
	}
	return cs.Min.X, cs.Max.X
}

func (a Axis) constraints(mainMin, mainMax, crossMin, crossMax int) Constraints {
	if a == Horizontal {
		return Constraints{Min: image.Pt(mainMin, crossMin), Max: image.Pt(mainMax, crossMax)}
	}
	return Constraints{Min: image.Pt(crossMin, mainMin), Max: image.Pt(crossMax, mainMax)}
}

func (a Axis) String() string {
	switch a {
	case Horizontal:
		return "Horizontal"
	case Vertical:
		return "Vertical"
	default:
		panic("unreachable")
	}
}

func (d Direction) String() string {
	switch d {
	case NW:
		return "NW"
	case N:
		return "N"
	case NE:
		return "NE"
	case E:
		return "E"
	case SE:
		return "SE"
	case S:
		return "S"
	case SW:
		return "SW"
	case W:
		return "W"
	case Center:
		return "Center"
	default:
		panic("unreachable")
	}
}
