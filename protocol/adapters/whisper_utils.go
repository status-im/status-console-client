package adapters

import (
	"github.com/status-im/status-console-client/protocol/v1"
	"github.com/status-im/status-go/services/shhext"
)

func createShhextRequestMessagesParam(enode, mailSymKeyID string, options protocol.RequestOptions) (shhext.MessagesRequest, error) {
	req := shhext.MessagesRequest{
		MailServerPeer: enode,
		From:           uint32(options.From),  // TODO: change to int in status-go
		To:             uint32(options.To),    // TODO: change to int in status-go
		Limit:          uint32(options.Limit), // TODO: change to int in status-go
		SymKeyID:       mailSymKeyID,
	}

	for _, chatOpts := range options.Chats {
		topic, err := topicForRequestOptions(chatOpts)
		if err != nil {
			return req, err
		}
		req.Topics = append(req.Topics, topic)
	}

	return req, nil
}
