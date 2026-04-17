package material

import (
	"image"
	"image/color"
	"math"

	"github.com/nanorele/gio/io/pointer"
	"github.com/nanorele/gio/layout"
	"github.com/nanorele/gio/op"
	"github.com/nanorele/gio/op/clip"
	"github.com/nanorele/gio/op/paint"
	"github.com/nanorele/gio/unit"
	"github.com/nanorele/gio/widget"
)

func fromListPosition(lp layout.Position, elements int, majorAxisSize int) (start, end float32) {
	if elements <= 0 || lp.Length <= 0 {
		return 0, 1
	}

	lengthEstPx := float32(lp.Length)
	elementLenEstPx := lengthEstPx / float32(elements)

	listOffsetF := float32(lp.Offset)
	listOffsetL := float32(lp.OffsetLast)

	viewportStart := clamp1((float32(lp.First)*elementLenEstPx + listOffsetF) / lengthEstPx)
	viewportEnd := clamp1((float32(lp.First+lp.Count)*elementLenEstPx + listOffsetL) / lengthEstPx)
	viewportFraction := viewportEnd - viewportStart

	visiblePx := float32(majorAxisSize)
	visibleFraction := visiblePx / lengthEstPx

	err := visibleFraction - viewportFraction
	adjStart := viewportStart
	adjEnd := viewportEnd
	if viewportFraction < 1 {
		startShare := viewportStart / (1 - viewportFraction)
		endShare := (1 - viewportEnd) / (1 - viewportFraction)
		startErr := startShare * err
		endErr := endShare * err

		adjStart -= startErr
		adjEnd += endErr
	}
	return adjStart, adjEnd
}

func rangeIsScrollable(start, end float32) bool {
	return end-start < 1
}

type ScrollTrackStyle struct {
	MajorPadding, MinorPadding unit.Dp

	Color color.NRGBA
}

type ScrollIndicatorStyle struct {
	MajorMinLen unit.Dp

	MinorWidth unit.Dp

	Color, HoverColor color.NRGBA

	CornerRadius unit.Dp
}

type ScrollbarStyle struct {
	Scrollbar *widget.Scrollbar
	Track     ScrollTrackStyle
	Indicator ScrollIndicatorStyle
}

func Scrollbar(th *Theme, state *widget.Scrollbar) ScrollbarStyle {
	lightFg := th.Palette.Fg
	lightFg.A = 150
	darkFg := lightFg
	darkFg.A = 200

	return ScrollbarStyle{
		Scrollbar: state,
		Track: ScrollTrackStyle{
			MajorPadding: 2,
			MinorPadding: 2,
		},
		Indicator: ScrollIndicatorStyle{
			MajorMinLen:  th.FingerSize,
			MinorWidth:   6,
			CornerRadius: 3,
			Color:        lightFg,
			HoverColor:   darkFg,
		},
	}
}

func (s ScrollbarStyle) Width() unit.Dp {
	return s.Indicator.MinorWidth + s.Track.MinorPadding + s.Track.MinorPadding
}

func (s ScrollbarStyle) Layout(gtx layout.Context, axis layout.Axis, viewportStart, viewportEnd float32) layout.Dimensions {
	if !rangeIsScrollable(viewportStart, viewportEnd) {
		return layout.Dimensions{}
	}

	convert := axis.Convert
	maxMajorAxis := convert(gtx.Constraints.Max).X
	gtx.Constraints.Min.X = maxMajorAxis
	gtx.Constraints.Min.Y = gtx.Dp(s.Width())
	gtx.Constraints.Min = convert(gtx.Constraints.Min)
	gtx.Constraints.Max = gtx.Constraints.Min

	s.Scrollbar.Update(gtx, axis, viewportStart, viewportEnd)

	if s.Scrollbar.IndicatorHovered() {
		s.Indicator.Color = s.Indicator.HoverColor
	}

	return s.layout(gtx, axis, viewportStart, viewportEnd)
}

