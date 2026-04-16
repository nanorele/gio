package clipboard

import (
	"io"

	"github.com/nanorele/gio/io/event"
)

type WriteCmd struct {
	Type string
	Data io.ReadCloser
}

type ReadCmd struct {
	Tag event.Tag
}

func (WriteCmd) ImplementsCommand() {}
func (ReadCmd) ImplementsCommand()  {}
