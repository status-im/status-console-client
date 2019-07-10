package client

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestChatTypeMarshal(t *testing.T) {
	ct := PublicChat
	data, err := json.Marshal(ct)
	require.NoError(t, err)
	require.Equal(t, `"PublicChat"`, string(data))
}

func TestChatMarshalUnmarshal(t *testing.T) {
	privateKey, _ := crypto.GenerateKey()
	publicKeyStr := fmt.Sprintf("%#x", crypto.FromECDSAPub(&privateKey.PublicKey))

	testCases := []struct {
		name   string
		c      Chat
		result string
	}{
		{
			name: "PublicChat",
			c: Chat{
				ID:        "status",
				Name:      "status",
				Type:      PublicChat,
				Timestamp: 20,
				UpdatedAt: 21,
			},
			result: `{"id":"status","name":"status","type":"PublicChat","timestamp":20,"updatedAt":21,"active":false,"color":"","deletedAtClockValue":0,"unviewedMessageCount":0,"lastClockValue":0,"lastMessageContentType":"","lastMessageContent":""}`,
		},
		{
			name: "ChatPublicKey",
			c: Chat{
				Name:      "user1",
				Type:      OneToOneChat,
				PublicKey: &privateKey.PublicKey,
			},
			result: fmt.Sprintf(`{"id":"","name":"user1","type":"OneToOneChat","timestamp":0,"updatedAt":0,"active":false,"color":"","deletedAtClockValue":0,"unviewedMessageCount":0,"lastClockValue":0,"lastMessageContentType":"","lastMessageContent":"","public_key":"%s"}`, publicKeyStr),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(tc.c)
			require.NoError(t, err)
			require.Equal(t, tc.result, string(data))

			var c Chat

			err = json.Unmarshal(data, &c)
			require.NoError(t, err)
			require.Equal(t, tc.c, c)
		})
	}
}
