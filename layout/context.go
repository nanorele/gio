package layout

import (
	"time"

	"github.com/nanorele/gio/io/input"
	"github.com/nanorele/gio/io/system"
	"github.com/nanorele/gio/op"
	"github.com/nanorele/gio/unit"
)

type Context struct {
	Constraints Constraints

	Metric unit.Metric

	Now time.Time

	Locale system.Locale

	Values map[string]any

	input.Source
	*op.Ops
}

func (c Context) Dp(v unit.Dp) int {
	return c.Metric.Dp(v)
}

func (c Context) Sp(v unit.Sp) int {
	return c.Metric.Sp(v)
}

func (c Context) Disabled() Context {
	c.Source = c.Source.Disabled()
	return c
}
