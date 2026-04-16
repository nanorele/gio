package material

import (
	"image/color"

	"golang.org/x/exp/shiny/materialdesign/icons"

	"github.com/nanorele/gio/font"
	"github.com/nanorele/gio/text"
	"github.com/nanorele/gio/unit"
	"github.com/nanorele/gio/widget"
)

type Palette struct {
	Bg color.NRGBA

	Fg color.NRGBA

	ContrastBg color.NRGBA

	ContrastFg color.NRGBA
}

type Theme struct {
	Shaper *text.Shaper
	Palette
	TextSize unit.Sp
	Icon     struct {
		CheckBoxChecked   *widget.Icon
		CheckBoxUnchecked *widget.Icon
		RadioChecked      *widget.Icon
		RadioUnchecked    *widget.Icon
	}

	Face font.Typeface

	FingerSize unit.Dp
}

func NewTheme() *Theme {
	t := &Theme{Shaper: &text.Shaper{}}
	t.Palette = Palette{
		Fg:         rgb(0x000000),
		Bg:         rgb(0xffffff),
		ContrastBg: rgb(0x3f51b5),
		ContrastFg: rgb(0xffffff),
	}
	t.TextSize = 16

	t.Icon.CheckBoxChecked = mustIcon(widget.NewIcon(icons.ToggleCheckBox))
	t.Icon.CheckBoxUnchecked = mustIcon(widget.NewIcon(icons.ToggleCheckBoxOutlineBlank))
	t.Icon.RadioChecked = mustIcon(widget.NewIcon(icons.ToggleRadioButtonChecked))
	t.Icon.RadioUnchecked = mustIcon(widget.NewIcon(icons.ToggleRadioButtonUnchecked))

	t.FingerSize = 38

	return t
}

func (t Theme) WithPalette(p Palette) Theme {
	t.Palette = p
	return t
}

func mustIcon(ic *widget.Icon, err error) *widget.Icon {
	if err != nil {
		panic(err)
	}
	return ic
}

func rgb(c uint32) color.NRGBA {
	return argb(0xff000000 | c)
}

func argb(c uint32) color.NRGBA {
	return color.NRGBA{A: uint8(c >> 24), R: uint8(c >> 16), G: uint8(c >> 8), B: uint8(c)}
}
