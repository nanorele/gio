package widget_test

import (
	"fmt"
	"image"
	"io"
	"strings"

	"github.com/nanorele/gio/f32"
	"github.com/nanorele/gio/io/event"
	"github.com/nanorele/gio/io/input"
	"github.com/nanorele/gio/io/pointer"
	"github.com/nanorele/gio/io/transfer"
	"github.com/nanorele/gio/layout"
	"github.com/nanorele/gio/op"
	"github.com/nanorele/gio/op/clip"
	"github.com/nanorele/gio/widget"
)

func ExampleClickable_passthrough() {

	var button1, button2 widget.Clickable
	var r input.Router
	gtx := layout.Context{
		Ops:         new(op.Ops),
		Constraints: layout.Exact(image.Pt(100, 100)),
		Source:      r.Source(),
	}

	widget := func() {
		content := func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{Size: gtx.Constraints.Min} }
		button1.Layout(gtx, content)

		defer pointer.PassOp{}.Push(gtx.Ops).Pop()
		button2.Layout(gtx, content)
	}

	widget()
	r.Frame(gtx.Ops)

	r.Queue(
		pointer.Event{
			Source:   pointer.Mouse,
			Buttons:  pointer.ButtonPrimary,
			Kind:     pointer.Press,
			Position: f32.Pt(50, 50),
		},
		pointer.Event{
			Source:   pointer.Mouse,
			Buttons:  pointer.ButtonPrimary,
			Kind:     pointer.Release,
			Position: f32.Pt(50, 50),
		},
	)

	if button1.Clicked(gtx) {
		fmt.Println("button1 clicked!")
	}
	if button2.Clicked(gtx) {
		fmt.Println("button2 clicked!")
	}

}

func ExampleDraggable_Layout() {
	var r input.Router
	gtx := layout.Context{
		Ops:         new(op.Ops),
		Constraints: layout.Exact(image.Pt(100, 100)),
		Source:      r.Source(),
	}

	const mime = "MyMime"
	drag := &widget.Draggable{Type: mime}
	var drop int

	widget := func() {

		w := func(gtx layout.Context) layout.Dimensions {
			sz := image.Pt(10, 10)
			return layout.Dimensions{Size: sz}
		}
		drag.Layout(gtx, w, w)

		if m, ok := drag.Update(gtx); ok {
			drag.Offer(gtx, m, io.NopCloser(strings.NewReader("hello world")))
		}

		ds := clip.Rect{
			Min: image.Pt(20, 20),
			Max: image.Pt(40, 40),
		}.Push(gtx.Ops)
		event.Op(gtx.Ops, &drop)
		ds.Pop()

		for {
			ev, ok := gtx.Event(transfer.TargetFilter{Target: &drop, Type: mime})
			if !ok {
				break
			}
			switch e := ev.(type) {
			case transfer.DataEvent:
				data := e.Open()
				defer data.Close()
				content, _ := io.ReadAll(data)
				fmt.Println(string(content))
			}
		}
	}

	widget()
	r.Frame(gtx.Ops)

	r.Queue(
		pointer.Event{
			Kind:     pointer.Press,
			Position: f32.Pt(5, 5),
		},
		pointer.Event{
			Kind:     pointer.Move,
			Position: f32.Pt(5, 5),
		},
		pointer.Event{
			Kind:     pointer.Release,
			Position: f32.Pt(30, 30),
		},
	)

	widget()
	r.Frame(gtx.Ops)

	widget()

}
