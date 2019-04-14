package protocol

import (
	"bytes"
	"crypto/ecdsa"
	"errors"
	"strings"
	"time"
)

const (
	// ContentTypeTextPlain means that the message contains plain text.
	ContentTypeTextPlain = "text/plain"
)

// Message types.
const (
	MessageTypePublicGroup = "public-group-user-message"
	MessageTypePrivate     = "user-message"
)

var (
	// ErrInvalidDecodedValue means that the decoded message is of wrong type.
	// This might mean that the status message serialization tag changed.
	ErrInvalidDecodedValue = errors.New("invalid decoded value type")
)

// Content contains the chat ID and the actual text of a message.
type Content struct {
	ChatID string `json:"chat_id"`
	Text   string `json:"text"`
}

// Message contains all message details.
type Message struct {
	Text      string  `json:"text"` // TODO: why is this duplicated?
	ContentT  string  `json:"content_type"`
	MessageT  string  `json:"message_type"`
	Clock     int64   `json:"clock"`     // in milliseconds; see CalcMessageClock for more details
	Timestamp int64   `json:"timestamp"` // in milliseconds
	Content   Content `json:"content"`

	// not protocol defined fields
	ID        []byte           `json:"id"`
	SigPubKey *ecdsa.PublicKey `json:"-"`
}

// CreateTextMessage creates a Message.
func CreateTextMessage(data []byte, lastClock int64, chatID, messageType string) Message {
	text := strings.TrimSpace(string(data))
	ts := time.Now().Unix() * 1000
	clock := CalcMessageClock(lastClock, ts)

	return Message{
		Text:      text,
		ContentT:  ContentTypeTextPlain,
		MessageT:  messageType,
		Clock:     clock,
		Timestamp: ts,
		Content:   Content{ChatID: chatID, Text: text},
	}
}

// CreatePublicTextMessage creates a public text Message.
func CreatePublicTextMessage(data []byte, lastClock int64, chatID string) Message {
	return CreateTextMessage(data, lastClock, chatID, MessageTypePublicGroup)
}

// CreatePrivateTextMessage creates a public text Message.
func CreatePrivateTextMessage(data []byte, lastClock int64, chatID string) Message {
	return CreateTextMessage(data, lastClock, chatID, MessageTypePrivate)
}

// DecodeMessage decodes a raw payload to Message struct.
func DecodeMessage(data []byte) (message Message, err error) {
	buf := bytes.NewBuffer(data)
	decoder := NewMessageDecoder(buf)
	value, err := decoder.Decode()
	if err != nil {
		return
	}

	message, ok := value.(Message)
	if !ok {
		return message, ErrInvalidDecodedValue
	}
	return
}

// EncodeMessage encodes a Message using Transit serialization.
func EncodeMessage(value Message) ([]byte, error) {
	var buf bytes.Buffer
	encoder := NewMessageEncoder(&buf)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
