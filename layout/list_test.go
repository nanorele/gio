package layout

import (
	"image"
	"testing"

	"github.com/nanorele/gio/f32"
	"github.com/nanorele/gio/io/event"
	"github.com/nanorele/gio/io/input"
	"github.com/nanorele/gio/io/pointer"
	"github.com/nanorele/gio/op"
)

func TestListPositionExtremes(t *testing.T) {
	var l List
	gtx := Context{
		Ops:         new(op.Ops),
		Constraints: Exact(image.Pt(20, 10)),
	}
	const n = 3
	layout := func(_ Context, idx int) Dimensions {
		if idx < 0 || idx >= n {
			t.Errorf("list index %d out of bounds [0;%d]", idx, n-1)
		}
		return Dimensions{}
	}
	l.Position.First = -1
	l.Layout(gtx, n, layout)
	l.Position.First = n + 1
	l.Layout(gtx, n, layout)
}

func TestEmptyList(t *testing.T) {
	var l List
	gtx := Context{
		Ops:         new(op.Ops),
		Constraints: Exact(image.Pt(20, 10)),
	}
	dims := l.Layout(gtx, 0, nil)
	if got, want := dims.Size, gtx.Constraints.Min; got != want {
		t.Errorf("got %v; want %v", got, want)
	}
}

func TestListScrollToEnd(t *testing.T) {
	l := List{
		ScrollToEnd: true,
	}
	gtx := Context{
		Ops:         new(op.Ops),
		Constraints: Exact(image.Pt(20, 10)),
	}
	l.Layout(gtx, 1, func(gtx Context, idx int) Dimensions {
		return Dimensions{
			Size: image.Pt(10, 10),
		}
	})
	if want, got := -10, l.Position.Offset; want != got {
		t.Errorf("got offset %d, want %d", got, want)
	}
}

func TestListScroll(t *testing.T) {
	var l List
	gtx := Context{
		Ops:         new(op.Ops),
		Constraints: Exact(image.Pt(20, 10)),
	}
	el := func(gtx Context, idx int) Dimensions {
		return Dimensions{Size: image.Pt(10, 10)}
	}
	l.ScrollTo(1)
	l.Layout(gtx, 5, el)
	if l.Position.First != 1 {
		t.Errorf("ScrollTo(1) failed, first is %d", l.Position.First)
	}

	l.ScrollBy(1.5) // scroll by 1.5 items? No, ScrollBy is float items.
	l.Layout(gtx, 5, el)
	// ScrollBy(1.5) from first=1 should be first=2, offset=5 (since width=10)
	if l.Position.First != 2 || l.Position.Offset != 5 {
		t.Errorf("ScrollBy(1.5) failed: First=%d, Offset=%d", l.Position.First, l.Position.Offset)
	}

	if l.Dragging() {
		t.Error("list should not be dragging")
	}
}

