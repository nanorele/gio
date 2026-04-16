package widget

import (
	"image"

	"github.com/nanorele/gio/f32"
	"github.com/nanorele/gio/layout"
)

type Fit uint8

const (
	Unscaled Fit = iota

	Contain

	Cover

	ScaleDown

	Fill
)

func (fit Fit) scale(cs layout.Constraints, pos layout.Direction, dims layout.Dimensions) (layout.Dimensions, f32.Affine2D) {
	widgetSize := dims.Size

	if fit == Unscaled || dims.Size.X == 0 || dims.Size.Y == 0 {
		dims.Size = cs.Constrain(dims.Size)

		offset := pos.Position(widgetSize, dims.Size)
		dims.Baseline += offset.Y
		return dims, f32.AffineId().Offset(layout.FPt(offset))
	}

	scale := f32.Point{
		X: float32(cs.Max.X) / float32(dims.Size.X),
		Y: float32(cs.Max.Y) / float32(dims.Size.Y),
	}

	switch fit {
	case Contain:
		if scale.Y < scale.X {
			scale.X = scale.Y
		} else {
			scale.Y = scale.X
		}
	case Cover:
		if scale.Y > scale.X {
			scale.X = scale.Y
		} else {
			scale.Y = scale.X
		}
	case ScaleDown:
		if scale.Y < scale.X {
			scale.X = scale.Y
		} else {
			scale.Y = scale.X
		}

		if scale.X >= 1 {
			dims.Size = cs.Constrain(dims.Size)

			offset := pos.Position(widgetSize, dims.Size)
			dims.Baseline += offset.Y
			return dims, f32.AffineId().Offset(layout.FPt(offset))
		}
	case Fill:
	}

	var scaledSize image.Point
	scaledSize.X = int(float32(widgetSize.X) * scale.X)
	scaledSize.Y = int(float32(widgetSize.Y) * scale.Y)
	dims.Size = cs.Constrain(scaledSize)
	dims.Baseline = int(float32(dims.Baseline) * scale.Y)

	offset := pos.Position(scaledSize, dims.Size)
	trans := f32.AffineId().
		Scale(f32.Point{}, scale).
		Offset(layout.FPt(offset))

	dims.Baseline += offset.Y

	return dims, trans
}
