// +build geth !nimbus

package gethservice

import (
	"crypto/ecdsa"

	"github.com/status-im/status-go/node"

	gethnode "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"

	status "github.com/status-im/status-protocol-go"
)

const (
	// StatusSecureMessagingProtocolAPIName is a name of the API namespace
	// with the protocol specific methods.
	StatusSecureMessagingProtocolAPIName = "ssm"
)

var _ gethnode.Service = (*Service)(nil)

// KeysGetter is an interface that specifies what kind of keys
// should an implementation provide.
type KeysGetter interface {
	PrivateKey() (*ecdsa.PrivateKey, error)
}

// Service is a wrapper around Protocol.
type Service struct {
	node      *node.StatusNode
	keys      KeysGetter
	messenger *status.Messenger
}

// New creates a new Service.
func New(node *node.StatusNode, keys KeysGetter) *Service {
	return &Service{
		node: node,
		keys: keys,
	}
}

// SetMessenger sets a Messenger.
func (s *Service) SetMessenger(m *status.Messenger) {
	s.messenger = m
}

// gethnode.Service interface implementation

// Protocols list a list of p2p protocols defined by this service.s
func (s *Service) Protocols() []p2p.Protocol {
	return nil
}

// APIs retrieves the list of RPC descriptors the service provides.
func (s *Service) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: StatusSecureMessagingProtocolAPIName,
			Version:   "1.0",
			Service:   NewPublicAPI(s),
			Public:    true,
		},
	}
}

// Start is called after all services have been constructed and the networking
// layer was also initialized to spawn any goroutines required by the service.
func (s *Service) Start(server *p2p.Server) error {
	return nil
}

// Stop terminates all goroutines belonging to the service, blocking until they
// are all terminated.
func (s *Service) Stop() error {
	return nil
}
