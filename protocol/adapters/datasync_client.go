package adapters

import (
	"context"
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gogo/protobuf/proto"
	"github.com/status-im/mvds"
	"github.com/status-im/status-console-client/protocol/v1"
	whisper "github.com/status-im/whisper/whisperv6"
)

// DataSyncClient is an adapter for MVDS
// that implements the Protocol interface.
type DataSyncClient struct {
	sync mvds.Node
}

// Subscribe subscribes to a public chat using the Whisper service.
func (*DataSyncClient) Subscribe(ctx context.Context, messages chan<- *protocol.Message, options protocol.SubscribeOptions) (*protocol.Subscription, error) {
	panic("implement me")
}

// Send appends a message to the data sync node for later sending.
func (c *DataSyncClient) Send(ctx context.Context, data []byte, options protocol.SendOptions) ([]byte, error) {
	if err := options.Validate(); err != nil {
		return nil, err
	}

	topic, err := topic(options)
	if err != nil {
		return nil, err
	}

	id, err := c.sync.AppendMessage(toGroupId(topic), data)
	if err != nil {
		return nil, err
	}

	return id[:], nil
}

func (*DataSyncClient) Request(ctx context.Context, params protocol.RequestOptions) error {
	panic("implement me")
}

type DataSyncWhisperTransport struct {
	shh         *whisper.Whisper
	keysManager *whisperClientKeysManager
}

func (*DataSyncWhisperTransport) Watch() mvds.Packet {
	panic("implement me")
}

// Send sends a new message using the Whisper service.
func (t *DataSyncWhisperTransport) Send(group mvds.GroupID, _ mvds.PeerId, peer mvds.PeerId, payload mvds.Payload) error {
	data, err := proto.Marshal(&payload)

	newMessage, err := newNewMessage(t.keysManager, data)
	if err != nil {
		return err
	}

	newMessage.Topic = toTopicType(group)

	// @todo set SymKeyID or PublicKey depending on chat type

	// we are only assuming private chats
	k := ecdsa.PublicKey(peer)
	newMessage.PublicKey = crypto.FromECDSAPub(&k)

	_, err = whisper.NewPublicWhisperAPI(t.shh).Post(context.Background(), newMessage.ToWhisper())
	return err
}


func toGroupId(topicType whisper.TopicType) mvds.GroupID {
	g := mvds.GroupID{}
	copy(g[:], topicType[:])
	return g
}

func toTopicType(g mvds.GroupID) whisper.TopicType {
	t := whisper.TopicType{}
	copy(t[:], g[:4])
	return t
}
