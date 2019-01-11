package protocol

import "sync"

type Subscription struct {
	sync.RWMutex

	err  error
	done chan struct{}
}

func NewSubscription() *Subscription {
	return &Subscription{
		done: make(chan struct{}),
	}
}

func (s *Subscription) cancel(err error) {
	s.Lock()
	s.err = err
	s.Unlock()
}

func (s *Subscription) Unsubscribe() {
	close(s.done)
}

func (s *Subscription) Err() error {
	s.RLock()
	defer s.RUnlock()
	return s.err
}

func (s *Subscription) Done() <-chan struct{} {
	return s.done
}