func TestListPosition(t *testing.T) {
	_s := func(e ...event.Event) []event.Event { return e }
	r := new(input.Router)
	gtx := Context{
		Ops: new(op.Ops),
		Constraints: Constraints{
			Max: image.Pt(20, 10),
		},
		Source: r.Source(),
	}
	el := func(gtx Context, idx int) Dimensions {
		return Dimensions{Size: image.Pt(10, 10)}
	}
	for _, tc := range []struct {
		label  string
		num    int
		scroll []event.Event
		first  int
		count  int
		offset int
		last   int
	}{
		{label: "no item", last: 20},
		{label: "1 visible 0 hidden", num: 1, count: 1, last: 10},
		{label: "2 visible 0 hidden", num: 2, count: 2},
		{label: "2 visible 1 hidden", num: 3, count: 2},
		{
			label: "3 visible 0 hidden small scroll", num: 3, count: 3, offset: 5, last: -5,
			scroll: _s(
				pointer.Event{
					Source:   pointer.Mouse,
					Buttons:  pointer.ButtonPrimary,
					Kind:     pointer.Press,
					Position: f32.Pt(0, 0),
				},
				pointer.Event{
					Source: pointer.Mouse,
					Kind:   pointer.Scroll,
					Scroll: f32.Pt(5, 0),
				},
				pointer.Event{
					Source:   pointer.Mouse,
					Buttons:  pointer.ButtonPrimary,
					Kind:     pointer.Release,
					Position: f32.Pt(5, 0),
				},
			),
		},
		{
			label: "3 visible 0 hidden small scroll 2", num: 3, count: 3, offset: 3, last: -7,
			scroll: _s(
				pointer.Event{
					Source:   pointer.Mouse,
					Buttons:  pointer.ButtonPrimary,
					Kind:     pointer.Press,
					Position: f32.Pt(0, 0),
				},
				pointer.Event{
					Source: pointer.Mouse,
					Kind:   pointer.Scroll,
					Scroll: f32.Pt(3, 0),
				},
				pointer.Event{
					Source:   pointer.Mouse,
					Buttons:  pointer.ButtonPrimary,
					Kind:     pointer.Release,
					Position: f32.Pt(5, 0),
				},
			),
		},
		{
			label: "2 visible 1 hidden large scroll", num: 3, count: 2, first: 1,
			scroll: _s(
				pointer.Event{
					Source:   pointer.Mouse,
					Buttons:  pointer.ButtonPrimary,
					Kind:     pointer.Press,
					Position: f32.Pt(0, 0),
				},
				pointer.Event{
					Source: pointer.Mouse,
					Kind:   pointer.Scroll,
					Scroll: f32.Pt(10, 0),
				},
				pointer.Event{
					Source:   pointer.Mouse,
					Buttons:  pointer.ButtonPrimary,
					Kind:     pointer.Release,
					Position: f32.Pt(15, 0),
				},
			),
		},
	} {
		t.Run(tc.label, func(t *testing.T) {
			gtx.Ops.Reset()

			var list List

			list.Layout(gtx, tc.num, el)

			r.Frame(gtx.Ops)
			r.Queue(tc.scroll...)

			list.Layout(gtx, tc.num, el)

			pos := list.Position
			if got, want := pos.First, tc.first; got != want {
				t.Errorf("List: invalid first position: got %v; want %v", got, want)
			}
			if got, want := pos.Count, tc.count; got != want {
				t.Errorf("List: invalid number of visible children: got %v; want %v", got, want)
			}
			if got, want := pos.Offset, tc.offset; got != want {
				t.Errorf("List: invalid first visible offset: got %v; want %v", got, want)
			}
			if got, want := pos.OffsetLast, tc.last; got != want {
				t.Errorf("List: invalid last visible offset: got %v; want %v", got, want)
			}
		})
	}
}

func TestListGap(t *testing.T) {
	gtx := Context{
		Ops: new(op.Ops),
		Constraints: Constraints{
			Max: image.Pt(100, 20),
		},
	}

	l := List{Gap: 5}
	dims := l.Layout(gtx, 2, func(gtx Context, idx int) Dimensions {
		return Dimensions{Size: image.Pt(10, 10)}
	})
	if got, exp := dims.Size.X, 25; got != exp {
		t.Errorf("two children with gap: got width %d, expected %d", got, exp)
	}

	l = List{Gap: 5}
	dims = l.Layout(gtx, 3, func(gtx Context, idx int) Dimensions {
		return Dimensions{Size: image.Pt(10, 10)}
	})
	if got, exp := dims.Size.X, 40; got != exp {
		t.Errorf("three children with gap: got width %d, expected %d", got, exp)
	}

	l = List{Gap: 5}
	dims = l.Layout(gtx, 1, func(gtx Context, idx int) Dimensions {
		return Dimensions{Size: image.Pt(10, 10)}
	})
	if got, exp := dims.Size.X, 10; got != exp {
		t.Errorf("single child with gap: got width %d, expected %d", got, exp)
	}

	l = List{Gap: 5}
	dims = l.Layout(gtx, 0, nil)
	if got, exp := dims.Size.X, 0; got != exp {
		t.Errorf("no children with gap: got width %d, expected %d", got, exp)
	}
}

func TestListGapVertical(t *testing.T) {
	gtx := Context{
		Ops: new(op.Ops),
		Constraints: Constraints{
			Max: image.Pt(20, 100),
		},
	}

	l := List{Axis: Vertical, Gap: 10}
	dims := l.Layout(gtx, 3, func(gtx Context, idx int) Dimensions {
		return Dimensions{Size: image.Pt(10, 15)}
	})

	if got, exp := dims.Size.Y, 65; got != exp {
		t.Errorf("vertical list with gap: got height %d, expected %d", got, exp)
	}
}

