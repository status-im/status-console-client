package adapters

import (
	"context"
	"crypto/ecdsa"
	"sync"

	"github.com/status-im/mvds"
	"github.com/status-im/status-console-client/protocol/v1"
	"github.com/status-im/status-go/node"
	whisper "github.com/status-im/whisper/whisperv6"
)

type dataSyncClientKeysManager struct {
	shh *whisper.Whisper

	// Identity of the current user.
	// It must be the same private key
	// that is used in the PFS service.
	privateKey *ecdsa.PrivateKey

	passToSymKeyMutex sync.RWMutex
	passToSymKeyCache map[string]string
}

func (m *dataSyncClientKeysManager) PrivateKey() *ecdsa.PrivateKey {
	return m.privateKey
}

func (m *dataSyncClientKeysManager) AddOrGetKeyPair(priv *ecdsa.PrivateKey) (string, error) {
	// caching is handled in Whisper
	return m.shh.AddKeyPair(priv)
}

func (m *dataSyncClientKeysManager) AddOrGetSymKeyFromPassword(password string) (string, error) {
	m.passToSymKeyMutex.Lock()
	defer m.passToSymKeyMutex.Unlock()

	if val, ok := m.passToSymKeyCache[password]; ok {
		return val, nil
	}

	id, err := m.shh.AddSymKeyFromPassword(password)
	if err != nil {
		return id, err
	}

	m.passToSymKeyCache[password] = id

	return id, nil
}

func (m *dataSyncClientKeysManager) GetRawSymKey(id string) ([]byte, error) {
	return m.shh.GetSymKey(id)
}

// DataSyncClient is an adapter for MVDS
// that implements the Protocol interface.
type DataSyncClient struct {
	sync mvds.Node

	node        *node.StatusNode // TODO: replace with an interface
	shh         *whisper.Whisper
	keysManager *dataSyncClientKeysManager
}

func (*DataSyncClient) Subscribe(ctx context.Context, messages chan<- *protocol.Message, options protocol.SubscribeOptions) (*protocol.Subscription, error) {
	panic("implement me")
}

func (c *DataSyncClient) Send(ctx context.Context, data []byte, options protocol.SendOptions) ([]byte, error) {
	if err := options.Validate(); err != nil {
		return nil, err
	}

	newMessage, err := newNewMessage(c.keysManager, data)
	if err != nil {
		return nil, err
	}

	if err := updateNewMessageFromSendOptions(newMessage, options); err != nil {
		return nil, err
	}

	msg, err := newMessage.MarshalJSON()
	if err != nil {
		return nil, err
	}

	id, err := c.sync.AppendMessage(toGroupId(newMessage.Topic), msg)
	if err != nil {
		return nil, err
	}

	return id[:], nil
}

func (*DataSyncClient) Request(ctx context.Context, params protocol.RequestOptions) error {
	panic("implement me")
}

func toGroupId(topicType whisper.TopicType) mvds.GroupID {
	g := mvds.GroupID{}
	copy(g[:], topicType[:])
	return g
}
