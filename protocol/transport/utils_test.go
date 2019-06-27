package transport

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestDialNewPeer(t *testing.T) {
	peer := "enode://2759d79ed966eef0b507480c0d217e471287cc708627e99b94b79ecba6c0cc53d86570bd2d8a3fa9df8c7f26d48876f7f5ce7b8d2aa1ef9531ac557f9ac40d2b@10.223.1.3:30303"
	node, err := enode.ParseV4(peer)
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	srv := NewMockServer(ctrl)
	srv.EXPECT().AddPeer(peer).Return(nil)
	srv.EXPECT().Connected(node.ID()).Return(true, nil)
	require.NoError(t, Dial(context.Background(), srv, peer, DialOpts{PollInterval: 100 * time.Millisecond}))
}

func TestDialExitsOnError(t *testing.T) {
	peer := "enode://2759d79ed966eef0b507480c0d217e471287cc708627e99b94b79ecba6c0cc53d86570bd2d8a3fa9df8c7f26d48876f7f5ce7b8d2aa1ef9531ac557f9ac40d2b@10.223.1.3:30303"
	err := errors.New("test")
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	srv := NewMockServer(ctrl)
	srv.EXPECT().AddPeer(peer).Return(err)
	require.EqualError(t, Dial(context.Background(), srv, peer, DialOpts{PollInterval: 100 * time.Millisecond}), err.Error())
}
