package datasync

import (
	"fmt"

	"github.com/status-im/status-console-client/protocol/client"
	"github.com/status-im/status-console-client/protocol/v1"
)

var (
	errUnsupportedContactType = fmt.Errorf("unsupported contact type")
)

func createSubscribeOptions(c client.Contact) (opts protocol.SubscribeOptions, err error) {
	switch c.Type {
	case client.ContactPublicRoom:
		opts.ChatName = c.Name
	case client.ContactPublicKey:
		opts.Recipient = c.PublicKey
	default:
		err = errUnsupportedContactType
	}
	return
}

func createSendOptions(c client.Contact) (opts protocol.SendOptions, err error) {
	switch c.Type {
	case client.ContactPublicRoom:
		opts.ChatName = c.Name
	case client.ContactPublicKey:
		opts.Recipient = c.PublicKey
	default:
		err = errUnsupportedContactType
	}
	return
}

func enhanceRequestOptions(c client.Contact, opts *protocol.RequestOptions) error {
	var chatOptions protocol.ChatOptions

	switch c.Type {
	case client.ContactPublicRoom:
		chatOptions.ChatName = c.Name
	case client.ContactPublicKey:
		chatOptions.Recipient = c.PublicKey
	default:
		return errUnsupportedContactType
	}

	opts.Chats = append(opts.Chats, chatOptions)

	return nil
}
