package protocol

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	testMessageBytes  = []byte(`["~#c4",["2","text/plain","~:public-group-user-message",154593077368201,1545930773682,["^ ","~:chat-id","testing-adamb","~:text","2"]]]`)
	testMessageStruct = StatusMessage{
		Text:      "2",
		ContentT:  "text/plain",
		MessageT:  "public-group-user-message",
		Clock:     154593077368201,
		Timestamp: 1545930773682,
		Content:   StatusMessageContent{"testing-adamb", "2"},
	}
)

func TestDecodeMessage(t *testing.T) {
	val, err := DecodeMessage(testMessageBytes)
	require.NoError(t, err)
	require.EqualValues(t, testMessageStruct, val)
}

func TestEncodeMessage(t *testing.T) {
	data, err := EncodeMessage(testMessageStruct)
	require.NoError(t, err)
	require.Equal(t, testMessageBytes, data)
}
