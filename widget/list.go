package widget

import (
	"image"

	"github.com/nanorele/gio/gesture"
	"github.com/nanorele/gio/io/key"
	"github.com/nanorele/gio/io/pointer"
	"github.com/nanorele/gio/layout"
	"github.com/nanorele/gio/op"
)

type Scrollbar struct {
	track, indicator gesture.Click
	drag             gesture.Drag
	delta            float32

	dragging   bool
	oldDragPos float32
}

func (s *Scrollbar) Update(gtx layout.Context, axis layout.Axis, viewportStart, viewportEnd float32) {

	trackHeight := float32(axis.Convert(gtx.Constraints.Max).X)
	s.delta = 0

	centerOnClick := func(normalizedPos float32) {

		target := normalizedPos - (viewportEnd-viewportStart)/2
		s.delta += target - viewportStart
		if s.delta < -viewportStart {
			s.delta = -viewportStart
		} else if s.delta > 1-viewportEnd {
			s.delta = 1 - viewportEnd
		}
	}

	for {
		event, ok := s.track.Update(gtx.Source)
		if !ok {
			break
		}
		if event.Kind != gesture.KindClick ||
			event.Modifiers != key.Modifiers(0) ||
			event.NumClicks > 1 {
			continue
		}
		pos := axis.Convert(image.Point{
			X: int(event.Position.X),
			Y: int(event.Position.Y),
		})
		normalizedPos := float32(pos.X) / trackHeight

		if !(normalizedPos >= viewportStart && normalizedPos <= viewportEnd) {
			centerOnClick(normalizedPos)
		}
	}

	for {
		event, ok := s.drag.Update(gtx.Metric, gtx.Source, gesture.Axis(axis))
		if !ok {
			break
		}
		switch event.Kind {
		case pointer.Drag:
		case pointer.Release, pointer.Cancel:
			s.dragging = false
			continue
		default:
			continue
		}
		dragOffset := axis.FConvert(event.Position).X

		if dragOffset < 0 {
			dragOffset = 0
		} else if dragOffset > trackHeight {
			dragOffset = trackHeight
		}
		normalizedDragOffset := dragOffset / trackHeight

		if !s.dragging {
			s.dragging = true
			s.oldDragPos = normalizedDragOffset

			if normalizedDragOffset < viewportStart || normalizedDragOffset > viewportEnd {

				pos := axis.Convert(image.Point{
					X: int(event.Position.X),
					Y: int(event.Position.Y),
				})
				normalizedPos := float32(pos.X) / trackHeight
				centerOnClick(normalizedPos)
			}
		} else {
			s.delta += normalizedDragOffset - s.oldDragPos

			if viewportStart+s.delta < 0 {

				normalizedDragOffset -= viewportStart + s.delta

				s.delta = -viewportStart
			} else if viewportEnd+s.delta > 1 {
				normalizedDragOffset += (1 - viewportEnd) - s.delta
				s.delta = 1 - viewportEnd
			}
			s.oldDragPos = normalizedDragOffset
		}
	}

	for {
		if _, ok := s.indicator.Update(gtx.Source); !ok {
			break
		}
	}
}

func (s *Scrollbar) AddTrack(ops *op.Ops) {
	s.track.Add(ops)
}

func (s *Scrollbar) AddIndicator(ops *op.Ops) {
	s.indicator.Add(ops)
}

func (s *Scrollbar) AddDrag(ops *op.Ops) {
	s.drag.Add(ops)
}

func (s *Scrollbar) IndicatorHovered() bool {
	return s.indicator.Hovered()
}

func (s *Scrollbar) TrackHovered() bool {
	return s.track.Hovered()
}

func (s *Scrollbar) ScrollDistance() float32 {
	return s.delta
}

func (s *Scrollbar) Dragging() bool {
	return s.dragging
}

type List struct {
	Scrollbar
	layout.List
}
