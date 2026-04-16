package material

import (
	"image/color"

	"github.com/nanorele/gio/font"
	"github.com/nanorele/gio/internal/f32color"
	"github.com/nanorele/gio/layout"
	"github.com/nanorele/gio/op"
	"github.com/nanorele/gio/op/paint"
	"github.com/nanorele/gio/text"
	"github.com/nanorele/gio/unit"
	"github.com/nanorele/gio/widget"
)

type LabelStyle struct {
	Font font.Font

	Color color.NRGBA

	SelectionColor color.NRGBA

	Alignment text.Alignment

	MaxLines int

	WrapPolicy text.WrapPolicy

	Truncator string

	Text string

	TextSize unit.Sp

	LineHeight unit.Sp

	LineHeightScale float32

	Shaper *text.Shaper

	State *widget.Selectable
}

func H1(th *Theme, txt string) LabelStyle {
	label := Label(th, th.TextSize*96.0/16.0, txt)
	label.Font.Weight = font.Light
	return label
}

func H2(th *Theme, txt string) LabelStyle {
	label := Label(th, th.TextSize*60.0/16.0, txt)
	label.Font.Weight = font.Light
	return label
}

func H3(th *Theme, txt string) LabelStyle {
	return Label(th, th.TextSize*48.0/16.0, txt)
}

func H4(th *Theme, txt string) LabelStyle {
	return Label(th, th.TextSize*34.0/16.0, txt)
}

func H5(th *Theme, txt string) LabelStyle {
	return Label(th, th.TextSize*24.0/16.0, txt)
}

func H6(th *Theme, txt string) LabelStyle {
	label := Label(th, th.TextSize*20.0/16.0, txt)
	label.Font.Weight = font.Medium
	return label
}

func Subtitle1(th *Theme, txt string) LabelStyle {
	return Label(th, th.TextSize*16.0/16.0, txt)
}

func Subtitle2(th *Theme, txt string) LabelStyle {
	label := Label(th, th.TextSize*14.0/16.0, txt)
	label.Font.Weight = font.Medium
	return label
}

func Body1(th *Theme, txt string) LabelStyle {
	return Label(th, th.TextSize, txt)
}

func Body2(th *Theme, txt string) LabelStyle {
	return Label(th, th.TextSize*14.0/16.0, txt)
}

func Caption(th *Theme, txt string) LabelStyle {
	return Label(th, th.TextSize*12.0/16.0, txt)
}

func Overline(th *Theme, txt string) LabelStyle {
	return Label(th, th.TextSize*10.0/16.0, txt)
}

func Label(th *Theme, size unit.Sp, txt string) LabelStyle {
	l := LabelStyle{
		Text:           txt,
		Color:          th.Palette.Fg,
		SelectionColor: f32color.MulAlpha(th.Palette.ContrastBg, 0x60),
		TextSize:       size,
		Shaper:         th.Shaper,
	}
	l.Font.Typeface = th.Face
	return l
}

func (l LabelStyle) Layout(gtx layout.Context) layout.Dimensions {
	textColorMacro := op.Record(gtx.Ops)
	paint.ColorOp{Color: l.Color}.Add(gtx.Ops)
	textColor := textColorMacro.Stop()
	selectColorMacro := op.Record(gtx.Ops)
	paint.ColorOp{Color: l.SelectionColor}.Add(gtx.Ops)
	selectColor := selectColorMacro.Stop()

	if l.State != nil {
		if l.State.Text() != l.Text {
			l.State.SetText(l.Text)
		}
		l.State.Alignment = l.Alignment
		l.State.MaxLines = l.MaxLines
		l.State.Truncator = l.Truncator
		l.State.WrapPolicy = l.WrapPolicy
		l.State.LineHeight = l.LineHeight
		l.State.LineHeightScale = l.LineHeightScale
		return l.State.Layout(gtx, l.Shaper, l.Font, l.TextSize, textColor, selectColor)
	}
	tl := widget.Label{
		Alignment:       l.Alignment,
		MaxLines:        l.MaxLines,
		Truncator:       l.Truncator,
		WrapPolicy:      l.WrapPolicy,
		LineHeight:      l.LineHeight,
		LineHeightScale: l.LineHeightScale,
	}
	return tl.Layout(gtx, l.Shaper, l.Font, l.TextSize, l.Text, textColor)
}
