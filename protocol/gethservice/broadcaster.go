package gethservice

import (
	"github.com/ethereum/go-ethereum/event"
	"github.com/status-im/status-console-client/protocol/client"
)

type publisher interface {
	Subscribe(chan client.Event) event.Subscription
}

type broadcaster struct {
	source publisher
	subs   map[client.Chat][]chan interface{}
	cancel chan struct{}
}

func newBroadcaster(source publisher) *broadcaster {
	b := broadcaster{
		source: source,
		subs:   make(map[client.Chat][]chan interface{}),
		cancel: make(chan struct{}),
	}

	go b.start(b.cancel)

	return &b
}

func (b *broadcaster) start(cancel chan struct{}) {
	events := make(chan client.Event)
	sub := b.source.Subscribe(events)
	for {
		select {
		case item := <-events:
			var subs []chan interface{}

			switch v := item.Interface.(type) {
			case client.EventWithChat:
				subs = b.subs[v.GetChat()]
			}

			// TODO: figure out if we need anything here
			// to handle a slower consumer.
			for _, out := range subs {
				out <- item
			}
		case <-sub.Err():
			return
		case <-cancel:
			sub.Unsubscribe()
			return
		}
	}
}

func (b *broadcaster) Subscribe(c client.Chat) <-chan interface{} {
	// TODO: think about whether this can be a buffered channel.
	sub := make(chan interface{})
	b.subs[c] = append(b.subs[c], sub)
	return sub
}

func (b *broadcaster) Unsubscribe(sub <-chan interface{}) {
	// filter without allocation
	for c, subs := range b.subs {
		subs := subs[:0]
		for _, x := range subs {
			if x != sub {
				subs = append(subs, x)
			}
		}
		// garbage collect
		for i := len(subs); i < len(b.subs[c]); i++ {
			b.subs[c][i] = nil
		}
		b.subs[c] = subs
	}
}