func TestListGapPosition(t *testing.T) {
	gtx := Context{
		Ops: new(op.Ops),
		Constraints: Constraints{
			Max: image.Pt(30, 20),
		},
	}

	l := List{Gap: 5}
	l.Layout(gtx, 5, func(gtx Context, idx int) Dimensions {
		return Dimensions{Size: image.Pt(10, 10)}
	})
	if got, exp := l.Position.Count, 3; got != exp {
		t.Errorf("visible count with gap: got %d, expected %d", got, exp)
	}
	if got, exp := l.Position.First, 0; got != exp {
		t.Errorf("first with gap: got %d, expected %d", got, exp)
	}

	if got, exp := l.Position.OffsetLast, -10; got != exp {
		t.Errorf("offset last with gap: got %d, expected %d", got, exp)
	}
}

func TestExtraChildren(t *testing.T) {
	var l List
	l.Position.First = 1
	gtx := Context{
		Ops:         new(op.Ops),
		Constraints: Exact(image.Pt(10, 10)),
	}
	count := 0
	const all = 3
	l.Layout(gtx, all, func(gtx Context, idx int) Dimensions {
		count++
		return Dimensions{Size: image.Pt(10, 10)}
	})
	if count != all {
		t.Errorf("laid out %d of %d children", count, all)
	}
}

func TestListScrollToFirst(t *testing.T) {
	var l List
	l.ScrollTo(0)
	if l.Position.First != 0 || l.Position.Offset != 0 || !l.Position.BeforeEnd {
		t.Errorf("ScrollTo(0): %+v", l.Position)
	}
}

func TestListScrollToMiddle(t *testing.T) {
	var l List
	l.ScrollTo(5)
	if l.Position.First != 5 || l.Position.Offset != 0 {
		t.Errorf("ScrollTo(5): %+v", l.Position)
	}
}

func TestListScrollByZero(t *testing.T) {
	// ScrollBy on a fresh list with l.len==0 must not panic (regression).
	var l List
	l.ScrollBy(2)
	if l.Position.First != 2 {
		t.Errorf("ScrollBy(2): First=%d, want 2", l.Position.First)
	}
	// Offset should be unchanged (no item height available).
	if l.Position.Offset != 0 {
		t.Errorf("ScrollBy(2) on empty len: Offset=%d, want 0", l.Position.Offset)
	}
}

func TestListScrollByNegative(t *testing.T) {
	var l List
	l.ScrollBy(-3)
	if l.Position.First != -3 {
		t.Errorf("ScrollBy(-3): First=%d, want -3", l.Position.First)
	}
}

func TestListScrollByFractional(t *testing.T) {
	var l List
	gtx := Context{
		Ops:         new(op.Ops),
		Constraints: Exact(image.Pt(20, 10)),
	}
	el := func(gtx Context, idx int) Dimensions {
		return Dimensions{Size: image.Pt(10, 10)}
	}
	l.Layout(gtx, 5, el)
	// Length is approximate but item-width=10. Now scroll by 0.5 items.
	l.ScrollBy(0.5)
	// Offset should advance by ~5.
	if l.Position.Offset < 4 || l.Position.Offset > 6 {
		t.Errorf("ScrollBy(0.5) offset: got %d, want ~5", l.Position.Offset)
	}
}

func TestListPositionFields(t *testing.T) {
	var l List
	gtx := Context{
		Ops:         new(op.Ops),
		Constraints: Exact(image.Pt(30, 10)),
	}
	el := func(gtx Context, idx int) Dimensions {
		return Dimensions{Size: image.Pt(10, 10)}
	}
	l.Layout(gtx, 5, el)
	if l.Position.Count != 3 {
		t.Errorf("Count: got %d, want 3", l.Position.Count)
	}
	if l.Position.First != 0 {
		t.Errorf("First: got %d, want 0", l.Position.First)
	}
	if l.Position.Offset != 0 {
		t.Errorf("Offset: got %d, want 0", l.Position.Offset)
	}
	if l.Position.OffsetLast != 0 {
		t.Errorf("OffsetLast: got %d, want 0", l.Position.OffsetLast)
	}
	if !l.Position.BeforeEnd {
		t.Errorf("BeforeEnd should be true (more items remain)")
	}
}

