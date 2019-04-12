package main

import (
	"fmt"
	stdlog "log"
	"os"

	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/params"
)

func init() {
	if err := logutils.OverrideRootLog(true, "DEBUG", logutils.FileOptions{}, false); err != nil {
		stdlog.Fatalf("failed to override root log: %v\n", err)
	}
}

func generateStatusNodeConfig(dataDir, fleet, configFile string) (*params.NodeConfig, error) {
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
		[]params.Option{params.WithFleet(fleet)},
		configFiles,
	)
	if err != nil {
		return nil, err
	}

	config.IPCEnabled = true

	return config, nil
}
