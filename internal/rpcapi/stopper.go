// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package rpcapi

import (
	"sync"
)

type stopChan chan struct{}

// stopper allows the RPC API to stop in-progress long-running operations. Each
// operation must add a new stop to the stopper, and remove it if the operation
// completes successfully. If a Stop RPC is received while the operation is
// running, the stops will all be processed, signalling to each operation that
// it should abort.
//
// Each stop is represented by a channel, which is closed to indicate that the
// operation should stop.
type stopper struct {
	stops map[stopChan]struct{}

	mu sync.Mutex
}

func newStopper() *stopper {
	return &stopper{
		stops: make(map[stopChan]struct{}),
	}
}

func (s *stopper) add() stopChan {
	s.mu.Lock()
	defer s.mu.Unlock()

	stop := make(chan struct{})
	s.stops[stop] = struct{}{}

	return stop
}

func (s *stopper) remove(stop stopChan) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.stops, stop)
}

func (s *stopper) stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for stop := range s.stops {
		close(stop)
		delete(s.stops, stop)
	}
}
