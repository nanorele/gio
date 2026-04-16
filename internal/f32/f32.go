package f32

import (
	"image"
	"math"

	"github.com/nanorele/gio/f32"
)

type Point = f32.Point

type Affine2D = f32.Affine2D

var NewAffine2D = f32.NewAffine2D

var AffineId = f32.AffineId

type Rectangle struct {
	Min, Max Point
}

func (r Rectangle) String() string {
	return r.Min.String() + "-" + r.Max.String()
}

func Rect(x0, y0, x1, y1 float32) Rectangle {
	if x0 > x1 {
		x0, x1 = x1, x0
	}
	if y0 > y1 {
		y0, y1 = y1, y0
	}
	return Rectangle{Point{X: x0, Y: y0}, Point{X: x1, Y: y1}}
}

var Pt = f32.Pt

func (r Rectangle) Size() Point {
	return Point{X: r.Dx(), Y: r.Dy()}
}

func (r Rectangle) Dx() float32 {
	return r.Max.X - r.Min.X
}

func (r Rectangle) Dy() float32 {
	return r.Max.Y - r.Min.Y
}

func (r Rectangle) Intersect(s Rectangle) Rectangle {
	if r.Min.X < s.Min.X {
		r.Min.X = s.Min.X
	}
	if r.Min.Y < s.Min.Y {
		r.Min.Y = s.Min.Y
	}
	if r.Max.X > s.Max.X {
		r.Max.X = s.Max.X
	}
	if r.Max.Y > s.Max.Y {
		r.Max.Y = s.Max.Y
	}
	if r.Empty() {
		return Rectangle{}
	}
	return r
}

func (r Rectangle) Union(s Rectangle) Rectangle {
	if r.Empty() {
		return s
	}
	if s.Empty() {
		return r
	}
	if r.Min.X > s.Min.X {
		r.Min.X = s.Min.X
	}
	if r.Min.Y > s.Min.Y {
		r.Min.Y = s.Min.Y
	}
	if r.Max.X < s.Max.X {
		r.Max.X = s.Max.X
	}
	if r.Max.Y < s.Max.Y {
		r.Max.Y = s.Max.Y
	}
	return r
}

func (r Rectangle) Canon() Rectangle {
	if r.Max.X < r.Min.X {
		r.Min.X, r.Max.X = r.Max.X, r.Min.X
	}
	if r.Max.Y < r.Min.Y {
		r.Min.Y, r.Max.Y = r.Max.Y, r.Min.Y
	}
	return r
}

func (r Rectangle) Empty() bool {
	return r.Min.X >= r.Max.X || r.Min.Y >= r.Max.Y
}

func (r Rectangle) Add(p Point) Rectangle {
	return Rectangle{
		Point{X: r.Min.X + p.X, Y: r.Min.Y + p.Y},
		Point{X: r.Max.X + p.X, Y: r.Max.Y + p.Y},
	}
}

func (r Rectangle) Sub(p Point) Rectangle {
	return Rectangle{
		Point{X: r.Min.X - p.X, Y: r.Min.Y - p.Y},
		Point{X: r.Max.X - p.X, Y: r.Max.Y - p.Y},
	}
}

func (r Rectangle) Round() image.Rectangle {
	return image.Rectangle{
		Min: image.Point{
			X: int(floor(r.Min.X)),
			Y: int(floor(r.Min.Y)),
		},
		Max: image.Point{
			X: int(ceil(r.Max.X)),
			Y: int(ceil(r.Max.Y)),
		},
	}
}

func FRect(r image.Rectangle) Rectangle {
	return Rectangle{
		Min: FPt(r.Min), Max: FPt(r.Max),
	}
}

func FPt(p image.Point) Point {
	return Point{X: float32(p.X), Y: float32(p.Y)}

}

func ceil(v float32) int {
	return int(math.Ceil(float64(v)))
}

func floor(v float32) int {
	return int(math.Floor(float64(v)))
}
