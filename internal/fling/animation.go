package fling

import (
	"math"
	"runtime"
	"time"

	"github.com/nanorele/gio/unit"
)

type Animation struct {
	x float32

	t0 time.Time

	v0 float32
}

const (
	minFlingVelocity  = unit.Dp(50)
	maxFlingVelocity  = unit.Dp(8000)
	thresholdVelocity = 1
)

func (f *Animation) Start(c unit.Metric, now time.Time, velocity float32) bool {
	min := float32(c.Dp(minFlingVelocity))
	v := velocity
	if -min <= v && v <= min {
		return false
	}
	max := float32(c.Dp(maxFlingVelocity))
	if v > max {
		v = max
	} else if v < -max {
		v = -max
	}
	f.init(now, v)
	return true
}

func (f *Animation) init(now time.Time, v0 float32) {
	f.t0 = now
	f.v0 = v0
	f.x = 0
}

func (f *Animation) Active() bool {
	return f.v0 != 0
}

func (f *Animation) Tick(now time.Time) int {
	if !f.Active() {
		return 0
	}
	var k float32
	if runtime.GOOS == "darwin" {
		k = -2
	} else {
		k = -4.2
	}
	t := now.Sub(f.t0)

	ekt := float32(math.Exp(float64(k) * t.Seconds()))
	x := f.v0*ekt/k - f.v0/k
	dist := x - f.x
	idist := int(dist)
	f.x += float32(idist)

	v := f.v0 * ekt
	if -thresholdVelocity < v && v < thresholdVelocity {
		f.v0 = 0
	}
	return idist
}
