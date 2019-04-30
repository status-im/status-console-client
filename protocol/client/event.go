package client

import (
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

type EventWithContact interface {
	GetContact() Contact
}

type EventWithType interface {
	GetType() EventType
}

type EventWithError interface {
	GetError() error
}

type EventWithMessage interface {
	GetMessage() *protocol.Message
}

type baseEvent struct {
	Contact Contact   `json:"contact"`
	Type    EventType `json:"type"`
}

func (e baseEvent) GetContact() Contact { return e.Contact }
func (e baseEvent) GetType() EventType  { return e.Type }

type errorEvent struct {
	baseEvent
	Error error `json:"error"`
}

func (e errorEvent) GetError() error { return e.Error }

type messageEvent struct {
	baseEvent
	Message *protocol.Message `json:"message"`
}

func (e messageEvent) GetMessage() *protocol.Message { return e.Message }
