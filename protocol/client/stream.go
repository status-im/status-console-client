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

func NewPrivateHandler(contacts []Contact, db Database) StreamHandler {
	keyed := map[string]Contact{}
	for i := range contacts {
		keyed[PubkeyToHex(contacts[i].PublicKey)] = contacts[i]
	}
	return PrivateStream{
		contacts: keyed,
		db:       db,
	}.Handle
}

// PrivateStream is a stream with messages from multiple sources.
// In our case every message will have a pubkey (derived from signature) that will be used
// to determine who is the writer
type PrivateStream struct {
	contacts map[string]Contact // key is a hex from public key
	db       Database
}

func (priv PrivateStream) Handle(msg protocol.Message) error {
	if msg.SigPubKey == nil {
		return errors.New("message should be signed")
	}
	keyhex := PubkeyToHex(msg.SigPubKey)
	// FIXME(dshulyak) Check if contact exist in database
	// preferably don't marshal key as a blob
	contact := Contact{
		Type:      ContactPublicKey,
		State:     ContactNew,
		Name:      keyhex, // TODO(dshulyak) replace with 3-word funny name
		PublicKey: msg.SigPubKey,
	}
	exist, err := priv.db.PublicContactExist(contact)
	if err != nil {
		return errors.Wrap(err, "error verifying if public contact exist")
	}
	if !exist {
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
	// TODO replace contact with a topic. content from single stream can be delivered to multiple contacts.
	contact Contact

	proto   protocol.Protocol
	handler StreamHandler

	parent context.Context
	cancel func()
	wg     sync.WaitGroup
}

func NewStream(ctx context.Context, contact Contact, proto protocol.Protocol, handler StreamHandler) *Stream {
	return &Stream{
		contact: contact,
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
	opts, err := createSubscribeOptions(stream.contact)
	if err != nil {
		return errors.Wrap(err, "failed to create subscribe options")
	}
	sub, err := stream.proto.Subscribe(ctx, msgs, opts)
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