func TestListAllVisibleAtEnd(t *testing.T) {
	// When everything fits, BeforeEnd should be false.
	var l List
	gtx := Context{
		Ops:         new(op.Ops),
		Constraints: Exact(image.Pt(100, 10)),
	}
	el := func(gtx Context, idx int) Dimensions {
		return Dimensions{Size: image.Pt(10, 10)}
	}
	l.Layout(gtx, 3, el)
	if l.Position.Count != 3 {
		t.Errorf("Count: got %d, want 3", l.Position.Count)
	}
	if l.Position.BeforeEnd {
		t.Errorf("BeforeEnd should be false when all items visible")
	}
}

func TestListScrollPastFirst(t *testing.T) {
	// Position.First = -1 should be normalized to 0 in init().
	var l List
	l.Position.First = -10
	gtx := Context{
		Ops:         new(op.Ops),
		Constraints: Exact(image.Pt(20, 10)),
	}
	el := func(gtx Context, idx int) Dimensions {
		if idx < 0 {
			t.Fatalf("negative index %d", idx)
		}
		return Dimensions{Size: image.Pt(10, 10)}
	}
	l.Layout(gtx, 3, el)
	if l.Position.First != 0 {
		t.Errorf("First not normalized: got %d", l.Position.First)
	}
}

func TestListScrollPastLast(t *testing.T) {
	var l List
	l.Position.First = 100
	gtx := Context{
		Ops:         new(op.Ops),
		Constraints: Exact(image.Pt(20, 10)),
	}
	el := func(gtx Context, idx int) Dimensions {
		if idx >= 3 {
			t.Fatalf("index %d >= len 3", idx)
		}
		return Dimensions{Size: image.Pt(10, 10)}
	}
	l.Layout(gtx, 3, el)
}

func TestListVerticalLayout(t *testing.T) {
	l := List{Axis: Vertical}
	gtx := Context{
		Ops:         new(op.Ops),
		Constraints: Exact(image.Pt(10, 30)),
	}
	el := func(gtx Context, idx int) Dimensions {
		return Dimensions{Size: image.Pt(10, 10)}
	}
	l.Layout(gtx, 5, el)
	if l.Position.Count != 3 {
		t.Errorf("vertical count: got %d, want 3", l.Position.Count)
	}
}

func TestListAlignmentMiddleVertical(t *testing.T) {
	// Cross alignment shouldn't crash with varying child sizes.
	l := List{Axis: Vertical, Alignment: Middle}
	gtx := Context{
		Ops:         new(op.Ops),
		Constraints: Constraints{Max: image.Pt(50, 100)},
	}
	l.Layout(gtx, 3, func(gtx Context, idx int) Dimensions {
		return Dimensions{Size: image.Pt(10+idx*5, 10)}
	})
	if l.Position.Count != 3 {
		t.Errorf("count: %d", l.Position.Count)
	}
}

func TestListDraggingFalse(t *testing.T) {
	var l List
	if l.Dragging() {
		t.Errorf("fresh list should not be dragging")
	}
}

func TestListScrollToEndManyItems(t *testing.T) {
	l := List{ScrollToEnd: true}
	gtx := Context{
		Ops:         new(op.Ops),
		Constraints: Exact(image.Pt(20, 10)),
	}
	el := func(gtx Context, idx int) Dimensions {
		return Dimensions{Size: image.Pt(10, 10)}
	}
	l.Layout(gtx, 5, el)
	// First should advance to show last items.
	if l.Position.First+l.Position.Count != 5 {
		t.Errorf("ScrollToEnd: First=%d, Count=%d, sum=%d, want 5",
			l.Position.First, l.Position.Count, l.Position.First+l.Position.Count)
	}
}

func TestListReuseAcrossLayouts(t *testing.T) {
	// Subsequent layouts should not duplicate children.
	var l List
	gtx := Context{
		Ops:         new(op.Ops),
		Constraints: Exact(image.Pt(20, 10)),
	}
	el := func(gtx Context, idx int) Dimensions {
		return Dimensions{Size: image.Pt(10, 10)}
	}
	for i := 0; i < 3; i++ {
		gtx.Ops.Reset()
		l.Layout(gtx, 5, el)
		if l.Position.Count != 2 {
			t.Errorf("iteration %d: Count=%d, want 2", i, l.Position.Count)
		}
	}
}

