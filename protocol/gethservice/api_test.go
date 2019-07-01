package gethservice

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	gethnode "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/node"
	"github.com/status-im/status-go/params"

	"github.com/status-im/status-console-client/protocol/client"
	"github.com/status-im/status-console-client/protocol/subscription"
	"github.com/status-im/status-console-client/protocol/v1"
	protomock "github.com/status-im/status-console-client/protocol/v1/mock"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestPublicAPISend(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	proto := protomock.NewMockProtocol(ctrl)

	client, aNode, err := setupRPCClient(proto)
	require.NoError(t, err)
	defer func() { go discardStop(aNode) }() // Stop() is slow so do it in a goroutine

	data := []byte("some payload")
	params := SendParams{
		Contact{
			Name: "test-chat",
		},
	}
	result := hexutil.Bytes("abc")

	proto.EXPECT().
		Send(
			gomock.Any(),
			gomock.Eq(data),
			gomock.Eq(protocol.SendOptions{
				ChatOptions: protocol.ChatOptions{
					ChatName: params.Name,
				},
			}),
		).
		Return(result, nil)

	var hash hexutil.Bytes
	err = client.Call(&hash, createRPCMethod("send"), hexutil.Encode(data), params)
	require.NoError(t, err)
	require.Equal(t, result, hash)
}

func TestPublicAPIRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	proto := protomock.NewMockProtocol(ctrl)

	client, aNode, err := setupRPCClient(proto)
	require.NoError(t, err)
	defer func() { go discardStop(aNode) }() // Stop() is slow so do it in a goroutine

	now := time.Now().Unix()
	params := RequestParams{
		Contact: Contact{
			Name: "test-chat",
		},
		Limit: 100,
		From:  now,
		To:    now,
	}

	proto.EXPECT().
		Request(
			gomock.Any(),
			gomock.Eq(protocol.RequestOptions{
				Chats: []protocol.ChatOptions{
					protocol.ChatOptions{
						ChatName: params.Name,
					},
				},
				Limit: 100,
				From:  now,
				To:    now,
			}),
		).
		Return(nil)

	// nil skips the result... because there is no result
	err = client.Call(nil, createRPCMethod("request"), params)
	require.NoError(t, err)
}

func TestPublicAPIMessages(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	proto := protomock.NewMockProtocol(ctrl)

	client, aNode, err := setupRPCClient(proto)
	require.NoError(t, err)
	defer func() { go discardStop(aNode) }() // Stop() is slow so do it in a goroutine

	messages := make(chan protocol.Message)
	params := MessagesParams{
		Contact{
			Name: "test-chat",
		},
	}

	proto.EXPECT().
		Subscribe(
			gomock.Any(),
			gomock.Any(),
			gomock.Eq(protocol.SubscribeOptions{
				ChatOptions: protocol.ChatOptions{
					ChatName: params.Name,
				},
			}),
		).
		Return(subscription.New(), nil)

	// The first argument is a name of the method to use for subscription.
	_, err = client.Subscribe(context.Background(), StatusSecureMessagingProtocolAPIName, messages, "messages", params)
	require.NoError(t, err)
}

func createAndStartNode(privateKey *ecdsa.PrivateKey) (*node.StatusNode, *Service, error) {
	n := node.New()
	service := New(n, &keysGetter{privateKey: privateKey})

	services := []gethnode.ServiceConstructor{
		func(*gethnode.ServiceContext) (gethnode.Service, error) {
			return service, nil
		},
		// func(*gethnode.ServiceContext) (gethnode.Service, error) {
		//	config := &whisper.Config{
		//		MinimumAcceptedPOW: 0.001,
		//		MaxMessageSize:     whisper.DefaultMaxMessageSize,
		//	}
		//	return whisper.New(config), nil
		// },
	}

	return n, service, n.Start(
		&params.NodeConfig{APIModules: StatusSecureMessagingProtocolAPIName},
		services...,
	)
}

func discardStop(n *node.StatusNode) {
	_ = n.Stop()
}

func setupRPCClient(proto protocol.Protocol) (*rpc.Client, *node.StatusNode, error) {
	privateKey, _ := crypto.GenerateKey()

	n, service, err := createAndStartNode(privateKey)
	if err != nil {
		return nil, nil, err
	}

	service.SetMessenger(client.NewMessenger(nil, proto, nil))
	service.SetProtocol(proto)

	client, err := n.GethNode().Attach()
	return client, n, err
}

func createRPCMethod(name string) string {
	return fmt.Sprintf("%s_%s", StatusSecureMessagingProtocolAPIName, name)
}

type keysGetter struct {
	privateKey *ecdsa.PrivateKey
}

func (k keysGetter) PrivateKey() (*ecdsa.PrivateKey, error) {
	return k.privateKey, nil
}
