package gethservice

import (
	"github.com/status-im/status-console-client/protocol/client"
)

type broadcaster struct {
	source <-chan interface{}
	subs   map[client.Contact][]chan interface{}
	cancel chan struct{}
}

func newBroadcaster(source <-chan interface{}) *broadcaster {
	b := broadcaster{
		source: source,
		subs:   make(map[client.Contact][]chan interface{}),
		cancel: make(chan struct{}),
	}

	go b.start(b.cancel)

	return &b
}

func (b *broadcaster) start(cancel chan struct{}) {
	for {
		select {
		case item := <-b.source:
			var subs []chan interface{}

			switch v := item.(type) {
			case client.EventWithContact:
				subs = b.subs[v.GetContact()]
			}

			// TODO: figure out if we need anything here
			// to handle a slower consumer.
			for _, out := range subs {
				out <- item
			}
		case <-cancel:
			return
		}
	}
}

func (b *broadcaster) Subscribe(c client.Contact) <-chan interface{} {
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