func TestListScrollByItemAdvancesFirst(t *testing.T) {
	var l List
	gtx := Context{
		Ops:         new(op.Ops),
		Constraints: Exact(image.Pt(20, 10)),
	}
	el := func(gtx Context, idx int) Dimensions {
		return Dimensions{Size: image.Pt(10, 10)}
	}
	l.Layout(gtx, 10, el)
	l.ScrollBy(2)
	l.Layout(gtx, 10, el)
	if l.Position.First != 2 {
		t.Errorf("ScrollBy(2): First=%d, want 2", l.Position.First)
	}
}

// TestListVisibleRange verifies that VisibleRange's prediction covers the
// range Layout actually lays out, for uniform items across scroll positions,
// and that Layout still consumes scroll events when VisibleRange ran first.
func TestListVisibleRange(t *testing.T) {
	const (
		n        = 100
		itemH    = 10
		viewport = 45
	)
	el := func(gtx Context, idx int) Dimensions {
		return Dimensions{Size: image.Pt(20, itemH)}
	}
	for _, start := range []int{0, 1, 7, 50, 95, 99} {
		for _, offset := range []int{0, 3, 9, -4} {
			var l List
			l.Axis = Vertical
			gtx := Context{
				Ops:         new(op.Ops),
				Constraints: Exact(image.Pt(20, viewport)),
			}
			// Establish Position.Length for the derived-size path.
			l.Layout(gtx, n, el)
			l.Position.First = start
			l.Position.Offset = offset
			l.Position.BeforeEnd = true

			gtx.Ops.Reset()
			first, count := l.VisibleRange(gtx, n, itemH)
			var laidOut []int
			l.Layout(gtx, n, func(gtx Context, idx int) Dimensions {
				laidOut = append(laidOut, idx)
				return el(gtx, idx)
			})
			// Every visible (laid out and displayed) index must be inside
			// the predicted range.
			visFirst, visCount := l.Position.First, l.Position.Count
			if visFirst < first || visFirst+visCount > first+count {
				t.Errorf("start=%d offset=%d: visible [%d,%d) outside predicted [%d,%d)",
					start, offset, visFirst, visFirst+visCount, first, first+count)
			}
			// The prediction must not be wastefully wide: at most the
			// visible count plus 2 items of slack.
			if count > visCount+2 {
				t.Errorf("start=%d offset=%d: predicted count %d far exceeds visible %d",
					start, offset, count, visCount)
			}
		}
	}
}

// TestListVisibleRangeDerivedSize covers itemSize == 0: the average from the
// previous frame drives the estimate, and the very first frame (no data)
// returns the full range.
func TestListVisibleRangeDerivedSize(t *testing.T) {
	const n = 50
	var l List
	l.Axis = Vertical
	gtx := Context{
		Ops:         new(op.Ops),
		Constraints: Exact(image.Pt(20, 40)),
	}
	if first, count := l.VisibleRange(gtx, n, 0); first != 0 || count != n {
		t.Errorf("first frame without size info: got [%d,%d), want full range", first, first+count)
	}
	l.Layout(gtx, n, func(gtx Context, idx int) Dimensions {
		return Dimensions{Size: image.Pt(20, 10)}
	})
	l.Position.First = 20
	first, count := l.VisibleRange(gtx, n, 0)
	if first > 20 || first+count < 24 {
		t.Errorf("derived-size estimate [%d,%d) does not cover visible rows 20-23", first, first+count)
	}
}

// TestListVisibleRangeScrollToEnd covers the ScrollToEnd tail range.
func TestListVisibleRangeScrollToEnd(t *testing.T) {
	const n = 30
	l := List{Axis: Vertical, ScrollToEnd: true}
	gtx := Context{
		Ops:         new(op.Ops),
		Constraints: Exact(image.Pt(20, 40)),
	}
	first, count := l.VisibleRange(gtx, n, 10)
	if first+count != n {
		t.Errorf("ScrollToEnd range [%d,%d) must end at %d", first, first+count, n)
	}
	if count < 4 || count > 6 {
		t.Errorf("ScrollToEnd count %d, want ~viewport/item+slack (4-6)", count)
	}
}
