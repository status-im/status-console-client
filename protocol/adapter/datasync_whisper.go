package adapter

import (
	"context"

	"github.com/status-im/status-console-client/protocol/subscription"
	"github.com/status-im/status-console-client/protocol/transport"
	"github.com/status-im/status-console-client/protocol/v1"
)

type DataSyncWhisperAdapter struct {
	node *node.Node
	transport *DataSyncWhisperTransport
}

func NewDataSyncWhisperAdapter(n *node.Node, t *DataSyncWhisperTransport) *DataSyncWhisperAdapter {
	return &DataSyncWhisperAdapter{node: n, transport: t}
}

// MVDSWhisperAdapter must implement Protocol interface.
var _ protocol.Protocol = (*DataSyncWhisperAdapter)(nil)

// Subscribe listens to new messages.
func (m *DataSyncWhisperAdapter) Subscribe(
	ctx context.Context,
	messages chan<- *protocol.Message,
	options protocol.SubscribeOptions,
) (*subscription.Subscription, error) {
	return c.t.subscribe(messages, options)
}

// Send sends a message to the network.
// Identity is required as the protocol requires
// all messages to be signed.
func (m *DataSyncWhisperAdapter) Send(ctx context.Context, data []byte, options protocol.SendOptions) ([]byte, error) {
	if err := options.Validate(); err != nil {
		return nil, err
	}

	if options.ChatName == "" {
		return nil, errors.New("missing chat name")
	}

	topic, err := ToTopic(options.ChatName)
	if err != nil {
		return nil, err
	}

	gid := toGroupId(topic)

	c.peer(gid, options.Recipient)

	id, err := c.node.AppendMessage(gid, data)
	if err != nil {
		return nil, err
	}

	return id[:], nil
}

// Request retrieves historic messages.
func (m *DataSyncWhisperAdapter) Request(ctx context.Context, params protocol.RequestOptions) error {
	return nil
}

func (c *DataSyncWhisperAdapter) peer(id state.GroupID, peer *ecdsa.PublicKey) {
	if peer == nil {
		return
	}

	p := PublicKeyToPeerID(*peer)

	if c.node.IsPeerInGroup(id, p) {
		return
	}

	c.node.AddPeer(id, p)
}
