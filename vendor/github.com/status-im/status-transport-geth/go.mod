module github.com/status-im/status-transport-geth

go 1.13

replace github.com/ethereum/go-ethereum v1.9.5 => github.com/status-im/go-ethereum v1.9.5-status.4

require (
	github.com/elastic/gosigar v0.10.5 // indirect
	github.com/ethereum/go-ethereum v1.9.5
	github.com/status-im/status-protocol-go v0.2.3-0.20191009075803-96398fc3d4b6
	github.com/status-im/whisper v1.5.1
)
