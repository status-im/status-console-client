package node

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/les"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/nat"
	"github.com/status-im/status-go/mailserver"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/incentivisation"
	"github.com/status-im/status-go/services/peer"
	"github.com/status-im/status-go/services/personal"
	"github.com/status-im/status-go/services/shhext"
	"github.com/status-im/status-go/services/status"
	"github.com/status-im/status-go/services/wallet"
	"github.com/status-im/status-go/static"
	"github.com/status-im/status-go/timesource"
	whisper "github.com/status-im/whisper/whisperv6"
	"github.com/syndtr/goleveldb/leveldb"
)

// Errors related to node and services creation.
var (
	ErrNodeMakeFailureFormat                      = "error creating p2p node: %s"
	ErrWhisperServiceRegistrationFailure          = errors.New("failed to register the Whisper service")
	ErrLightEthRegistrationFailure                = errors.New("failed to register the LES service")
	ErrLightEthRegistrationFailureUpstreamEnabled = errors.New("failed to register the LES service, upstream is also configured")
	ErrPersonalServiceRegistrationFailure         = errors.New("failed to register the personal api service")
	ErrStatusServiceRegistrationFailure           = errors.New("failed to register the Status service")
	ErrPeerServiceRegistrationFailure             = errors.New("failed to register the Peer service")
	ErrIncentivisationServiceRegistrationFailure  = errors.New("failed to register the Incentivisation service")
	ErrWalletServiceRegistrationFailure           = errors.New("failed to register the Wallet service")
)

// All general log messages in this package should be routed through this logger.
var logger = log.New("package", "status-go/node")

// MakeNode creates a geth node entity
func MakeNode(config *params.NodeConfig, db *leveldb.DB) (*node.Node, error) {
	// If DataDir is empty, it means we want to create an ephemeral node
	// keeping data only in memory.
	if config.DataDir != "" {
		// make sure data directory exists
		if err := os.MkdirAll(filepath.Clean(config.DataDir), os.ModePerm); err != nil {
			return nil, fmt.Errorf("make node: make data directory: %v", err)
		}

		// make sure keys directory exists
		if err := os.MkdirAll(filepath.Clean(config.KeyStoreDir), os.ModePerm); err != nil {
			return nil, fmt.Errorf("make node: make keys directory: %v", err)
		}
	}

	stackConfig, err := newGethNodeConfig(config)
	if err != nil {
		return nil, err
	}

	stack, err := node.New(stackConfig)
	if err != nil {
		return nil, fmt.Errorf(ErrNodeMakeFailureFormat, err.Error())
	}

	// start Ethereum service if we are not expected to use an upstream server
	if !config.UpstreamConfig.Enabled {
		if err := activateLightEthService(stack, config); err != nil {
			return nil, fmt.Errorf("%v: %v", ErrLightEthRegistrationFailure, err)
		}
	} else {
		if config.LightEthConfig.Enabled {
			return nil, fmt.Errorf("%v: %v", ErrLightEthRegistrationFailureUpstreamEnabled, err)
		}

		logger.Info("LES protocol is disabled")

		// `personal_sign` and `personal_ecRecover` methods are important to
		// keep DApps working.
		// Usually, they are provided by an ETH or a LES service, but when using
		// upstream, we don't start any of these, so we need to start our own
		// implementation.
		if err := activatePersonalService(stack, config); err != nil {
			return nil, fmt.Errorf("%v: %v", ErrPersonalServiceRegistrationFailure, err)
		}
	}

	// start Whisper service.
	if err := activateShhService(stack, config, db); err != nil {
		return nil, fmt.Errorf("%v: %v", ErrWhisperServiceRegistrationFailure, err)
	}

	// start incentivisation service
	if err := activateIncentivisationService(stack, config); err != nil {
		return nil, fmt.Errorf("%v: %v", ErrIncentivisationServiceRegistrationFailure, err)
	}

	// start status service.
	if err := activateStatusService(stack, config); err != nil {
		return nil, fmt.Errorf("%v: %v", ErrStatusServiceRegistrationFailure, err)
	}

	// start peer service
	if err := activatePeerService(stack); err != nil {
		return nil, fmt.Errorf("%v: %v", ErrPeerServiceRegistrationFailure, err)
	}

	if err := activateWalletService(stack, config.WalletConfig); err != nil {
		return nil, fmt.Errorf("%v: %v", ErrWalletServiceRegistrationFailure, err)
	}

	return stack, nil
}

