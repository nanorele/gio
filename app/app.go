package app

import (
	"github.com/nanorele/gio/io/event"
	"golang.org/x/net/idna"
	"image"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nanorele/gio/io/input"
	"github.com/nanorele/gio/layout"
	"github.com/nanorele/gio/op"
	"github.com/nanorele/gio/unit"
)

var extraArgs string

var ID = ""

type FrameEvent struct {
	Now time.Time

	Metric unit.Metric

	Size image.Point

	Insets Insets

	Frame func(frame *op.Ops)

	Source input.Source
}

type URLEvent struct {
	URL *url.URL
}

type ViewEvent interface {
	implementsViewEvent()
	ImplementsEvent()

	Valid() bool
}

type Insets struct {
	Top, Bottom, Left, Right unit.Dp
}

func NewContext(ops *op.Ops, e FrameEvent) layout.Context {
	ops.Reset()

	size := e.Size

	if e.Insets != (Insets{}) {
		left := e.Metric.Dp(e.Insets.Left)
		top := e.Metric.Dp(e.Insets.Top)
		op.Offset(image.Point{
			X: left,
			Y: top,
		}).Add(ops)

		size.X -= left + e.Metric.Dp(e.Insets.Right)
		size.Y -= top + e.Metric.Dp(e.Insets.Bottom)
	}

	return layout.Context{
		Ops:         ops,
		Now:         e.Now,
		Source:      e.Source,
		Metric:      e.Metric,
		Constraints: layout.Exact(size),
	}
}

func DataDir() (string, error) {
	return dataDir()
}

func Main() {
	osMain()
}

func Events(yield func(event.Event) bool) {
	yieldGlobalEvent = yield
	osMain()
}

var yieldGlobalEvent func(evt event.Event) bool

func processGlobalEvent(evt event.Event) {
	if yieldGlobalEvent == nil {
		return
	}
	if !yieldGlobalEvent(evt) {
		yieldGlobalEvent = nil
	}
}

func (FrameEvent) ImplementsEvent() {}
func (URLEvent) ImplementsEvent()   {}

func init() {
	if extraArgs != "" {
		args := strings.Split(extraArgs, "|")
		os.Args = append(os.Args, args...)
	}
	if ID == "" {
		ID = filepath.Base(os.Args[0])
	}
}

func newURLEvent(rawurl string) (URLEvent, error) {
	u, err := url.Parse(rawurl)
	if err != nil {
		return URLEvent{}, err
	}
	u.Host, err = idna.Punycode.ToUnicode(u.Hostname())
	if err != nil {
		return URLEvent{}, err
	}
	u, err = url.Parse(u.String())
	if err != nil {
		return URLEvent{}, err
	}
	return URLEvent{URL: u}, nil
}
