package client

import (
	"time"

	"github.com/status-im/status-console-client/protocol/v1"
)

type requester struct {
	oldest protocol.RequestOptions
	newest protocol.RequestOptions
}

func (r requester) options(newest bool) protocol.RequestOptions {
	params := protocol.DefaultRequestOptions()

	if newest && !r.newest.Equal(protocol.RequestOptions{}) {
		params.From = r.newest.From
	} else if !r.oldest.Equal(protocol.RequestOptions{}) {
		params.From = r.oldest.From - int64(protocol.DefaultDurationRequestOptions/time.Second)
		params.To = r.oldest.To
	}

	return params
}

func (r *requester) update(opts protocol.RequestOptions) {
	if r.oldest.From == 0 || r.oldest.From > opts.From {
		r.oldest = opts
	}

	if r.newest.To < opts.To {
		r.newest = opts
	}
}
