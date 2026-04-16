package layout

import (
	"image"
	"math"

	"github.com/nanorele/gio/gesture"
	"github.com/nanorele/gio/io/pointer"
	"github.com/nanorele/gio/op"
	"github.com/nanorele/gio/op/clip"
)

type scrollChild struct {
	size image.Point
	call op.CallOp
}

type List struct {
	Axis Axis

	ScrollToEnd bool

	Alignment Alignment

	ScrollAnyAxis bool

	Gap int

	cs          Constraints
	scroll      gesture.Scroll
	scrollDelta int

	Position Position

	len int

	maxSize  int
	children []scrollChild
	dir      iterationDir
}

type ListElement func(gtx Context, index int) Dimensions

type iterationDir uint8

type Position struct {
	BeforeEnd bool

	First int

	Offset int

	OffsetLast int

	Count int

	Length int
}

const (
	iterateNone iterationDir = iota
	iterateForward
	iterateBackward
)

const inf = 1e6

func (l *List) init(gtx Context, len int) {
	if l.more() {
		panic("unfinished child")
	}
	l.cs = gtx.Constraints
	l.maxSize = 0
	l.children = l.children[:0]
	l.len = len
	l.update(gtx)
	if l.Position.First < 0 {
		l.Position.Offset = 0
		l.Position.First = 0
	}
	if l.scrollToEnd() || l.Position.First > len {
		l.Position.Offset = 0
		l.Position.First = len
	}
}

func (l *List) Layout(gtx Context, len int, w ListElement) Dimensions {
	l.init(gtx, len)
	crossMin, crossMax := l.Axis.crossConstraint(gtx.Constraints)
	gtx.Constraints = l.Axis.constraints(0, inf, crossMin, crossMax)
	macro := op.Record(gtx.Ops)
	laidOutTotalLength := 0
	numLaidOut := 0

	for l.next(); l.more(); l.next() {
		child := op.Record(gtx.Ops)
		dims := w(gtx, l.index())
		call := child.Stop()
		l.end(dims, call)
		laidOutTotalLength += l.Axis.Convert(dims.Size).X
		numLaidOut++
	}

	if numLaidOut > 0 {
		l.Position.Length = laidOutTotalLength*len/numLaidOut + l.Gap*(len-1)
	} else {
		l.Position.Length = 0
	}
	return l.layout(gtx.Ops, macro)
}

func (l *List) scrollToEnd() bool {
	return l.ScrollToEnd && !l.Position.BeforeEnd
}

func (l *List) Dragging() bool {
	return l.scroll.State() == gesture.StateDragging
}

func (l *List) update(gtx Context) {
	min, max := int(-inf), int(inf)
	if l.Position.First == 0 {

		min = -l.Position.Offset
		if min > 0 {
			min = 0
		}
	}
	if l.Position.First+l.Position.Count == l.len {
		max = -l.Position.OffsetLast
		if max < 0 {
			max = 0
		}
	}

	xrange := pointer.ScrollRange{Min: min, Max: max}
	yrange := pointer.ScrollRange{}

	axis := gesture.Axis(l.Axis)
	if l.ScrollAnyAxis {
		axis = gesture.Both
		yrange = xrange
	} else if l.Axis == Vertical {
		xrange, yrange = yrange, xrange
	}
	d := l.scroll.Update(gtx.Metric, gtx.Source, gtx.Now, axis, xrange, yrange)

	l.scrollDelta = d
	l.Position.Offset += d
}

func (l *List) next() {
	l.dir = l.nextDir()

	if l.scrollToEnd() && !l.more() && l.scrollDelta < 0 {
		l.Position.BeforeEnd = true
		l.Position.Offset += l.scrollDelta
		l.dir = l.nextDir()
	}
}

func (l *List) index() int {
	switch l.dir {
	case iterateBackward:
		return l.Position.First - 1
	case iterateForward:
		return l.Position.First + len(l.children)
	default:
		panic("Index called before Next")
	}
}

func (l *List) more() bool {
	return l.dir != iterateNone
}

func (l *List) nextDir() iterationDir {
	_, vsize := l.Axis.mainConstraint(l.cs)
	last := l.Position.First + len(l.children)

	if l.maxSize-l.Position.Offset < vsize && last == l.len {
		l.Position.Offset = l.maxSize - vsize
	}
	if l.Position.Offset < 0 && l.Position.First == 0 {
		l.Position.Offset = 0
	}

	firstSize, lastSize := 0, 0
	if len(l.children) > 0 {
		if l.Position.First > 0 {
			firstChild := l.children[0]
			firstSize = l.Axis.Convert(firstChild.size).X + l.Gap
		}
		if last < l.len {
			lastChild := l.children[len(l.children)-1]
			lastSize = l.Axis.Convert(lastChild.size).X + l.Gap
		}
	}
	switch {
	case len(l.children) == l.len:
		return iterateNone
	case l.maxSize-l.Position.Offset-lastSize < vsize:
		return iterateForward
	case l.Position.Offset-firstSize < 0:
		return iterateBackward
	}
	return iterateNone
}

