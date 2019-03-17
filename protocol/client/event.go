package client

import "github.com/status-im/status-console-client/protocol/v1"

// A list of available events sent from the client.
const (
	EventTypeInit      int = iota + 1
	EventTypeRearrange     // messages were rearranged
	EventTypeMessage       // a new message was appended
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

type EventMessage interface {
	Event
	Message() *protocol.Message
}

type baseEvent struct {
	contact Contact
	typ     int
}

func (e baseEvent) Contact() Contact { return e.contact }
func (e baseEvent) Type() int        { return e.typ }

type errorEvent struct {
	baseEvent
	err error
}

func (e errorEvent) Error() error { return e.err }

type messageEvent struct {
	baseEvent
	message *protocol.Message
}

func (e messageEvent) Message() *protocol.Message { return e.message }

// type eventError struct {
// 	Event
// 	err error
// }

// func (e eventError) Error() error { return e.err }
