package protocol

import (
	"context"
	"crypto/ecdsa"
	"log"
	"sync"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/services/shhext"
	"github.com/status-im/whisper/shhclient"
	whisper "github.com/status-im/whisper/whisperv6"
)

// WhisperClientAdapter is an adapter for Whisper client
// which implements Chat interface. It requires an RPC client
// which can use various transports like HTTP, IPC or in-proc.
type WhisperClientAdapter struct {
	rpcClient *rpc.Client
	shhClient *shhclient.Client

	mu              sync.RWMutex
	passSymKeyCache map[string]string
}

// WhisperClientAdapter must implement Chat interface.
var _ Chat = (*WhisperClientAdapter)(nil)

// NewWhisperClientAdapter returns a new WhisperClientAdapter.
func NewWhisperClientAdapter(c *rpc.Client) *WhisperClientAdapter {
	return &WhisperClientAdapter{
		rpcClient:       c,
		shhClient:       shhclient.NewClient(c),
		passSymKeyCache: make(map[string]string),
	}
}

// SubscribePublicChat subscribes to a public channel.
// in channel is used to receive messages.
// errCh is used to forward any errors that may occur
// during the subscription.
func (a *WhisperClientAdapter) SubscribePublicChat(ctx context.Context, name string, in chan<- *ReceivedMessage) (*Subscription, error) {
	symKeyID, err := a.getOrAddSymKey(ctx, name)
	if err != nil {
		return nil, err
	}

	topic, err := PublicChatTopic(name)
	if err != nil {
		return nil, err
	}

	messages := make(chan *whisper.Message)
	criteria := whisper.Criteria{
		SymKeyID: symKeyID,
		MinPow:   0, // TODO: set it to proper value
		Topics:   []whisper.TopicType{topic},
		AllowP2P: true, // messages from mail server are direct p2p messages
	}
	shhSub, err := a.shhClient.SubscribeMessages(ctx, criteria, messages)
	if err != nil {
		return nil, err
	}

	sub := NewSubscription()

	go func() {
		defer shhSub.Unsubscribe()

		for {
			select {
			case raw := <-messages:
				m, err := DecodeMessage(raw.Payload)
				if err != nil {
					log.Printf("failed to decode message: %v", err)
					break
				}

				sigPubKey, err := crypto.UnmarshalPubkey(raw.Sig)
				if err != nil {
					log.Printf("failed to get a signature: %v", err)
					break
				}

				in <- &ReceivedMessage{
					Decoded:   m,
					SigPubKey: sigPubKey,
				}
			case err := <-shhSub.Err():
				sub.cancel(err)
				return
			case <-sub.Done():
				return
			}
		}
	}()

	return sub, nil
}

// SendPublicMessage sends a new message to a public chat.
// Identity is required to sign a message as only signed messages
// are accepted and displayed.
func (a *WhisperClientAdapter) SendPublicMessage(ctx context.Context, name string, data []byte, identity *ecdsa.PrivateKey) (string, error) {
	identityID, err := a.shhClient.AddPrivateKey(ctx, crypto.FromECDSA(identity))
	if err != nil {
		return "", err
	}

	symKeyID, err := a.getOrAddSymKey(ctx, name)
	if err != nil {
		return "", err
	}

	topic, err := PublicChatTopic(name)
	if err != nil {
		return "", err
	}

	return a.shhClient.Post(ctx, whisper.NewMessage{
		SymKeyID:  symKeyID,
		TTL:       60,
		Topic:     topic,
		Payload:   data,
		PowTarget: 2.0,
		PowTime:   5,
		Sig:       identityID,
	})
}

// RequestMessagesParams is a list of params sent while requesting historic messages.
type RequestMessagesParams struct {
	MailServerEnode string
	Limit           int
	From            int64
	To              int64
}

// RequestPublicMessages sends a request to MailServer for historic messages.
func (a *WhisperClientAdapter) RequestPublicMessages(ctx context.Context, name string, params RequestMessagesParams) error {
	if err := a.rpcClient.CallContext(ctx, nil, "admin_addPeer"); err != nil {
		return err
	}

	// TODO: check if a peer was added using admin_peers

	if err := a.shhClient.MarkTrustedPeer(ctx, params.MailServerEnode); err != nil {
		return err
	}

	mailServerSymKeyID, err := a.getOrAddSymKey(ctx, MailServerPassword)
	if err != nil {
		return err
	}

	topic, err := PublicChatTopic(name)
	if err != nil {
		return err
	}

	req := shhext.MessagesRequest{
		MailServerPeer: params.MailServerEnode,
		SymKeyID:       mailServerSymKeyID,
		From:           uint32(params.From),  // TODO: change to int in status-go
		To:             uint32(params.To),    // TODO: change to int in status-go
		Limit:          uint32(params.Limit), // TODO: change to int in status-go
		Topics:         []whisper.TopicType{topic},
	}

	return a.rpcClient.CallContext(ctx, nil, "shhext_requestMessages", req)
}

func (a *WhisperClientAdapter) getOrAddSymKey(ctx context.Context, pass string) (string, error) {
	a.mu.RLock()
	symKeyID, ok := a.passSymKeyCache[pass]
	a.mu.RUnlock()

	if ok {
		return symKeyID, nil
	}

	symKeyID, err := a.shhClient.GenerateSymmetricKeyFromPassword(ctx, pass)
	if err != nil {
		return "", err
	}

	a.mu.Lock()
	a.passSymKeyCache[pass] = symKeyID
	a.mu.Unlock()

	return symKeyID, nil
}
