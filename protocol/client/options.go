package client

import (
	"fmt"

	"github.com/status-im/status-console-client/protocol/v1"
)

var (
	errUnsupportedChatType = fmt.Errorf("unsupported chat type")
)

func createSendOptions(c Chat) (opts protocol.SendOptions, err error) {
	opts.ChatName = c.Name
	switch c.Type {
	case PublicChat:
	case OneToOneChat:
		opts.Recipient = c.PublicKey
	default:
		err = errUnsupportedChatType
	}
	return
}

func enhanceRequestOptions(c Chat, opts *protocol.RequestOptions) error {
	var chatOptions protocol.ChatOptions
	chatOptions.ChatName = c.Name
	switch c.Type {
	case PublicChat:
	case OneToOneChat:
		chatOptions.Recipient = c.PublicKey
	default:
		return errUnsupportedChatType
	}

	opts.Chats = append(opts.Chats, chatOptions)

	return nil
}