func (l *List) end(dims Dimensions, call op.CallOp) {
	child := scrollChild{dims.Size, call}
	mainSize := l.Axis.Convert(child.size).X
	if len(l.children) > 0 {
		l.maxSize += l.Gap
	}
	l.maxSize += mainSize
	switch l.dir {
	case iterateForward:
		l.children = append(l.children, child)
	case iterateBackward:
		l.children = append(l.children, scrollChild{})
		copy(l.children[1:], l.children[:len(l.children)-1])
		l.children[0] = child
		l.Position.First--
		l.Position.Offset += mainSize + l.Gap
	default:
		panic("call Next before End")
	}
	l.dir = iterateNone
}

func (l *List) layout(ops *op.Ops, macro op.MacroOp) Dimensions {
	if l.more() {
		panic("unfinished child")
	}
	mainMin, mainMax := l.Axis.mainConstraint(l.cs)
	children := l.children
	var first scrollChild

	for len(children) > 0 {
		child := children[0]
		sz := child.size
		mainSize := l.Axis.Convert(sz).X
		if l.Position.Offset < mainSize {

			break
		}
		l.Position.First++
		l.Position.Offset -= mainSize + l.Gap
		first = child
		children = children[1:]
	}
	size := -l.Position.Offset
	var maxCross int
	var last scrollChild
	for i, child := range children {
		sz := l.Axis.Convert(child.size)
		if c := sz.Y; c > maxCross {
			maxCross = c
		}
		if i > 0 {
			size += l.Gap
		}
		size += sz.X
		if size >= mainMax {
			if i < len(children)-1 {
				last = children[i+1]
			}
			children = children[:i+1]
			break
		}
	}
	l.Position.Count = len(children)
	l.Position.OffsetLast = mainMax - size

	if space := l.Position.OffsetLast; l.ScrollToEnd && space > 0 {
		l.Position.Offset -= space
	}
	pos := -l.Position.Offset
	layout := func(child scrollChild) {
		sz := l.Axis.Convert(child.size)
		var cross int
		switch l.Alignment {
		case End:
			cross = maxCross - sz.Y
		case Middle:
			cross = (maxCross - sz.Y) / 2
		}
		childSize := sz.X
		pt := l.Axis.Convert(image.Pt(pos, cross))
		trans := op.Offset(pt).Push(ops)
		child.call.Add(ops)
		trans.Pop()
		pos += childSize
	}

	if first != (scrollChild{}) {
		sz := l.Axis.Convert(first.size)
		pos -= sz.X + l.Gap
		layout(first)
		pos += l.Gap
	}
	for i, child := range children {
		if i > 0 {
			pos += l.Gap
		}
		layout(child)
	}

	if last != (scrollChild{}) {
		pos += l.Gap
		layout(last)
	}
	atStart := l.Position.First == 0 && l.Position.Offset <= 0
	atEnd := l.Position.First+len(children) == l.len && mainMax >= pos
	if atStart && l.scrollDelta < 0 || atEnd && l.scrollDelta > 0 {
		l.scroll.Stop()
	}
	l.Position.BeforeEnd = !atEnd
	if pos < mainMin {
		pos = mainMin
	}
	if pos > mainMax {
		pos = mainMax
	}
	if crossMin, crossMax := l.Axis.crossConstraint(l.cs); maxCross < crossMin {
		maxCross = crossMin
	} else if maxCross > crossMax {
		maxCross = crossMax
	}
	dims := l.Axis.Convert(image.Pt(pos, maxCross))
	call := macro.Stop()
	defer clip.Rect(image.Rectangle{Max: dims}).Push(ops).Pop()

	l.scroll.Add(ops)

	call.Add(ops)
	return Dimensions{Size: dims}
}

func (l *List) ScrollBy(num float32) {

	i, f := math.Modf(float64(num))

	l.Position.First += int(i)

	itemHeight := float64(l.Position.Length) / float64(l.len)
	l.Position.Offset += int(math.Round(itemHeight * f))

	l.Position.BeforeEnd = true
}

func (l *List) ScrollTo(n int) {
	l.Position.First = n
	l.Position.Offset = 0
	l.Position.BeforeEnd = true
}
