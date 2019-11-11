package main

import (
	"encoding/hex"
	"encoding/json"
	"log"
	"sync"

	"github.com/status-im/status-go/signal"
)

type signalEnvelope struct {
	Type  string          `json:"type"`
	Event json.RawMessage `json:"event"`
}

type mailTypeEvent struct {
	RequestID        []byte `json:"requestID"`
	Hash             []byte `json:"hash"`
	LastEnvelopeHash []byte `json:"lastEnvelopeHash"`
	Cursor           string `json:"cursor"`
	Error            string `json:"errorMessage"`
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

func filterMailTypesHandler(in chan<- mailTypeSignal) func(string) {
	return func(event string) {
		var envelope signalEnvelope
		if err := json.Unmarshal([]byte(event), &envelope); err != nil {
			log.Printf("faild to unmarshal signal Envelope: %v", err)
			return
		}

		log.Printf("recieved signal (%s) with event: %s", envelope.Type, envelope.Event)

		switch envelope.Type {
		case signal.EventMailServerRequestCompleted:
			var event mailTypeEvent
			if err := json.Unmarshal(envelope.Event, &event); err != nil {
				log.Printf("faild to unmarshal signal event: %v", err)
			}
			in <- mailTypeSignal{
				envelope.Type,
				hex.EncodeToString(event.RequestID),
				event.LastEnvelopeHash,
			}
		case signal.EventMailServerRequestExpired:
			var event mailTypeEvent
			if err := json.Unmarshal(envelope.Event, &event); err != nil {
				log.Printf("faild to unmarshal signal event: %v", err)
			}
			in <- mailTypeSignal{
				envelope.Type,
				hex.EncodeToString(event.Hash),
				nil,
			}
		}
	}
}
