package app

type DestroyEvent struct {
	Err error
}

func (DestroyEvent) ImplementsEvent() {}
