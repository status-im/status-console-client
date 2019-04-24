package adapters

import (
	"errors"

	"github.com/status-im/status-console-client/protocol/v1"
	"github.com/status-im/status-go/services/shhext"
	whisper "github.com/status-im/whisper/whisperv6"
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
		topic, err := topicForChatOptions(chatOpts)
		if err != nil {
			return req, err
		}
		req.Topics = append(req.Topics, topic)
	}

	return req, nil
}

func topicForChatOptions(options protocol.ChatOptions) (whisper.TopicType, error) {
	if options.Recipient != nil {
		return PrivateChatTopic()
	}

	if options.ChatName != "" {
		return PublicChatTopic(options.ChatName)
	}

	return whisper.TopicType{}, errors.New("invalid options")
}
