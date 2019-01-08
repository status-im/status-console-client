package main

import (
	"crypto/ecdsa"
	"testing"

	"github.com/status-im/status-term-client/protocol/v1"

	"github.com/stretchr/testify/require"
)

type message struct {
	chat string
	data []byte
}

type PublicChatMock struct {
	messages []message
}

func (m *PublicChatMock) SubscribePublicChat(name string) (MessagesSubscription, error) {
	return &WhisperSubscription{}, nil
}

func (m *PublicChatMock) SendPublicMessage(chatName string, data []byte, identity *ecdsa.PrivateKey) (string, error) {
	m.messages = append(m.messages, message{chatName, data})
	return "", nil
}

type ChatMock struct {
	*PublicChatMock
}

func (m *ChatMock) RequestPublicMessages(chatName string, params RequestMessagesParams) error {
	return nil
}

func TestSendMessage(t *testing.T) {
	chatName := "test-chat"
	payload := []byte("test message")
	chatMock := ChatMock{
		&PublicChatMock{},
	}
	vc, err := NewChatViewController(nil, nil, &chatMock)
	require.NoError(t, err)
	vc.currentContact = Contact{chatName, ContactPublicChat}

	_, err = vc.SendMessage(payload)
	require.NoError(t, err)

	message := chatMock.messages[0]
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
