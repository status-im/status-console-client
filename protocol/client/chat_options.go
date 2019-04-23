package client

import (
	"fmt"

	"github.com/status-im/status-console-client/protocol/v1"
)

var (
	errUnsupportedContactType = fmt.Errorf("unsupported contact type")
)

func createSubscribeOptions(c Contact) (opts protocol.SubscribeOptions, err error) {
	switch c.Type {
	case ContactPublicRoom:
		opts.ChatName = c.Name
	case ContactPublicKey:
		opts.Recipient = c.PublicKey
	default:
		err = errUnsupportedContactType
	}
	return
}

func createSendOptions(c Contact) (opts protocol.SendOptions, err error) {
	switch c.Type {
	case ContactPublicRoom:
		opts.ChatName = c.Name
	case ContactPublicKey:
		opts.Recipient = c.PublicKey
	default:
		err = errUnsupportedContactType
	}
	return
}

func enhanceRequestOptions(c Contact, opts *protocol.RequestOptions) error {
	var chatOptions protocol.ChatOptions

	switch c.Type {
	case ContactPublicRoom:
		chatOptions.ChatName = c.Name
	case ContactPublicKey:
		chatOptions.Recipient = c.PublicKey
	default:
		return errUnsupportedContactType
	}

	opts.Chats = append(opts.Chats, chatOptions)

	return nil
}
