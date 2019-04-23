package client

import (
	"encoding/json"

	"github.com/status-im/status-console-client/protocol/v1"
)

type EventType int

//go:generate stringer -type=EventType

// A list of available events sent from the client.
const (
	EventTypeInit      EventType = iota + 1
	EventTypeRearrange           // messages were rearranged
	EventTypeMessage             // a new message was appended
	EventTypeError
)

type Event interface {
	Contact() Contact
	Type() EventType
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
	typ     EventType
}

func (e baseEvent) Contact() Contact { return e.contact }
func (e baseEvent) Type() EventType  { return e.typ }

func (e baseEvent) MarshalJSON() ([]byte, error) {
	item := struct {
		Contact Contact `json:"contact"`
		Type    string  `json:"type"`
	}{
		Contact: e.contact,
		Type:    e.typ.String(),
	}

	return json.Marshal(item)
}

type errorEvent struct {
	baseEvent
	err error
}

func (e errorEvent) Error() error { return e.err }

func (e errorEvent) MarshalJSON() ([]byte, error) {
	item := struct {
		Contact Contact `json:"contact"`
		Type    string  `json:"type"`
		Error   error   `json:"error"`
	}{
		Contact: e.contact,
		Type:    e.typ.String(),
		Error:   e.err,
	}

	return json.Marshal(item)
}

type messageEvent struct {
	baseEvent
	message *protocol.Message
}

func (e messageEvent) Message() *protocol.Message { return e.message }

func (e messageEvent) MarshalJSON() ([]byte, error) {
	item := struct {
		Contact Contact           `json:"contact"`
		Type    string            `json:"type"`
		Message *protocol.Message `json:"message"`
	}{
		Contact: e.contact,
		Type:    e.typ.String(),
		Message: e.message,
	}

	return json.Marshal(item)
}
