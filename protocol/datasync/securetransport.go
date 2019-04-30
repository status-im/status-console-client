package datasync

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"sync"

	"github.com/pkg/errors"
	"github.com/status-im/status-console-client/protocol/client"
	"github.com/status-im/status-console-client/protocol/v1"
)

type SecureTransport struct {
	sync.RWMutex
	proto protocol.Protocol

	// Identity and Contact between the conversation happens.
	identity *ecdsa.PrivateKey
	contact  client.Contact

	db *client.Database

	requester requester

	cancel chan struct{} // can be closed by any goroutine and closes all others

	ownMessages chan *protocol.Message // my private messages channel
}

func (st *SecureTransport) Send(data []byte) error {
	// If cancel is closed then it will return an error.
	// Otherwise, the execution will continue.
	// This is needed to prevent sending messages
	// if the chat is already left/canceled
	// as a it can't be guaranteed that processing
	// loop goroutines are still running.
	select {
	case _, ok := <-st.cancel:
		if !ok {
			return errors.New("chat is already left")
		}
	default:
	}

	var message protocol.Message

	// @todo do we need thsi?
	switch st.contact.Type {
	case client.ContactPublicRoom:
		message = protocol.CreatePublicTextMessage(data, st.lastClock, st.contact.Name)
	case client.ContactPublicKey:
		message = protocol.CreatePrivateTextMessage(data, st.lastClock, st.contact.Name)
	default:
		return fmt.Errorf("failed to send message: unsupported contact type")
	}

	encodedMessage, err := protocol.EncodeMessage(message)
	if err != nil {
		return errors.Wrap(err, "failed to encode message")
	}

	opts, err := createSendOptions(st.contact)
	if err != nil {
		return errors.Wrap(err, "failed to prepare send options")
	}

	hash, err := st.proto.Send(context.Background(), encodedMessage, opts)

	// Own messages need to be pushed manually to the pipeline.
	if st.contact.Type == client.ContactPublicKey {
		log.Printf("[Chat::Send] sent a private message")

		// TODO: this should be created by st.proto
		message.SigPubKey = &st.identity.PublicKey
		message.ID = hash
		st.ownMessages <- &message
	}

	return err
}
