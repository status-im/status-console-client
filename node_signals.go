package main

import (
	"encoding/hex"
	"encoding/json"
	"log"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/signal"
)

func printHandler(event string) {
	log.Printf("received signal: %s", event)
}

type signalEnvelope struct {
	Type  string          `json:"type"`
	Event json.RawMessage `json:"event"`
}

type mailTypeEvent struct {
	RequestID        common.Hash `json:"requestID"`
	Hash             common.Hash `json:"hash"`
	LastEnvelopeHash common.Hash `json:"lastEnvelopeHash"`
}

type mailTypeSignal struct {
	Type           string
	RequestID      string
	LastEnvelopeID []byte
}

type signalForwarder struct {
	sync.Mutex

	in  chan mailTypeSignal
	out map[string]chan<- mailTypeSignal
}

func newSignalForwarder() *signalForwarder {
	return &signalForwarder{
		in:  make(chan mailTypeSignal),
		out: make(map[string]chan<- mailTypeSignal),
	}
}

func (s *signalForwarder) Start() {
	for {
		sig, ok := <-s.in
		if !ok {
			return
		}

		s.Lock()
		out, found := s.out[sig.RequestID]
		s.Unlock()
		if found {
			out <- sig
		}
	}
}

func (s *signalForwarder) cancel(reqID []byte) {
	s.Lock()
	delete(s.out, hex.EncodeToString(reqID))
	s.Unlock()
}

func (s *signalForwarder) Filter(reqID []byte) (<-chan mailTypeSignal, func()) {
	c := make(chan mailTypeSignal)
	s.Lock()
	s.out[hex.EncodeToString(reqID)] = c
	s.Unlock()
	return c, func() { s.cancel(reqID); close(c) }
}

func filterMailTypesHandler(fn func(string), in chan<- mailTypeSignal) func(string) {
	return func(event string) {
		fn(event)

		var envelope signalEnvelope
		if err := json.Unmarshal([]byte(event), &envelope); err != nil {
			log.Printf("faild to unmarshal signal Envelope: %v", err)
		}

		switch envelope.Type {
		case signal.EventMailServerRequestCompleted:
			var event mailTypeEvent
			if err := json.Unmarshal(envelope.Event, &event); err != nil {
				log.Printf("faild to unmarshal signal event: %v", err)
			}
			in <- mailTypeSignal{
				envelope.Type,
				hex.EncodeToString(event.RequestID.Bytes()),
				event.LastEnvelopeHash.Bytes(),
			}
		case signal.EventMailServerRequestExpired:
			var event mailTypeEvent
			if err := json.Unmarshal(envelope.Event, &event); err != nil {
				log.Printf("faild to unmarshal signal event: %v", err)
			}
			in <- mailTypeSignal{
				envelope.Type,
				hex.EncodeToString(event.Hash.Bytes()),
				nil,
			}
		}
	}
}