// newGethNodeConfig returns default stack configuration for mobile client node
func newGethNodeConfig(config *params.NodeConfig) (*node.Config, error) {
	nc := &node.Config{
		DataDir:           config.DataDir,
		KeyStoreDir:       config.KeyStoreDir,
		UseLightweightKDF: true,
		NoUSB:             true,
		Name:              config.Name,
		Version:           config.Version,
		P2P: p2p.Config{
			NoDiscovery:     true, // we always use only v5 server
			ListenAddr:      config.ListenAddr,
			NAT:             nat.Any(),
			MaxPeers:        config.MaxPeers,
			MaxPendingPeers: config.MaxPendingPeers,
		},
		HTTPModules: config.FormatAPIModules(),
	}

	if config.IPCEnabled {
		// use well-known defaults
		if config.IPCFile == "" {
			config.IPCFile = "geth.ipc"
		}

		nc.IPCPath = config.IPCFile
	}

	if config.HTTPEnabled {
		nc.HTTPHost = config.HTTPHost
		nc.HTTPPort = config.HTTPPort
		nc.HTTPVirtualHosts = config.HTTPVirtualHosts
		nc.HTTPCors = config.HTTPCors
	}

	if config.ClusterConfig.Enabled {
		nc.P2P.BootstrapNodesV5 = parseNodesV5(config.ClusterConfig.BootNodes)
		nc.P2P.StaticNodes = parseNodes(config.ClusterConfig.StaticNodes)
	}

	if config.NodeKey != "" {
		sk, err := crypto.HexToECDSA(config.NodeKey)
		if err != nil {
			return nil, err
		}
		// override node's private key
		nc.P2P.PrivateKey = sk
	}

	return nc, nil
}

// calculateGenesis retrieves genesis value for given network
func calculateGenesis(networkID uint64) (*core.Genesis, error) {
	var genesis *core.Genesis

	switch networkID {
	case params.MainNetworkID:
		genesis = core.DefaultGenesisBlock()
	case params.RopstenNetworkID:
		genesis = core.DefaultTestnetGenesisBlock()
	case params.RinkebyNetworkID:
		genesis = core.DefaultRinkebyGenesisBlock()
	case params.GoerliNetworkID:
		genesis = core.DefaultGoerliGenesisBlock()
	case params.StatusChainNetworkID:
		var err error
		if genesis, err = defaultStatusChainGenesisBlock(); err != nil {
			return nil, err
		}
	default:
		return nil, nil
	}

	return genesis, nil
}

// defaultStatusChainGenesisBlock returns the StatusChain network genesis block.
func defaultStatusChainGenesisBlock() (*core.Genesis, error) {
	genesisJSON, err := static.ConfigStatusChainGenesisJsonBytes()
	if err != nil {
		return nil, fmt.Errorf("status-chain-genesis.json could not be loaded: %s", err)
	}

	var genesis *core.Genesis
	err = json.Unmarshal(genesisJSON, &genesis)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal status-chain-genesis.json: %s", err)
	}
	return genesis, nil
}

// activateLightEthService configures and registers the eth.Ethereum service with a given node.
func activateLightEthService(stack *node.Node, config *params.NodeConfig) error {
	if !config.LightEthConfig.Enabled {
		logger.Info("LES protocol is disabled")
		return nil
	}

	genesis, err := calculateGenesis(config.NetworkID)
	if err != nil {
		return err
	}

	ethConf := eth.DefaultConfig
	ethConf.Genesis = genesis
	ethConf.SyncMode = downloader.LightSync
	ethConf.NetworkId = config.NetworkID
	ethConf.DatabaseCache = config.LightEthConfig.DatabaseCache
	ethConf.ULC = &eth.ULCConfig{
		TrustedServers:     config.LightEthConfig.TrustedNodes,
		MinTrustedFraction: config.LightEthConfig.MinTrustedFraction,
	}
	return stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		return les.New(ctx, &ethConf)
	})
}

func activatePersonalService(stack *node.Node, config *params.NodeConfig) error {
	return stack.Register(func(*node.ServiceContext) (node.Service, error) {
		svc := personal.New(stack.AccountManager())
		return svc, nil
	})
}

func activateStatusService(stack *node.Node, config *params.NodeConfig) error {
	if !config.EnableStatusService {
		logger.Info("Status service api is disabled")
		return nil
	}

	return stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		var whisper *whisper.Whisper
		if err := ctx.Service(&whisper); err != nil {
			return nil, err
		}
		svc := status.New(whisper)
		return svc, nil
	})
}

func activatePeerService(stack *node.Node) error {
	return stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		svc := peer.New()
		return svc, nil
	})
}

func activateWalletService(stack *node.Node, config params.WalletConfig) error {
	if !config.Enabled {
		logger.Info("service.Wallet is disabled")
		return nil
	}
	return stack.Register(func(*node.ServiceContext) (node.Service, error) {
		return wallet.NewService(), nil
	})
}

func registerMailServer(whisperService *whisper.Whisper, config *params.WhisperConfig) (err error) {
	var mailServer mailserver.WMailServer
	whisperService.RegisterServer(&mailServer)

	return mailServer.Init(whisperService, config)
}

