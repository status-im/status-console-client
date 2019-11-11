module github.com/status-im/status-console-client

go 1.13

replace github.com/ethereum/go-ethereum v1.9.5 => github.com/status-im/go-ethereum v1.9.5-status.5

replace github.com/NaySoftware/go-fcm => github.com/status-im/go-fcm v1.0.0-status

replace github.com/Sirupsen/logrus => github.com/sirupsen/logrus v1.4.2

replace github.com/docker/docker => github.com/docker/engine v1.4.2-0.20190717161051-705d9623b7c1

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
	github.com/status-im/status-go v0.34.0-beta.5
	github.com/status-im/status-nim v0.0.0-20190724023117-a5693e6e4820
	github.com/status-im/status-protocol-go v0.5.0
	github.com/stretchr/objx v0.2.0 // indirect
	go.uber.org/zap v1.10.0
	google.golang.org/genproto v0.0.0-20190701230453-710ae3a149df // indirect
	google.golang.org/grpc v1.22.0 // indirect
)
