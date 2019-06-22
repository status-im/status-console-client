package adapter

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-console-client/protocol/transport"
	"github.com/status-im/status-console-client/protocol/v1"
	transmock "github.com/status-im/status-console-client/protocol/transport/mock"

	whisper "github.com/status-im/whisper/whisperv6"
)

func TestRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	transMock := transmock.NewMockWhisperTransport(ctrl)

	topic, err := ToTopic("test")
	require.NoError(t, err)

	reqOptions := transport.RequestOptions{
		Topics:   []whisper.TopicType{topic},
		Password: MailServerPassword,
		From:     10,
		To:       20,
		Limit:    5,
	}

	transMock.EXPECT().
		Request(gomock.Any(), reqOptions).
		Return(nil).
		Times(1)

	a := NewProtocolWhisperAdapter(transMock, nil)
	err = a.Request(context.TODO(), protocol.RequestOptions{
		Chats: []protocol.ChatOptions{
			protocol.ChatOptions{ChatName: "test"},
		},
		From: 10,
		To: 20,
		Limit: 5,
	})
	require.NoError(t, err)
}
