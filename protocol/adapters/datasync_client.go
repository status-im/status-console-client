package adapters

import (
	"context"
	"encoding/hex"

	"github.com/status-im/mvds"
	"github.com/status-im/status-console-client/protocol/v1"
	"github.com/status-im/status-go/node"
	whisper "github.com/status-im/whisper/whisperv6"
)

// DataSyncClient is an adapter for MVDS
// that implements the Protocol interface.
type DataSyncClient struct {
	sync mvds.Node

	node *node.StatusNode // TODO: replace with an interface
	shh  *whisper.Whisper
}

func (*DataSyncClient) Subscribe(ctx context.Context, messages chan<- *protocol.Message, options protocol.SubscribeOptions) (*protocol.Subscription, error) {
	panic("implement me")
}

func (c *DataSyncClient) Send(ctx context.Context, data []byte, options protocol.SendOptions) ([]byte, error) {

	if err := options.Validate(); err != nil {
		return nil, err
	}

	newMessage, err := newNewMessage(a.keysManager, data) // @todo
	if err != nil {
		return nil, err
	}
	if err := updateNewMessageFromSendOptions(newMessage, options); err != nil {
		return nil, err
	}

	id, err := c.sync.AppendMessage(toGroupId(newMessage.Topic), newMessage.MarshalJSON())
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
