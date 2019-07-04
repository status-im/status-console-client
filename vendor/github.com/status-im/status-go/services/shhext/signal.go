package shhext

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/signal"
)

// EnvelopeSignalHandler sends signals when envelope is sent or expired.
type EnvelopeSignalHandler struct{}

// EnvelopeSent triggered when envelope delivered atleast to 1 peer.
func (h EnvelopeSignalHandler) EnvelopeSent(hash common.Hash) {
	signal.SendEnvelopeSent(hash)
}

// EnvelopeExpired triggered when envelope is expired but wasn't delivered to any peer.
func (h EnvelopeSignalHandler) EnvelopeExpired(hash common.Hash, err error) {
	signal.SendEnvelopeExpired(hash, err)
}

// MailServerRequestCompleted triggered when the mailserver sends a message to notify that the request has been completed
func (h EnvelopeSignalHandler) MailServerRequestCompleted(requestID common.Hash, lastEnvelopeHash common.Hash, cursor []byte, err error) {
	signal.SendMailServerRequestCompleted(requestID, lastEnvelopeHash, cursor, err)
}

// MailServerRequestExpired triggered when the mailserver request expires
func (h EnvelopeSignalHandler) MailServerRequestExpired(hash common.Hash) {
	signal.SendMailServerRequestExpired(hash)
}

// PublisherSignalHandler sends signals on protocol events
type PublisherSignalHandler struct{}

func (h PublisherSignalHandler) DecryptMessageFailed(pubKey string) {
	signal.SendDecryptMessageFailed(pubKey)
}

func (h PublisherSignalHandler) BundleAdded(identity string, installationID string) {
	signal.SendBundleAdded(identity, installationID)
}

func (h PublisherSignalHandler) WhisperFilterAdded(filters []*signal.Filter) {
	signal.SendWhisperFilterAdded(filters)
}
