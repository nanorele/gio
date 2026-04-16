package layout_test

import (
	"fmt"
	"image"

	"github.com/nanorele/gio/layout"
	"github.com/nanorele/gio/op"
)

func ExampleInset() {
	gtx := layout.Context{
		Ops: new(op.Ops),

		Constraints: layout.Constraints{
			Max: image.Point{X: 100, Y: 100},
		},
	}

	inset := layout.UniformInset(10)
	dims := inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {

		dims := layoutWidget(gtx, 50, 50)
		fmt.Println(dims.Size)
		return dims
	})

	fmt.Println(dims.Size)

}

func ExampleDirection() {
	gtx := layout.Context{
		Ops: new(op.Ops),

		Constraints: layout.Exact(image.Point{X: 100, Y: 100}),
	}

	dims := layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {

		dims := layoutWidget(gtx, 50, 50)
		fmt.Println(dims.Size)
		return dims
	})

	fmt.Println(dims.Size)

}

func ExampleFlex() {
	gtx := layout.Context{
		Ops: new(op.Ops),

		Constraints: layout.Exact(image.Point{X: 100, Y: 100}),
	}

	layout.Flex{WeightSum: 2}.Layout(gtx,

		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			fmt.Printf("Rigid: %v\n", gtx.Constraints)
			return layoutWidget(gtx, 10, 10)
		}),

		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			fmt.Printf("50%%: %v\n", gtx.Constraints)
			return layoutWidget(gtx, 10, 10)
		}),
	)

}

func ExampleStack() {
	gtx := layout.Context{
		Ops: new(op.Ops),
		Constraints: layout.Constraints{
			Max: image.Point{X: 100, Y: 100},
		},
	}

	layout.Stack{}.Layout(gtx,

		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			fmt.Printf("Expand: %v\n", gtx.Constraints)
			return layoutWidget(gtx, 10, 10)
		}),

		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return layoutWidget(gtx, 50, 50)
		}),
	)

}

func ExampleBackground() {
	gtx := layout.Context{
		Ops: new(op.Ops),
		Constraints: layout.Constraints{
			Max: image.Point{X: 100, Y: 100},
		},
	}

	layout.Background{}.Layout(gtx,

		func(gtx layout.Context) layout.Dimensions {
			fmt.Printf("Expand: %v\n", gtx.Constraints)
			return layoutWidget(gtx, 10, 10)
		},

		func(gtx layout.Context) layout.Dimensions {
			return layoutWidget(gtx, 50, 50)
		},
	)

}

func ExampleList() {
	gtx := layout.Context{
		Ops: new(op.Ops),

		Constraints: layout.Exact(image.Point{X: 100, Y: 100}),
	}

	const listLen = 1e6

	var list layout.List
	list.Layout(gtx, listLen, func(gtx layout.Context, i int) layout.Dimensions {
		return layoutWidget(gtx, 20, 20)
	})

	fmt.Println(list.Position.Count)

}

func layoutWidget(ctx layout.Context, width, height int) layout.Dimensions {
	return layout.Dimensions{
		Size: image.Point{
			X: width,
			Y: height,
		},
	}
}
