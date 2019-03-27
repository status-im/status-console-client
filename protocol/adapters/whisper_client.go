package adapters

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/status-im/status-console-client/protocol/v1"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/whisper/shhclient"

	whisper "github.com/status-im/whisper/whisperv6"
)

// WhisperClientAdapter is an adapter for Whisper client
// which implements Chat interface. It requires an RPC client
// which can use various transports like HTTP, IPC or in-proc.
type WhisperClientAdapter struct {
	rpcClient               *rpc.Client
	shhClient               *shhclient.Client
	mailServerEnodes        []string
	selectedMailServerEnode string

	mu              sync.RWMutex
	passSymKeyCache map[string]string
}

// WhisperClientAdapter must implement Chat interface.
var _ protocol.Chat = (*WhisperClientAdapter)(nil)

// NewWhisperClientAdapter returns a new WhisperClientAdapter.
func NewWhisperClientAdapter(c *rpc.Client, mailServers []string) *WhisperClientAdapter {
	return &WhisperClientAdapter{
		rpcClient:        c,
		shhClient:        shhclient.NewClient(c),
		mailServerEnodes: mailServers,
		passSymKeyCache:  make(map[string]string),
	}
}

// Subscribe subscribes to a public channel.
// in channel is used to receive messages.
func (a *WhisperClientAdapter) Subscribe(
	ctx context.Context,
	in chan<- *protocol.Message,
	options protocol.SubscribeOptions,
) (*protocol.Subscription, error) {
	if err := options.Validate(); err != nil {
		return nil, err
	}

	criteria := whisper.Criteria{
		MinPow:   0,    // TODO: set it to proper value
		AllowP2P: true, // messages from mail server are direct p2p messages
	}

	if options.Identity != nil {
		identityID, err := a.shhClient.AddPrivateKey(ctx, crypto.FromECDSA(options.Identity))
		if err != nil {
			return nil, err
		}
		criteria.PrivateKeyID = identityID

		topic, err := PrivateChatTopic()
		if err != nil {
			return nil, err
		}
		criteria.Topics = append(criteria.Topics, topic)
	}

	if options.ChatName != "" {
		symKeyID, err := a.getOrAddSymKey(ctx, options.ChatName)
		if err != nil {
			return nil, err
		}
		criteria.SymKeyID = symKeyID

		topic, err := PublicChatTopic(options.ChatName)
		if err != nil {
			return nil, err
		}
		criteria.Topics = append(criteria.Topics, topic)
	}

	return a.subscribeMessages(ctx, criteria, in)
}

func (a *WhisperClientAdapter) subscribeMessages(
	ctx context.Context,
	crit whisper.Criteria,
	in chan<- *protocol.Message,
) (*protocol.Subscription, error) {
	messages := make(chan *whisper.Message)
	shhSub, err := a.shhClient.SubscribeMessages(ctx, crit, messages)
	if err != nil {
		return nil, err
	}

	sub := protocol.NewSubscription()

	go func() {
		defer shhSub.Unsubscribe()

		for {
			select {
			case raw := <-messages:
				m, err := protocol.DecodeMessage(raw.Payload)
				if err != nil {
					log.Printf("failed to decode message: %v", err)
					break
				}

				sigPubKey, err := crypto.UnmarshalPubkey(raw.Sig)
				if err != nil {
					log.Printf("failed to get a signature: %v", err)
					break
				}

				in <- &protocol.Message{
					Decoded:   m,
					SigPubKey: sigPubKey,
				}
			case err := <-shhSub.Err():
				sub.Cancel(err)
				return
			case <-sub.Done():
				return
			}
		}
	}()

	return sub, nil
}

// Send sends a new message to a public chat.
// Identity is required to sign a message as only signed messages
// are accepted and displayed.
func (a *WhisperClientAdapter) Send(
	ctx context.Context,
	data []byte,
	options protocol.SendOptions,
) ([]byte, error) {
	if err := options.Validate(); err != nil {
		return nil, err
	}

	identityID, err := a.shhClient.AddPrivateKey(ctx, crypto.FromECDSA(options.Identity))
	if err != nil {
		return nil, err
	}

	message, err := a.createNewMessage(data, identityID, options)
	if err != nil {
		return nil, err
	}

	hash, err := a.shhClient.Post(ctx, message)
	if err != nil {
		return nil, err
	}
	return hex.DecodeString(hash)
}

func (a *WhisperClientAdapter) createNewMessage(data []byte, sigKey string, options protocol.SendOptions) (whisper.NewMessage, error) {
	message := createWhisperNewMessage(data, sigKey)

	if options.Recipient != nil {
		message.PublicKey = crypto.FromECDSAPub(options.Recipient)

		topic, err := PrivateChatTopic()
		if err != nil {
			return message, err
		}
		message.Topic = topic
	}

	if options.ChatName != "" {
		ctx := context.Background()
		symKeyID, err := a.getOrAddSymKey(ctx, options.ChatName)
		if err != nil {
			return message, err
		}
		message.SymKeyID = symKeyID

		topic, err := PublicChatTopic(options.ChatName)
		if err != nil {
			return message, err
		}
		message.Topic = topic
	}

	return message, nil
}

// Request sends a request to MailServer for historic messages.
func (a *WhisperClientAdapter) Request(ctx context.Context, params protocol.RequestOptions) error {
	enode, err := a.selectAndAddMailServer(ctx)
	if err != nil {
		return err
	}
	return a.requestMessages(ctx, enode, params)
}

func (a *WhisperClientAdapter) selectAndAddMailServer(ctx context.Context) (string, error) {
	if a.selectedMailServerEnode != "" {
		return a.selectedMailServerEnode, nil
	}

	enode := randomItem(a.mailServerEnodes)

	if err := a.rpcClient.CallContext(ctx, nil, "admin_addPeer", enode); err != nil {
		return "", err
	}

	// Adding peer is asynchronous operation so we need to retry a few times.
	retries := 0
	for {
		err := a.shhClient.MarkTrustedPeer(ctx, enode)
		if ctx.Err() == context.Canceled {
			log.Printf("requesting public messages canceled")
			return "", err
		}
		if err == nil {
			break
		}
		if retries < 3 {
			retries++
			<-time.After(time.Second)
		} else {
			return "", fmt.Errorf("failed to mark peer as trusted: %v", err)
		}
	}

	a.selectedMailServerEnode = enode

	return enode, nil
}

func (a *WhisperClientAdapter) requestMessages(ctx context.Context, enode string, params protocol.RequestOptions) error {
	log.Printf("requesting messages from node %s", enode)

	arg, err := a.createMessagesRequest(enode, params)
	if err != nil {
		return err
	}

	return a.rpcClient.CallContext(ctx, nil, "shhext_requestMessages", arg)
}

func (a *WhisperClientAdapter) createMessagesRequest(
	enode string,
	params protocol.RequestOptions,
) (req shhextRequestMessagesParam, err error) {
	mailSymKeyID, err := a.getOrAddSymKey(context.Background(), MailServerPassword)
	if err != nil {
		return req, err
	}

	req = shhextRequestMessagesParam{
		MailServerPeer: enode,
		From:           params.From,
		To:             params.To,
		Limit:          params.Limit,
		SymKeyID:       mailSymKeyID,
	}

	if params.Recipient != nil {
		topic, err := PrivateChatTopic()
		if err != nil {
			return req, err
		}
		req.Topics = append(req.Topics, topic)
	}

	if params.ChatName != "" {
		topic, err := PublicChatTopic(params.ChatName)
		if err != nil {
			return req, err
		}
		req.Topics = append(req.Topics, topic)
	}

	return
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
