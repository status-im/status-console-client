package client

const (
	EventTypeMessage int = iota + 1
	EventTypeError
)

type Event interface {
	Contact() Contact
	Type() int
}

type EventError interface {
	Event
	Error() error
}

type baseEvent struct {
	contact Contact
	typ     int
}

func (e baseEvent) Contact() Contact { return e.contact }
func (e baseEvent) Type() int        { return e.typ }

// type eventError struct {
// 	Event
// 	err error
// }

// func (e eventError) Error() error { return e.err }
