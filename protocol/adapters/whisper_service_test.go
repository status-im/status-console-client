package adapters

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/status-im/status-console-client/protocol/v1"
	whisper "github.com/status-im/whisper/whisperv6"

	"github.com/stretchr/testify/suite"
)

func TestWhisperServiceAdapterTestSuite(t *testing.T) {
	suite.Run(t, new(WhisperServiceAdapterTestSuite))
}

type WhisperServiceAdapterTestSuite struct {
	suite.Suite

	ws *WhisperServiceAdapter
}

func (s *WhisperServiceAdapterTestSuite) SetupTest() {
	identity, err := crypto.GenerateKey()
	s.Require().NoError(err)

	shhConfig := whisper.DefaultConfig
	shhConfig.MinimumAcceptedPOW = 0
	shh := whisper.New(&shhConfig)

	s.ws = NewWhisperServiceAdapter(nil, shh, identity)
}

func (s *WhisperServiceAdapterTestSuite) TestSendDirectMessage() {
	recipient, err := crypto.GenerateKey()
	s.Require().NoError(err)

	// It will succeed because the message is not immediately pushed through the wire
	// but instead put into a batch which will be sent later.
	_, err = s.ws.Send(context.Background(), []byte("abc"), protocol.SendOptions{
		ChatOptions: protocol.ChatOptions{
			ChatName:  "test-name",
			Recipient: &recipient.PublicKey,
		},
	})
	s.Require().NoError(err)
}

func (s *WhisperServiceAdapterTestSuite) TestSendPublicMessage() {
	// It will succeed because the message is not immediately pushed through the wire
	// but instead put into a batch which will be sent later.
	_, err := s.ws.Send(context.Background(), []byte("abc"), protocol.SendOptions{
		ChatOptions: protocol.ChatOptions{
			ChatName: "test-name",
		},
	})
	s.Require().NoError(err)
}

func TestWhisperServiceAdapterWithPFSTestSuite(t *testing.T) {
	suite.Run(t, new(WhisperServiceAdapterWithPFSTestSuite))
}

// WhisperServiceAdapterWithPFSTestSuite runs the same tests as WhisperServiceAdapterTestSuite,
// with an exception that PFS is used. In this struct, there should be only tests exclusively
// testing PFS integration, otherwise, please put them in WhisperServiceAdapterTestSuite.
type WhisperServiceAdapterWithPFSTestSuite struct {
	WhisperServiceAdapterTestSuite
}

func (s *WhisperServiceAdapterWithPFSTestSuite) SetupTest() {
	s.WhisperServiceAdapterTestSuite.SetupTest()

	dir, err := ioutil.TempDir("", "")
	s.Require().NoError(err)
	err = s.ws.InitPFS(dir)
	s.Require().NoError(err)
}
