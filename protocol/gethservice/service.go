package gethservice

import (
	"crypto/ecdsa"

	"github.com/status-im/status-console-client/protocol/v1"

	"github.com/status-im/status-go/node"

	gethnode "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
)

const (
	// ServiceProtosAPIName is a name of the API namespace
	// with the protocol specific methods.
	ServiceProtosAPIName = "protos"
)

var _ gethnode.Service = (*Service)(nil)

// KeysGetter is an interface that specifies what kind of keys
// should an implementation provide.
type KeysGetter interface {
	PrivateKey() (*ecdsa.PrivateKey, error)
}

// Service is a wrapper around Protocol.
type Service struct {
	node     *node.StatusNode
	keys     KeysGetter
	protocol protocol.Protocol
}

// New creates a new Service.
func New(node *node.StatusNode, keys KeysGetter) *Service {
	return &Service{
		node: node,
		keys: keys,
	}
}

// SetProtocol sets a given Protocol implementation.
func (s *Service) SetProtocol(proto protocol.Protocol) {
	s.protocol = proto
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
			Namespace: ServiceProtosAPIName,
			Version:   "1.0",
			Service:   &PublicAPI{service: s},
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
