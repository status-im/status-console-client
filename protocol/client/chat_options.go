package client

import (
	"fmt"

	"github.com/status-im/status-console-client/protocol/v1"
)

var (
	errUnsupportedContactType = fmt.Errorf("unsupported contact type")
)

func extendSubscribeOptions(opts protocol.SubscribeOptions, c *Chat) (protocol.SubscribeOptions, error) {
	switch c.contact.Type {
	case ContactPublicChat:
		opts.ChatName = c.contact.Name
	case ContactPrivateChat:
		opts.Recipient = c.contact.PublicKey
	default:
		return opts, errUnsupportedContactType
	}
	return opts, nil
}

func extendRequestOptions(opts protocol.RequestOptions, c *Chat) (protocol.RequestOptions, error) {
	switch c.contact.Type {
	case ContactPublicChat:
		opts.ChatName = c.contact.Name
	case ContactPrivateChat:
		opts.Recipient = c.contact.PublicKey
	default:
		return opts, errUnsupportedContactType
	}
	return opts, nil
}

func extendSendOptions(opts protocol.SendOptions, c *Chat) (protocol.SendOptions, error) {
	switch c.contact.Type {
	case ContactPublicChat:
		opts.ChatName = c.contact.Name
	case ContactPrivateChat:
		opts.Recipient = c.contact.PublicKey
	default:
		return opts, errUnsupportedContactType
	}
	return opts, nil
}
