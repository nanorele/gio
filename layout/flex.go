package layout

import (
	"image"

	"github.com/nanorele/gio/op"
)

type Flex struct {
	Axis Axis

	Spacing Spacing

	Alignment Alignment

	WeightSum float32

	Gap int
}

type FlexChild struct {
	flex   bool
	weight float32

	widget Widget
}

type Spacing uint8

const (
	SpaceEnd Spacing = iota

	SpaceStart

	SpaceSides

	SpaceAround

	SpaceBetween

	SpaceEvenly
)

func Rigid(widget Widget) FlexChild {
	return FlexChild{
		widget: widget,
	}
}

func Flexed(weight float32, widget Widget) FlexChild {
	return FlexChild{
		flex:   true,
		weight: weight,
		widget: widget,
	}
}

func (f Flex) Layout(gtx Context, children ...FlexChild) Dimensions {
	size := 0
	cs := gtx.Constraints
	mainMin, mainMax := f.Axis.mainConstraint(cs)
	crossMin, crossMax := f.Axis.crossConstraint(cs)
	remaining := mainMax

	if len(children) > 1 && f.Gap > 0 {
		totalGap := f.Gap * (len(children) - 1)
		remaining -= totalGap
		if remaining < 0 {
			remaining = 0
		}
	}
	var totalWeight float32
	cgtx := gtx

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

	for i, child := range children {
		if child.flex {
			totalWeight += child.weight
			continue
		}
		macro := op.Record(gtx.Ops)
		cgtx.Constraints = f.Axis.constraints(0, remaining, crossMin, crossMax)
		dims := child.widget(cgtx)
		c := macro.Stop()
		sz := f.Axis.Convert(dims.Size).X
		size += sz
		remaining -= sz
		if remaining < 0 {
			remaining = 0
		}
		scratch[i].call = c
		scratch[i].dims = dims
	}
	if w := f.WeightSum; w != 0 {
		totalWeight = w
	}

	var fraction float32
	flexTotal := remaining

	for i, child := range children {
		if !child.flex {
			continue
		}
		var flexSize int
		if remaining > 0 && totalWeight > 0 {

			childSize := float32(flexTotal) * child.weight / totalWeight
			flexSize = int(childSize + fraction + .5)
			fraction = childSize - float32(flexSize)
			if flexSize > remaining {
				flexSize = remaining
			}
		}
		macro := op.Record(gtx.Ops)
		cgtx.Constraints = f.Axis.constraints(flexSize, flexSize, crossMin, crossMax)
		dims := child.widget(cgtx)
		c := macro.Stop()
		sz := f.Axis.Convert(dims.Size).X
		size += sz
		remaining -= sz
		if remaining < 0 {
			remaining = 0
		}
		scratch[i].call = c
		scratch[i].dims = dims
	}
	maxCross := crossMin
	var maxBaseline int
	for _, scratchChild := range scratch {
		if c := f.Axis.Convert(scratchChild.dims.Size).Y; c > maxCross {
			maxCross = c
		}
		if b := scratchChild.dims.Size.Y - scratchChild.dims.Baseline; b > maxBaseline {
			maxBaseline = b
		}
	}
	if len(children) > 1 && f.Gap > 0 {
		size += f.Gap * (len(children) - 1)
	}
	var space int
	if mainMin > size {
		space = mainMin - size
	}
	var mainSize int
	switch f.Spacing {
	case SpaceSides:
		mainSize += space / 2
	case SpaceStart:
		mainSize += space
	case SpaceEvenly:
		mainSize += space / (1 + len(children))
	case SpaceAround:
		if len(children) > 0 {
			mainSize += space / (len(children) * 2)
		}
	}
	for i, scratchChild := range scratch {
		dims := scratchChild.dims
		b := dims.Size.Y - dims.Baseline
		var cross int
		switch f.Alignment {
		case End:
			cross = maxCross - f.Axis.Convert(dims.Size).Y
		case Middle:
			cross = (maxCross - f.Axis.Convert(dims.Size).Y) / 2
		case Baseline:
			if f.Axis == Horizontal {
				cross = maxBaseline - b
			}
		}
		pt := f.Axis.Convert(image.Pt(mainSize, cross))
		trans := op.Offset(pt).Push(gtx.Ops)
		scratchChild.call.Add(gtx.Ops)
		trans.Pop()
		mainSize += f.Axis.Convert(dims.Size).X
		if i < len(children)-1 {
			mainSize += f.Gap
			switch f.Spacing {
			case SpaceEvenly:
				mainSize += space / (1 + len(children))
			case SpaceAround:
				if len(children) > 0 {
					mainSize += space / len(children)
				}
			case SpaceBetween:
				if len(children) > 1 {
					mainSize += space / (len(children) - 1)
				}
			}
		}
	}
	switch f.Spacing {
	case SpaceSides:
		mainSize += space / 2
	case SpaceEnd:
		mainSize += space
	case SpaceEvenly:
		mainSize += space / (1 + len(children))
	case SpaceAround:
		if len(children) > 0 {
			mainSize += space / (len(children) * 2)
		}
	}
	sz := f.Axis.Convert(image.Pt(mainSize, maxCross))
	sz = cs.Constrain(sz)
	return Dimensions{Size: sz, Baseline: sz.Y - maxBaseline}
}

func (s Spacing) String() string {
	switch s {
	case SpaceEnd:
		return "SpaceEnd"
	case SpaceStart:
		return "SpaceStart"
	case SpaceSides:
		return "SpaceSides"
	case SpaceAround:
		return "SpaceAround"
	case SpaceBetween:
		return "SpaceBetween"
	case SpaceEvenly:
		return "SpaceEvenly"
	default:
		panic("unreachable")
	}
}
