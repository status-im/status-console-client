package main

import (
	"context"
	"crypto/ecdsa"
	"testing"

	"github.com/status-im/status-console-client/protocol/v1"

	"github.com/stretchr/testify/require"
)

type message struct {
	chat string
	dest *ecdsa.PublicKey
	data []byte
}

type PublicChatMock struct {
	messages []message
}

func (m *PublicChatMock) SubscribePublicChat(
	ctx context.Context,
	name string,
	in chan<- *protocol.ReceivedMessage,
) (*protocol.Subscription, error) {
	return protocol.NewSubscription(), nil
}

func (m *PublicChatMock) SendPublicMessage(
	ctx context.Context,
	chatName string,
	data []byte,
	identity *ecdsa.PrivateKey,
) (string, error) {
	m.messages = append(m.messages, message{chatName, nil, data})
	return "", nil
}

func (m *PublicChatMock) RequestPublicMessages(
	ctx context.Context,
	chatName string,
	params protocol.RequestMessagesParams,
) error {
	return nil
}

type PrivateChatMock struct {
	messages []message
}

func (m *PrivateChatMock) SubscribePrivateChat(
	ctx context.Context,
	identity *ecdsa.PrivateKey,
	in chan<- *protocol.ReceivedMessage,
) (*protocol.Subscription, error) {
	return protocol.NewSubscription(), nil
}

func (m *PrivateChatMock) SendPrivateMessage(
	ctx context.Context,
	recipient *ecdsa.PublicKey,
	data []byte,
	identity *ecdsa.PrivateKey,
) (string, error) {
	m.messages = append(m.messages, message{"", recipient, data})
	return "", nil
}

func (m *PrivateChatMock) RequestPrivateMessages(
	ctx context.Context,
	params protocol.RequestMessagesParams,
) error {
	return nil
}

type ChatMock struct {
	*PublicChatMock
	*PrivateChatMock
}

func TestSendMessage(t *testing.T) {
	chatName := "test-chat"
	payload := []byte("test message")
	chatMock := ChatMock{
		&PublicChatMock{},
		&PrivateChatMock{},
	}
	vc, err := NewChatViewController(nil, nil, &chatMock, nil)
	require.NoError(t, err)
	vc.currentContact = Contact{
		Name: chatName,
		Type: ContactPublicChat,
	}

	_, err = vc.SendMessage(payload)
	require.NoError(t, err)

	message := chatMock.PublicChatMock.messages[0]
	require.Equal(t, chatName, message.chat)
	statusMessage, err := protocol.DecodeMessage(message.data)
	require.NoError(t, err)
	require.EqualValues(t, payload, statusMessage.Text)
	require.Equal(t, protocol.ContentTypeTextPlain, statusMessage.ContentT)
	require.Equal(t, protocol.MessageTypePublicGroupUserMessage, statusMessage.MessageT)
	require.Equal(t,
		protocol.StatusMessageContent{ChatID: chatName, Text: string(payload)},
		statusMessage.Content)
}
