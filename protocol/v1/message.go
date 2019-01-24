package protocol

import (
	"bytes"
	"errors"
)

const (
	// ContentTypeTextPlain means that the message contains plain text.
	ContentTypeTextPlain = "text/plain"
)

// Message types.
const (
	MessageTypePublicGroupUserMessage = "public-group-user-message"
	MessageTypePrivateUserMessage     = "user-message"
)

var (
	// ErrInvalidDecodedValue means that the decoded message is of wrong type.
	// This might mean that the status message serialization tag changed.
	ErrInvalidDecodedValue = errors.New("invalid decoded value type")
)

// StatusMessageContent contains the chat ID and the actual text of a message.
type StatusMessageContent struct {
	ChatID string
	Text   string
}

// StatusMessage contains all message details.
type StatusMessage struct {
	Text      string // TODO: why is this duplicated?
	ContentT  string
	MessageT  string
	Clock     int64
	Timestamp int64
	Content   StatusMessageContent
}

// DecodeMessage decodes a raw payload to StatusMessage struct.
func DecodeMessage(data []byte) (message StatusMessage, err error) {
	buf := bytes.NewBuffer(data)
	decoder := NewMessageDecoder(buf)
	value, err := decoder.Decode()
	if err != nil {
		return
	}

	message, ok := value.(StatusMessage)
	if !ok {
		return message, ErrInvalidDecodedValue
	}
	return
}

// EncodeMessage encodes a StatusMessage using Transit serialization.
func EncodeMessage(value StatusMessage) ([]byte, error) {
	var buf bytes.Buffer
	encoder := NewMessageEncoder(&buf)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
