package main

import (
	"fmt"
	"os"

	"github.com/status-im/status-go/params"
)

func withListenAddr(listenAddr string) params.Option {
	return func(c *params.NodeConfig) error {
		c.ListenAddr = listenAddr
		return nil
	}
}

func generateStatusNodeConfig(dataDir, fleet, listenAddr string, configFile string) (*params.NodeConfig, error) {
	if err := os.MkdirAll(dataDir, os.ModeDir|0755); err != nil {
		return nil, fmt.Errorf("failed to create a data dir: %v", err)
	}

	var configFiles []string
	if configFile != "" {
		configFiles = append(configFiles, configFile)
	}

	config, err := params.NewNodeConfigWithDefaultsAndFiles(
		dataDir,
		params.MainNetworkID,
		[]params.Option{
			params.WithFleet(fleet),
			withListenAddr(listenAddr),
		},
		configFiles,
	)
	if err != nil {
		return nil, err
	}

	config.IPCEnabled = true

	return config, nil
}
