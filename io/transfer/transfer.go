package transfer

import (
	"io"

	"github.com/nanorele/gio/io/event"
)

type OfferCmd struct {
	Tag event.Tag

	Type string

	Data io.ReadCloser
}

func (OfferCmd) ImplementsCommand() {}

type SourceFilter struct {
	Target event.Tag

	Type string
}

type TargetFilter struct {
	Target event.Tag

	Type string
}

type RequestEvent struct {
	Type string
}

func (RequestEvent) ImplementsEvent() {}

type InitiateEvent struct{}

func (InitiateEvent) ImplementsEvent() {}

type CancelEvent struct{}

func (CancelEvent) ImplementsEvent() {}

type DataEvent struct {
	Type string

	Open func() io.ReadCloser
}

func (DataEvent) ImplementsEvent() {}

func (SourceFilter) ImplementsFilter() {}
func (TargetFilter) ImplementsFilter() {}
