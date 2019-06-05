package main

import (
	"context"
	"crypto/ecdsa"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/status-im/status-console-client/protocol/client"
	"github.com/status-im/status-console-client/protocol/v1"
	"github.com/stretchr/testify/require"
)

type message struct {
	chat string
	dest *ecdsa.PublicKey
	data []byte
}

type ChatMock struct {
	messages []message
}

func (m *ChatMock) Subscribe(
	ctx context.Context,
	messages chan<- *protocol.Message,
	options protocol.SubscribeOptions,
) (*protocol.Subscription, error) {
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
	return crypto.Keccak256(data), nil
}

func (m *ChatMock) Request(ctx context.Context, params protocol.RequestOptions) error {
	return nil
}

func TestSendMessage(t *testing.T) {
	chatName := "test-chat"
	payload := []byte("test message")
	chatMock := ChatMock{}
	identity, err := crypto.GenerateKey()
	require.NoError(t, err)

	db, err := client.InitializeTmpDB()
	require.NoError(t, err)
	defer db.Close()

	messenger := client.NewMessenger(identity, &chatMock, db)
	vc := NewChatViewController(nil, nil, messenger, nil)

	err = vc.Select(client.Contact{Name: chatName, Type: client.ContactPublicRoom, Topic: chatName})
	require.NoError(t, err)
	// close reading loops
	close(vc.cancel)

	err = vc.Send(payload)
	require.NoError(t, err)

	message := chatMock.messages[0]
	require.Equal(t, chatName, message.chat)
	statusMessage, err := protocol.DecodeMessage(message.data)
	require.NoError(t, err)
	require.EqualValues(t, payload, statusMessage.Text)
	require.Equal(t, protocol.ContentTypeTextPlain, statusMessage.ContentT)
	require.Equal(t, protocol.MessageTypePublicGroup, statusMessage.MessageT)
	require.Equal(t,
		protocol.Content{ChatID: chatName, Text: string(payload)},
		statusMessage.Content)
}
