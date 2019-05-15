package client

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestContactTypeMarshal(t *testing.T) {
	ct := ContactPublicRoom
	data, err := json.Marshal(ct)
	require.NoError(t, err)
	require.Equal(t, `"ContactPublicRoom"`, string(data))
}

func TestContactMarshalUnmarshal(t *testing.T) {
	privateKey, _ := crypto.GenerateKey()
	publicKeyStr := fmt.Sprintf("%#x", crypto.FromECDSAPub(&privateKey.PublicKey))

	testCases := []struct {
		name   string
		c      Contact
		result string
	}{
		{
			name: "ContactPublicRoom",
			c: Contact{
				Name: "status",
				Type: ContactPublicRoom,
			},
			result: `{"name":"status","type":"ContactPublicRoom","state":0,"topic":""}`,
		},
		{
			name: "ContactPublicKey",
			c: Contact{
				Name:      "user1",
				Type:      ContactPublicKey,
				PublicKey: &privateKey.PublicKey,
			},
			result: fmt.Sprintf(`{"name":"user1","type":"ContactPublicKey","state":0,"topic":"","public_key":"%s"}`, publicKeyStr),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(tc.c)
			require.NoError(t, err)
			require.Equal(t, tc.result, string(data))

			var c Contact

			err = json.Unmarshal(data, &c)
			require.NoError(t, err)
			require.Equal(t, tc.c, c)
		})
	}
}
