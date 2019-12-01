module github.com/status-im/status-console-client

go 1.13

replace github.com/ethereum/go-ethereum v1.9.5 => github.com/status-im/go-ethereum v1.9.5-status.6

replace github.com/Sirupsen/logrus v1.4.2 => github.com/sirupsen/logrus v1.4.2

replace github.com/docker/docker => github.com/docker/engine v1.4.2-0.20190717161051-705d9623b7c1

replace github.com/gomarkdown/markdown => github.com/status-im/markdown v0.0.0-20191113114344-af599402d015

replace github.com/status-im/status-go/eth-node => github.com/status-im/status-go/eth-node v0.36.0

require (
	github.com/Microsoft/go-winio v0.4.12 // indirect
	github.com/allegro/bigcache v1.2.1 // indirect
	github.com/dhui/dktest v0.3.1 // indirect
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/ethereum/go-ethereum v1.9.5
	github.com/fatih/color v1.7.0
	github.com/google/uuid v1.1.1
	github.com/gorilla/mux v1.7.3 // indirect
	github.com/jroimartin/gocui v0.4.0
	github.com/mattn/go-colorable v0.1.2 // indirect
	github.com/nsf/termbox-go v0.0.0-20190624072549-eeb6cd0a1762 // indirect
	github.com/peterbourgon/ff v1.2.0
	github.com/pkg/errors v0.8.1
	github.com/status-im/status-go v0.36.1
	github.com/status-im/status-go/eth-node v0.36.0
	github.com/status-im/status-go/protocol v0.5.3-0.20191205162534-fd49b0140eba
	github.com/stretchr/objx v0.2.0 // indirect
	go.uber.org/zap v1.13.0
	google.golang.org/genproto v0.0.0-20190701230453-710ae3a149df // indirect
	google.golang.org/grpc v1.22.0 // indirect
)
