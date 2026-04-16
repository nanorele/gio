package system

import (
	"strings"

	"github.com/nanorele/gio/internal/ops"
	"github.com/nanorele/gio/op"
)

type ActionInputOp Action

type Action uint

const (
	ActionMinimize Action = 1 << iota

	ActionMaximize

	ActionUnmaximize

	ActionFullscreen

	ActionRaise

	ActionCenter

	ActionClose

	ActionMove
)

func (op ActionInputOp) Add(o *op.Ops) {
	data := ops.Write(&o.Internal, ops.TypeActionInputLen)
	data[0] = byte(ops.TypeActionInput)
	data[1] = byte(op)
}

func (a Action) String() string {
	var buf strings.Builder
	for b := Action(1); a != 0; b <<= 1 {
		if a&b != 0 {
			if buf.Len() > 0 {
				buf.WriteByte('|')
			}
			buf.WriteString(b.string())
			a &^= b
		}
	}
	return buf.String()
}

func (a Action) string() string {
	switch a {
	case ActionMinimize:
		return "ActionMinimize"
	case ActionMaximize:
		return "ActionMaximize"
	case ActionUnmaximize:
		return "ActionUnmaximize"
	case ActionClose:
		return "ActionClose"
	case ActionMove:
		return "ActionMove"
	}
	return ""
}
