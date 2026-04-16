package app

import (
	"errors"
	"image"
	"image/color"

	"github.com/nanorele/gio/io/event"
	"github.com/nanorele/gio/io/key"
	"github.com/nanorele/gio/op"

	"github.com/nanorele/gio/gpu"
	"github.com/nanorele/gio/io/pointer"
	"github.com/nanorele/gio/io/system"
	"github.com/nanorele/gio/unit"
)

var errOutOfDate = errors.New("app: GPU surface out of date")

type Config struct {
	Size image.Point

	MaxSize image.Point

	MinSize image.Point

	Title string

	Mode WindowMode

	NavigationColor color.NRGBA

	Orientation Orientation

	CustomRenderer bool

	Decorated bool

	TopMost bool

	Focused bool

	decoHeight unit.Dp
}

type ConfigEvent struct {
	Config Config
}

func (c *Config) apply(m unit.Metric, options []Option) {
	for _, o := range options {
		o(m, c)
	}
}

type wakeupEvent struct{}

type WindowMode uint8

const (
	Windowed WindowMode = iota

	Fullscreen

	Minimized

	Maximized
)

func (m WindowMode) Option() Option {
	return func(_ unit.Metric, cnf *Config) {
		cnf.Mode = m
	}
}

func (m WindowMode) String() string {
	switch m {
	case Windowed:
		return "windowed"
	case Fullscreen:
		return "fullscreen"
	case Minimized:
		return "minimized"
	case Maximized:
		return "maximized"
	}
	return ""
}

type Orientation uint8

const (
	AnyOrientation Orientation = iota

	LandscapeOrientation

	PortraitOrientation
)

func (o Orientation) Option() Option {
	return func(_ unit.Metric, cnf *Config) {
		cnf.Orientation = o
	}
}

func (o Orientation) String() string {
	switch o {
	case AnyOrientation:
		return "any"
	case LandscapeOrientation:
		return "landscape"
	case PortraitOrientation:
		return "portrait"
	}
	return ""
}

type eventLoop struct {
	win *callbacks

	wakeup func()

	driverFuncs chan func()

	invalidates chan struct{}

	immediateInvalidates chan struct{}

	events   chan event.Event
	frames   chan *op.Ops
	frameAck chan struct{}

	delivering bool
}

type frameEvent struct {
	FrameEvent

	Sync bool
}

type context interface {
	API() gpu.API
	RenderTarget() (gpu.RenderTarget, error)
	Present() error
	Refresh() error
	Release()
	Lock() error
	Unlock()
}

type driver interface {
	Event() event.Event

	Invalidate()

	SetAnimating(anim bool)

	ShowTextInput(show bool)
	SetInputHint(mode key.InputHint)
	NewContext() (context, error)

	ReadClipboard()

	WriteClipboard(mime string, s []byte)

	Configure([]Option)

	SetCursor(cursor pointer.Cursor)

	Perform(system.Action)

	EditorStateChanged(old, new editorState)

	Run(f func())

	Frame(frame *op.Ops)

	ProcessEvent(e event.Event)
}

type windowRendezvous struct {
	in      chan windowAndConfig
	out     chan windowAndConfig
	windows chan struct{}
}

type windowAndConfig struct {
	window  *callbacks
	options []Option
}

func newWindowRendezvous() *windowRendezvous {
	wr := &windowRendezvous{
		in:      make(chan windowAndConfig),
		out:     make(chan windowAndConfig),
		windows: make(chan struct{}),
	}
	go func() {
		in := wr.in
		var window windowAndConfig
		var out chan windowAndConfig
		for {
			select {
			case w := <-in:
				window = w
				out = wr.out
			case out <- window:
			}
		}
	}()
	return wr
}

func newEventLoop(w *callbacks, wakeup func()) *eventLoop {
	return &eventLoop{
		win:                  w,
		wakeup:               wakeup,
		events:               make(chan event.Event),
		invalidates:          make(chan struct{}, 1),
		immediateInvalidates: make(chan struct{}),
		frames:               make(chan *op.Ops),
		frameAck:             make(chan struct{}),
		driverFuncs:          make(chan func(), 1),
	}
}

func (e *eventLoop) Frame(frame *op.Ops) {
	e.frames <- frame
	<-e.frameAck
}

func (e *eventLoop) Event() event.Event {
	for {
		evt := <-e.events

		if _, ok := evt.(flushEvent); ok {
			continue
		}
		return evt
	}
}

func (e *eventLoop) Invalidate() {
	select {
	case e.immediateInvalidates <- struct{}{}:

	case e.invalidates <- struct{}{}:

		e.wakeup()
	default:

	}
}

func (e *eventLoop) Run(f func()) {
	e.driverFuncs <- f
	e.wakeup()
}

func (e *eventLoop) FlushEvents() {
	if e.delivering {
		return
	}
	e.delivering = true
	defer func() { e.delivering = false }()
	for {
		evt, ok := e.win.nextEvent()
		if !ok {
			break
		}
		e.deliverEvent(evt)
	}
}

func (e *eventLoop) deliverEvent(evt event.Event) {
	var frames <-chan *op.Ops
	for {
		select {
		case f := <-e.driverFuncs:
			f()
		case frame := <-frames:

			frames = nil
			e.win.ProcessFrame(frame, e.frameAck)
		case e.events <- evt:
			switch evt.(type) {
			case flushEvent, DestroyEvent:

				return
			case FrameEvent:
				frames = e.frames
			}
			evt = theFlushEvent
		case <-e.invalidates:
			e.win.Invalidate()
		case <-e.immediateInvalidates:
			e.win.Invalidate()
		}
	}
}

func (e *eventLoop) Wakeup() {
	for {
		select {
		case f := <-e.driverFuncs:
			f()
		case <-e.invalidates:
			e.win.Invalidate()
		case <-e.immediateInvalidates:
			e.win.Invalidate()
		default:
			return
		}
	}
}

func walkActions(actions system.Action, do func(system.Action)) {
	for a := system.Action(1); actions != 0; a <<= 1 {
		if actions&a != 0 {
			actions &^= a
			do(a)
		}
	}
}

func (wakeupEvent) ImplementsEvent() {}
func (ConfigEvent) ImplementsEvent() {}
