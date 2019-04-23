package client

import (
	"context"
	"crypto/ecdsa"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/status-im/status-console-client/protocol/v1"
	"github.com/stretchr/testify/require"
)

const (
	testPubKey = "0x047d036c25b97a377df74ca4f1780369b1f5475cb58b95d8683cce7f7cfd832271072c18ebf75d09b1c04ae066efcf46b10e14bda83fc220b39ae3dece38f91993"
)

var (
	timeZero = time.Unix(0, 0)
)

type message struct {
	chat string
	dest *ecdsa.PublicKey
	data []byte
}

type ChatMock struct {
	input    chan<- *protocol.Message
	messages []message
}

func (m *ChatMock) Subscribe(
	ctx context.Context,
	messages chan<- *protocol.Message,
	options protocol.SubscribeOptions,
) (*protocol.Subscription, error) {
	m.input = messages
	return protocol.NewSubscription(), nil
}

func (m *ChatMock) Send(
	ctx context.Context,
	data []byte,
	options protocol.SendOptions,
) ([]byte, error) {
	message := message{
		chat: options.ChatName,
		dest: options.Recipient,
		data: data,
	}
	m.messages = append(m.messages, message)
	return []byte{}, nil
}

func (m *ChatMock) Request(ctx context.Context, params protocol.RequestOptions) error {
	return nil
}

func TestSubscribe(t *testing.T) {
	proto := ChatMock{}
	contact := Contact{Name: "test", Type: ContactPublicRoom}

	db, err := NewDatabase("")
	require.NoError(t, err)
	defer db.Close()

	chat := NewChat(&proto, nil, contact, db)

	err = chat.subscribe()
	require.NoError(t, err)
	// Subscribe to already subscribed chat.
	err = chat.subscribe()
	require.EqualError(t, err, "already subscribed")
}

func TestSendPrivateMessage(t *testing.T) {
	proto := ChatMock{}
	contact, err := ContactWithPublicKey("contact1", testPubKey)
	require.NoError(t, err)

	identity, _ := crypto.GenerateKey()

	db, err := NewDatabase("")
	require.NoError(t, err)
	defer db.Close()

	chat := NewChat(&proto, identity, contact, db)

	// act
	err = chat.Send([]byte("some message"))
	require.NoError(t, err)

	// assert
	waitForEventTypeMessage(t, chat)
	require.Len(t, chat.Messages(), 1)

	// the message should be also saved in the database
	result, err := db.Messages(contact, timeZero, time.Now())
	require.NoError(t, err)
	require.Len(t, result, 1)

	// clock should be updated
	require.NotZero(t, chat.lastClock)
}

func TestHandleMessageFromProtocol(t *testing.T) {
	proto := ChatMock{}
	contact := Contact{Name: "chat1", Type: ContactPublicRoom}

	db, err := NewDatabase("")
	require.NoError(t, err)
	defer db.Close()

	chat := NewChat(&proto, nil, contact, db)

	// act
	err = chat.subscribe()
	require.NoError(t, err)

	now := time.Now()
	ts := protocol.TimestampInMsFromTime(now)
	message := &protocol.Message{
		ID:        []byte{0x01, 0x02, 0x03},
		Text:      "some",
		ContentT:  protocol.ContentTypeTextPlain,
		MessageT:  protocol.MessageTypePublicGroup,
		Timestamp: ts,
		Clock:     int64(ts),
	}
	proto.input <- message

	// assert
	waitForEventTypeMessage(t, chat)
	require.Len(t, chat.Messages(), 1)
	require.True(t, chat.HasMessage(message))

	// the message should be also saved in the database
	result, err := db.Messages(contact, timeZero, now)
	require.NoError(t, err)
	require.Len(t, result, 1)

	// clock should be updated
	require.Equal(t, int64(ts), chat.lastClock)
}

func waitForEventTypeMessage(t *testing.T, chat *Chat) {
	for {
		select {
		case ev := <-chat.Events():
			if v, ok := ev.(Event); ok && v.Type() == EventTypeMessage {
				return
			}
		case <-time.After(time.Millisecond * 100):
			require.NoError(t, chat.Err())
			t.Fatalf("timed out")
		}
	}
}