func (s ScrollbarStyle) layout(gtx layout.Context, axis layout.Axis, viewportStart, viewportEnd float32) layout.Dimensions {
	inset := layout.Inset{
		Top:    s.Track.MajorPadding,
		Bottom: s.Track.MajorPadding,
		Left:   s.Track.MinorPadding,
		Right:  s.Track.MinorPadding,
	}
	if axis == layout.Horizontal {
		inset.Top, inset.Bottom, inset.Left, inset.Right = inset.Left, inset.Right, inset.Top, inset.Bottom
	}

	return layout.Background{}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {

			area := image.Rectangle{
				Max: gtx.Constraints.Min,
			}
			pointerArea := clip.Rect(area)
			defer pointerArea.Push(gtx.Ops).Pop()
			s.Scrollbar.AddDrag(gtx.Ops)

			defer pointer.PassOp{}.Push(gtx.Ops).Pop()
			defer pointerArea.Push(gtx.Ops).Pop()
			s.Scrollbar.AddTrack(gtx.Ops)

			paint.FillShape(gtx.Ops, s.Track.Color, clip.Rect(area).Op())
			return layout.Dimensions{Size: gtx.Constraints.Min}
		},
		func(gtx layout.Context) layout.Dimensions {
			return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {

				gtx.Constraints.Min = axis.Convert(gtx.Constraints.Min)
				gtx.Constraints.Max = axis.Convert(gtx.Constraints.Max)

				trackLen := gtx.Constraints.Min.X
				viewStart := int(math.Round(float64(viewportStart) * float64(trackLen)))
				viewEnd := int(math.Round(float64(viewportEnd) * float64(trackLen)))
				indicatorLen := max(viewEnd-viewStart, gtx.Dp(s.Indicator.MajorMinLen))
				if viewStart+indicatorLen > trackLen {
					viewStart = trackLen - indicatorLen
				}
				indicatorDims := axis.Convert(image.Point{
					X: indicatorLen,
					Y: gtx.Dp(s.Indicator.MinorWidth),
				})
				radius := gtx.Dp(s.Indicator.CornerRadius)

				offset := axis.Convert(image.Pt(viewStart, 0))
				defer op.Offset(offset).Push(gtx.Ops).Pop()
				paint.FillShape(gtx.Ops, s.Indicator.Color, clip.RRect{
					Rect: image.Rectangle{
						Max: indicatorDims,
					},
					SW: radius,
					NW: radius,
					NE: radius,
					SE: radius,
				}.Op(gtx.Ops))

				area := clip.Rect(image.Rectangle{Max: indicatorDims})
				defer pointer.PassOp{}.Push(gtx.Ops).Pop()
				defer area.Push(gtx.Ops).Pop()
				s.Scrollbar.AddIndicator(gtx.Ops)

				return layout.Dimensions{Size: axis.Convert(gtx.Constraints.Min)}
			})
		},
	)
}

type AnchorStrategy uint8

const (
	Occupy AnchorStrategy = iota

	Overlay
)

type ListStyle struct {
	state *widget.List
	ScrollbarStyle
	AnchorStrategy
}

func List(th *Theme, state *widget.List) ListStyle {
	return ListStyle{
		state:          state,
		ScrollbarStyle: Scrollbar(th, &state.Scrollbar),
	}
}

func (l ListStyle) Layout(gtx layout.Context, length int, w layout.ListElement) layout.Dimensions {
	originalConstraints := gtx.Constraints

	barWidth := gtx.Dp(l.Width())

	if l.AnchorStrategy == Occupy {

		max := l.state.Axis.Convert(gtx.Constraints.Max)
		min := l.state.Axis.Convert(gtx.Constraints.Min)
		max.Y -= barWidth
		if max.Y < 0 {
			max.Y = 0
		}
		min.Y -= barWidth
		if min.Y < 0 {
			min.Y = 0
		}
		gtx.Constraints.Max = l.state.Axis.Convert(max)
		gtx.Constraints.Min = l.state.Axis.Convert(min)
	}

	listDims := l.state.List.Layout(gtx, length, w)
	gtx.Constraints = originalConstraints

	anchoring := layout.E
	if l.state.Axis == layout.Horizontal {
		anchoring = layout.S
	}
	majorAxisSize := l.state.Axis.Convert(listDims.Size).X
	start, end := fromListPosition(l.state.Position, length, majorAxisSize)

	gtx.Constraints.Min = listDims.Size
	if l.AnchorStrategy == Occupy {
		min := l.state.Axis.Convert(gtx.Constraints.Min)
		min.Y += barWidth
		gtx.Constraints.Min = l.state.Axis.Convert(min)
	}
	anchoring.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return l.ScrollbarStyle.Layout(gtx, l.state.Axis, start, end)
	})

	if delta := l.state.ScrollDistance(); delta != 0 {

		l.state.List.ScrollBy(delta * float32(length))
	}

	if l.AnchorStrategy == Occupy {

		cross := l.state.Axis.Convert(listDims.Size)
		cross.Y += barWidth
		listDims.Size = l.state.Axis.Convert(cross)
	}

	return listDims
}

func (l ListStyle) LayoutWidgets(gtx layout.Context, widgets ...layout.Widget) layout.Dimensions {
	return l.Layout(gtx, len(widgets), func(gtx layout.Context, index int) layout.Dimensions {
		return widgets[index](gtx)
	})
}
