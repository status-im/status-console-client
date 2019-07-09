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
	clientmock "github.com/status-im/status-console-client/protocol/client/mock"
	"github.com/status-im/status-console-client/protocol/v1"
	protomock "github.com/status-im/status-console-client/protocol/v1/mock"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestPublicAPISend(t *testing.T) {
	ctrlProto := gomock.NewController(t)
	defer ctrlProto.Finish()
	proto := protomock.NewMockProtocol(ctrlProto)

	ctrlDatabase := gomock.NewController(t)
	defer ctrlDatabase.Finish()
	database := clientmock.NewMockDatabase(ctrlDatabase)

	client, aNode, err := setupRPCClient(database, proto)
	require.NoError(t, err)
	defer func() { go discardStop(aNode) }() // Stop() is slow so do it in a goroutine

	data := []byte("some payload")
	contact := Contact{
		Name: "test-chat",
	}
	result := hexutil.Bytes("abc")

	database.EXPECT().
		LastMessageClock(gomock.Any()).
		Return(time.Now().Unix(), nil)

	database.EXPECT().
		SaveMessages(gomock.Any(), gomock.Any()).
		Return(int64(0), nil)

	proto.EXPECT().
		Send(
			gomock.Any(),
			gomock.Any(),
			gomock.Eq(protocol.SendOptions{
				ChatOptions: protocol.ChatOptions{
					ChatName: contact.Name,
				},
			}),
		).
		Return(result, nil)

	var hash hexutil.Bytes
	err = client.Call(&hash, createRPCMethod("send"), contact, hexutil.Encode(data))
	require.NoError(t, err)
	require.Equal(t, result, hash)
}

func TestPublicAPIRequest(t *testing.T) {
	ctrlProto := gomock.NewController(t)
	defer ctrlProto.Finish()
	proto := protomock.NewMockProtocol(ctrlProto)

	ctrlDatabase := gomock.NewController(t)
	defer ctrlDatabase.Finish()
	database := clientmock.NewMockDatabase(ctrlDatabase)

	client, aNode, err := setupRPCClient(database, proto)
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
	ctrlProto := gomock.NewController(t)
	defer ctrlProto.Finish()
	proto := protomock.NewMockProtocol(ctrlProto)

	ctrlDatabase := gomock.NewController(t)
	defer ctrlDatabase.Finish()
	database := clientmock.NewMockDatabase(ctrlDatabase)

	rpcClient, aNode, err := setupRPCClient(database, proto)
	require.NoError(t, err)
	defer func() { go discardStop(aNode) }() // Stop() is slow so do it in a goroutine

	proto.EXPECT().
		LoadChats(
			gomock.Any(),
			gomock.Eq([]protocol.ChatOptions{
				protocol.ChatOptions{ChatName: "test-chat"},
			}),
		).
		Return(nil)

	proto.EXPECT().
		Request(gomock.Any(), gomock.Any()).
		Return(nil)

	database.EXPECT().
		UpdateHistories(gomock.Any()).
		Return(nil)

	messages := make(chan protocol.Message)
	// The first argument is a name of the method to use for subscription.
	_, err = rpcClient.Subscribe(context.Background(), StatusSecureMessagingProtocolAPIName, messages, "messages", client.Contact{Type: 1, Name: "test-chat"})
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

func setupRPCClient(db client.Database, proto protocol.Protocol) (*rpc.Client, *node.StatusNode, error) {
	privateKey, _ := crypto.GenerateKey()

	n, service, err := createAndStartNode(privateKey)
	if err != nil {
		return nil, nil, err
	}

	service.SetMessenger(client.NewMessenger(privateKey, proto, db))

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
