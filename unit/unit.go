package unit

import (
	"math"
)

type Metric struct {
	PxPerDp float32

	PxPerSp float32
}

type (
	Dp float32

	Sp float32
)

func (c Metric) Dp(v Dp) int {
	return int(math.Round(float64(nonZero(c.PxPerDp)) * float64(v)))
}

func (c Metric) Sp(v Sp) int {
	return int(math.Round(float64(nonZero(c.PxPerSp)) * float64(v)))
}

func (c Metric) DpToSp(v Dp) Sp {
	return Sp(float32(v) * nonZero(c.PxPerDp) / nonZero(c.PxPerSp))
}

func (c Metric) SpToDp(v Sp) Dp {
	return Dp(float32(v) * nonZero(c.PxPerSp) / nonZero(c.PxPerDp))
}

func (c Metric) PxToSp(v int) Sp {
	return Sp(float32(v) / nonZero(c.PxPerSp))
}

func (c Metric) PxToDp(v int) Dp {
	return Dp(float32(v) / nonZero(c.PxPerDp))
}

func nonZero(v float32) float32 {
	if v == 0. {
		return 1
	}
	return v
}
