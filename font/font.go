package font

import (
	"github.com/nanorele/typesetting/font"
)

type FontFace struct {
	Font Font
	Face Face
}

type Style int

type Weight int

type Font struct {
	Typeface Typeface

	Style Style

	Weight Weight
}

type Face interface {
	Face() *font.Face
}

type Typeface string

const (
	Regular Style = iota
	Italic
)

const (
	Thin       Weight = -300
	ExtraLight Weight = -200
	Light      Weight = -100
	Normal     Weight = 0
	Medium     Weight = 100
	SemiBold   Weight = 200
	Bold       Weight = 300
	ExtraBold  Weight = 400
	Black      Weight = 500
)

func (s Style) String() string {
	switch s {
	case Regular:
		return "Regular"
	case Italic:
		return "Italic"
	default:
		panic("invalid Style")
	}
}

func (w Weight) String() string {
	switch w {
	case Thin:
		return "Thin"
	case ExtraLight:
		return "ExtraLight"
	case Light:
		return "Light"
	case Normal:
		return "Normal"
	case Medium:
		return "Medium"
	case SemiBold:
		return "SemiBold"
	case Bold:
		return "Bold"
	case ExtraBold:
		return "ExtraBold"
	case Black:
		return "Black"
	default:
		panic("invalid Weight")
	}
}