// activateShhService configures Whisper and adds it to the given node.
func activateShhService(stack *node.Node, config *params.NodeConfig, db *leveldb.DB) (err error) {
	if !config.WhisperConfig.Enabled {
		logger.Info("SHH protocol is disabled")
		return nil
	}
	if config.WhisperConfig.EnableNTPSync {
		if err = stack.Register(func(*node.ServiceContext) (node.Service, error) {
			return timesource.Default(), nil
		}); err != nil {
			return
		}
	}

	err = stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		whisperServiceConfig := &whisper.Config{
			MaxMessageSize:     whisper.DefaultMaxMessageSize,
			MinimumAcceptedPOW: params.WhisperMinimumPoW,
		}

		if config.WhisperConfig.MaxMessageSize > 0 {
			whisperServiceConfig.MaxMessageSize = config.WhisperConfig.MaxMessageSize
		}
		if config.WhisperConfig.MinimumPoW > 0 {
			whisperServiceConfig.MinimumAcceptedPOW = config.WhisperConfig.MinimumPoW
		}

		whisperService := whisper.New(whisperServiceConfig)

		if config.WhisperConfig.EnableNTPSync {
			timesource, err := whisperTimeSource(ctx)
			if err != nil {
				return nil, err
			}
			whisperService.SetTimeSource(timesource)
		}

		// enable mail service
		if config.WhisperConfig.EnableMailServer {
			if err := registerMailServer(whisperService, &config.WhisperConfig); err != nil {
				return nil, fmt.Errorf("failed to register MailServer: %v", err)
			}
		}

		if config.WhisperConfig.LightClient {
			emptyBloomFilter := make([]byte, 64)
			if err := whisperService.SetBloomFilter(emptyBloomFilter); err != nil {
				return nil, err
			}
		}

		return whisperService, nil
	})
	if err != nil {
		return
	}

	// TODO(dshulyak) add a config option to enable it by default, but disable if app is started from statusd
	return stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		var whisper *whisper.Whisper
		if err := ctx.Service(&whisper); err != nil {
			return nil, err
		}
		return shhext.New(whisper, shhext.EnvelopeSignalHandler{}, db, config.ShhextConfig), nil
	})
}

// activateIncentivisationService configures Whisper and adds it to the given node.
func activateIncentivisationService(stack *node.Node, config *params.NodeConfig) (err error) {
	if !config.WhisperConfig.Enabled {
		logger.Info("SHH protocol is disabled")
		return nil
	}

	if !config.IncentivisationConfig.Enabled {
		logger.Info("Incentivisation is disabled")
		return nil
	}

	logger.Info("activating incentivisation")
	// TODO(dshulyak) add a config option to enable it by default, but disable if app is started from statusd
	return stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		var w *whisper.Whisper
		if err := ctx.Service(&w); err != nil {
			return nil, err
		}
		incentivisationConfig := &incentivisation.ServiceConfig{
			ContractAddress: config.IncentivisationConfig.ContractAddress,
			RPCEndpoint:     config.IncentivisationConfig.RPCEndpoint,
			IP:              config.IncentivisationConfig.IP,
			Port:            config.IncentivisationConfig.Port,
		}
		privateKey, err := crypto.HexToECDSA(config.NodeKey)
		if err != nil {
			return nil, err
		}
		client, err := ethclient.DialContext(context.TODO(), incentivisationConfig.RPCEndpoint)
		if err != nil {
			return nil, err
		}

		contract, err := incentivisation.NewContract(gethcommon.HexToAddress(incentivisationConfig.ContractAddress), client, client)
		if err != nil {
			return nil, err
		}

		return incentivisation.New(privateKey, whisper.NewPublicWhisperAPI(w), incentivisationConfig, contract), nil
	})
}

// parseNodes creates list of enode.Node out of enode strings.
func parseNodes(enodes []string) []*enode.Node {
	var nodes []*enode.Node
	for _, item := range enodes {
		parsedPeer, err := enode.ParseV4(item)
		if err == nil {
			nodes = append(nodes, parsedPeer)
		} else {
			logger.Error("Failed to parse enode", "enode", item, "err", err)
		}

	}
	return nodes
}

// parseNodesV5 creates list of discv5.Node out of enode strings.
func parseNodesV5(enodes []string) []*discv5.Node {
	var nodes []*discv5.Node
	for _, enode := range enodes {
		parsedPeer, err := discv5.ParseNode(enode)

		if err == nil {
			nodes = append(nodes, parsedPeer)
		} else {
			logger.Error("Failed to parse enode", "enode", enode, "err", err)
		}
	}
	return nodes
}

func parseNodesToNodeID(enodes []string) []enode.ID {
	nodeIDs := make([]enode.ID, 0, len(enodes))
	for _, node := range parseNodes(enodes) {
		nodeIDs = append(nodeIDs, node.ID())
	}
	return nodeIDs
}

// whisperTimeSource get timeSource to be used by whisper
func whisperTimeSource(ctx *node.ServiceContext) (func() time.Time, error) {
	var timeSource *timesource.NTPTimeSource
	if err := ctx.Service(&timeSource); err != nil {
		return nil, err
	}
	return timeSource.Now, nil
}
