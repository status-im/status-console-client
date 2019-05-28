package client

import (
	"context"
	"crypto/ecdsa"
	"log"
	"sync"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
	"github.com/status-im/status-console-client/protocol/v1"
)

func PubkeyToHex(key *ecdsa.PublicKey) string {
	buf := crypto.FromECDSAPub(key)
	return hexutil.Encode(buf)
}

type StreamHandler func(protocol.Message) error

// TODO(dshulyak) find better names for PublicStream and PrivateStream
func NewPublicHandler(contact Contact, db Database) StreamHandler {
	pub := PublicStream{
		contact: contact,
		db:      db,
	}
	return pub.Handle
}

type PublicStream struct {
	contact Contact
	db      Database
}

func (pub PublicStream) Handle(msg protocol.Message) error {
	_, err := pub.db.SaveMessages(pub.contact, []*protocol.Message{&msg})
	if err == ErrMsgAlreadyExist {
		return err
	} else if err != nil {
		return errors.Wrap(err, "can't add message")
	}
	return nil
}

func NewPrivateHandler(db Database) StreamHandler {
	return PrivateStream{
		db: db,
	}.Handle
}

// PrivateStream is a stream with messages from multiple sources.
// In our case every message will have a pubkey (derived from signature) that will be used
// to determine who is the writer
type PrivateStream struct {
	db Database
}

func (priv PrivateStream) Handle(msg protocol.Message) error {
	publicKey := msg.SigPubKey

	if publicKey == nil {
		return errors.New("message should be signed")
	}

	contact := Contact{
		Type:      ContactPublicKey,
		State:     ContactNew,
		Name:      PubkeyToHex(publicKey), // TODO(dshulyak) replace with 3-word funny name
		PublicKey: publicKey,
		Topic:     DefaultPrivateTopic(),
	}

	exists, err := priv.db.PublicContactExist(contact)
	if err != nil {
		return errors.Wrap(err, "error verifying if public contact exist")
	}
	if exists {
		// TODO: replace with db.ContactByPublicKey()
		contacts, err := priv.db.Contacts()
		if err != nil {
			return errors.Wrap(err, "error getting contacts")
		}
		for _, c := range contacts {
			if c.PublicKey == nil {
				continue
			}

			// TODO: extract
			if publicKey.X.Cmp(c.PublicKey.X) == 0 && publicKey.Y.Cmp(c.PublicKey.Y) == 0 {
				contact = c
				break
			}
		}
	} else {
		err := priv.db.SaveContacts([]Contact{contact})
		if err != nil {
			return errors.Wrap(err, "can't add new contact")
		}
	}
	// TODO discard message from blocked contact (State != ContactBlocked)
	_, err = priv.db.SaveMessages(contact, []*protocol.Message{&msg})
	if err == ErrMsgAlreadyExist {
		return err
	} else if err != nil {
		return errors.Wrap(err, "can't add message")
	}
	return nil
}

type AsyncStream interface {
	Start() error
	Stop()
}

type Stream struct {
	options protocol.SubscribeOptions

	proto   protocol.Protocol
	handler StreamHandler

	parent context.Context
	cancel func()
	wg     sync.WaitGroup
}

func NewStream(ctx context.Context, options protocol.SubscribeOptions, proto protocol.Protocol, handler StreamHandler) *Stream {
	return &Stream{
		options: options,
		proto:   proto,
		handler: handler,
		parent:  ctx,
	}
}

func (stream *Stream) Start() error {
	if stream.cancel != nil {
		return errors.New("already started")
	}
	ctx, cancel := context.WithCancel(stream.parent)
	stream.cancel = cancel
	msgs := make(chan *protocol.Message, 100)
	sub, err := stream.proto.Subscribe(ctx, msgs, stream.options)
	if err != nil {
		stream.cancel = nil
		return err
	}
	stream.wg.Add(1)
	go func() {
		for {
			select {
			case msg := <-msgs:
				err = stream.handler(*msg)
				if err == ErrMsgAlreadyExist {
					log.Printf("[DEBUG] message with ID %x already exist\n", msg.ID)
				} else if err != nil {
					log.Printf("[ERROR] failed to save message with ID %x: %v\n", msg.ID, err)
				}
			case <-ctx.Done():
				sub.Unsubscribe()
				stream.wg.Done()
				return
			}
		}
	}()
	return nil
}

func (stream *Stream) Stop() {
	if stream.cancel == nil {
		return
	}
	stream.cancel()
	stream.wg.Wait()
	stream.cancel = nil
}
