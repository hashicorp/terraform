// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gensupport

import "time"

type BackoffStrategy interface {
	// Pause returns the duration of the next pause before a retry should be attempted.
	Pause() time.Duration

	// Reset restores the strategy to its initial state.
	Reset()
}

type ExponentialBackoff struct {
	BasePause time.Duration
	nextPause time.Duration
}

func (eb *ExponentialBackoff) Pause() time.Duration {
	if eb.nextPause == 0 {
		eb.Reset()
	}

	d := eb.nextPause
	eb.nextPause *= 2
	return d
}

func (eb *ExponentialBackoff) Reset() {
	eb.nextPause = eb.BasePause
}
