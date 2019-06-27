package transport

import (
	"context"
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/pkg/errors"
)

func randomItem(items []string) string {
	l := len(items)
	return items[rand.Intn(l)]
}

// DialOpts used in Dial function.
type DialOpts struct {
	// PollInterval is used for time.Ticker. Must be greated then zero.
	PollInterval time.Duration
}

// Dial selected peer and wait until it is connected.
func Dial(ctx context.Context, srv server, peer string, opts DialOpts) error {
	if opts.PollInterval == 0 {
		return errors.New("poll interval cannot be zero")
	}
	if err := srv.AddPeer(peer); err != nil {
		return err
	}
	parsed, err := enode.ParseV4(peer)
	if err != nil {
		return err
	}
	connected, err := srv.Connected(parsed.ID())
	if err != nil {
		return err
	}
	if connected {
		return nil
	}
	period := time.NewTicker(opts.PollInterval)
	defer period.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-period.C:
			connected, err := srv.Connected(parsed.ID())
			if err != nil {
				return err
			}
			if connected {
				return nil
			}
		}
	}
}
