package adapter

import (
	"context"

	"github.com/status-im/status-console-client/protocol/subscription"
	"github.com/status-im/status-console-client/protocol/transport"
	"github.com/status-im/status-console-client/protocol/v1"
)

type MVDSWhisperAdapter struct {
	transport transport.WhisperTransport //nolint: structcheck,unused
	// TODO: probably some *mvds.MVDS struct implementing mvds.
}

// MVDSWhisperAdapter must implement Protocol interface.
var _ protocol.Protocol = (*MVDSWhisperAdapter)(nil)

// Subscribe listens to new messages.
func (m *MVDSWhisperAdapter) Subscribe(
	ctx context.Context,
	messages chan<- *protocol.Message,
	options protocol.SubscribeOptions,
) (*subscription.Subscription, error) {
	return nil, nil
}

// Send sends a message to the network.
// Identity is required as the protocol requires
// all messages to be signed.
func (m *MVDSWhisperAdapter) Send(ctx context.Context, data []byte, options protocol.SendOptions) ([]byte, error) {
	return nil, nil
}

// Request retrieves historic messages.
func (m *MVDSWhisperAdapter) Request(ctx context.Context, params protocol.RequestOptions) error {
	return nil
}
