package f32

import (
	"image"
	"math"
	"strconv"
)

type Point struct {
	X, Y float32
}

func (p Point) String() string {
	return "(" + strconv.FormatFloat(float64(p.X), 'f', -1, 32) +
		"," + strconv.FormatFloat(float64(p.Y), 'f', -1, 32) + ")"
}

func Pt(x, y float32) Point {
	return Point{X: x, Y: y}
}

func (p Point) Add(p2 Point) Point {
	return Point{X: p.X + p2.X, Y: p.Y + p2.Y}
}

func (p Point) Sub(p2 Point) Point {
	return Point{X: p.X - p2.X, Y: p.Y - p2.Y}
}

func (p Point) Mul(s float32) Point {
	return Point{X: p.X * s, Y: p.Y * s}
}

func (p Point) Div(s float32) Point {
	return Point{X: p.X / s, Y: p.Y / s}
}

func (p Point) Round() image.Point {
	return image.Point{
		X: int(math.Round(float64(p.X))),
		Y: int(math.Round(float64(p.Y))),
	}
}
