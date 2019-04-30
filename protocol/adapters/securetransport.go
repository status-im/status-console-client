package adapters

import (
	"github.com/status-im/mvds"
	"github.com/status-im/status-console-client/protocol/v1"
)

type SecureTransport struct {
	p protocol.Protocol
}

func (st *SecureTransport) SendPayload(senderId mvds.PeerId, to mvds.PeerId, payload mvds.Payload) error {
	//st.p.Send(context.Background(), )
	return nil
}

