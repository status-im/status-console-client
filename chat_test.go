package main

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/status-im/status-console-client/protocol/client"
	protomock "github.com/status-im/status-console-client/protocol/v1/mock"
)

func TestSendMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	protoMock := protomock.NewMockProtocol(ctrl)

	chatName := "test-chat"
	payload := []byte("test message")
	identity, err := crypto.GenerateKey()
	require.NoError(t, err)

	db, err := client.InitializeTmpDB()
	require.NoError(t, err)
	defer db.Close()

	messenger := client.NewMessenger(identity, protoMock, db)
	vc := NewChatViewController(nil, nil, messenger, nil)

	protoMock.EXPECT().
		LoadChats(
			gomock.Any(),
			gomock.Any(),
		).
		Return(nil).
		Times(1)

	protoMock.EXPECT().
		Request(
			gomock.Any(),
			gomock.Any(),
		).
		Return(nil).
		Times(1)

	err = vc.Select(client.Contact{
		Name:  chatName,
		Type:  client.ContactPublicRoom,
		Topic: chatName,
	})
	require.NoError(t, err)
	// close reading loops
	close(vc.cancel)

	var sendPayload = []byte{}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	protoMock.EXPECT().
		Send(
			gomock.AssignableToTypeOf(ctx),
			gomock.AssignableToTypeOf(sendPayload),
			gomock.Any(),
		).
		Return([]byte{0x01}, nil).
		Times(1)

	err = vc.Send(payload)
	require.NoError(t, err)

	// TODO: move to another layer
	// statusMessage, err := protocol.DecodeMessage(message.data)
	// require.NoError(t, err)
	// require.EqualValues(t, payload, statusMessage.Text)
	// require.Equal(t, protocol.ContentTypeTextPlain, statusMessage.ContentT)
	// require.Equal(t, protocol.MessageTypePublicGroup, statusMessage.MessageT)
	// require.Equal(t,
	// 	protocol.Content{ChatID: chatName, Text: string(payload)},
	// 	statusMessage.Content)
}
