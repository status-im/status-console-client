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
	case ContactPublicChat:
		opts.ChatName = c.Name
	case ContactPrivateChat:
		opts.Recipient = c.PublicKey
	default:
		err = errUnsupportedContactType
	}
	return
}

func createRequestOptions(c Contact) (opts protocol.RequestOptions, err error) {
	switch c.Type {
	case ContactPublicChat:
		opts.ChatName = c.Name
	case ContactPrivateChat:
		opts.Recipient = c.PublicKey
	default:
		err = errUnsupportedContactType
	}
	return
}

func createSendOptions(c Contact) (opts protocol.SendOptions, err error) {
	switch c.Type {
	case ContactPublicChat:
		opts.ChatName = c.Name
	case ContactPrivateChat:
		opts.Recipient = c.PublicKey
	default:
		err = errUnsupportedContactType
	}
	return
}
