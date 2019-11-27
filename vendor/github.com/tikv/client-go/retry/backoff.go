// Copyright 2018 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package retry

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/tikv/client-go/metrics"
)

const (
	// NoJitter makes the backoff sequence strict exponential.
	NoJitter = 1 + iota
	// FullJitter applies random factors to strict exponential.
	FullJitter
	// EqualJitter is also randomized, but prevents very short sleeps.
	EqualJitter
	// DecorrJitter increases the maximum jitter based on the last random value.
	DecorrJitter
)

// NewBackoffFn creates a backoff func which implements exponential backoff with
// optional jitters.
// See http://www.awsarchitectureblog.com/2015/03/backoff.html
func NewBackoffFn(base, cap, jitter int) func(ctx context.Context) int {
	if base < 2 {
		// Top prevent panic in 'rand.Intn'.
		base = 2
	}
	attempts := 0
	lastSleep := base
	return func(ctx context.Context) int {
		var sleep int
		switch jitter {
		case NoJitter:
			sleep = expo(base, cap, attempts)
		case FullJitter:
			v := expo(base, cap, attempts)
			sleep = rand.Intn(v)
		case EqualJitter:
			v := expo(base, cap, attempts)
			sleep = v/2 + rand.Intn(v/2)
		case DecorrJitter:
			sleep = int(math.Min(float64(cap), float64(base+rand.Intn(lastSleep*3-base))))
		}
		log.Debugf("backoff base %d, sleep %d", base, sleep)
		select {
		case <-time.After(time.Duration(sleep) * time.Millisecond):
		case <-ctx.Done():
		}

		attempts++
		lastSleep = sleep
		return lastSleep
	}
}

func expo(base, cap, n int) int {
	return int(math.Min(float64(cap), float64(base)*math.Pow(2.0, float64(n))))
}

// BackoffType is the retryable error type.
type BackoffType int

// Back off types.
const (
	BoTiKVRPC BackoffType = iota
	BoTxnLock
	BoTxnLockFast
	BoPDRPC
	BoRegionMiss
	BoUpdateLeader
	BoServerBusy
)

func (t BackoffType) createFn() func(context.Context) int {
	switch t {
	case BoTiKVRPC:
		return NewBackoffFn(100, 2000, EqualJitter)
	case BoTxnLock:
		return NewBackoffFn(200, 3000, EqualJitter)
	case BoTxnLockFast:
		return NewBackoffFn(50, 3000, EqualJitter)
	case BoPDRPC:
		return NewBackoffFn(500, 3000, EqualJitter)
	case BoRegionMiss:
		// change base time to 2ms, because it may recover soon.
		return NewBackoffFn(2, 500, NoJitter)
	case BoUpdateLeader:
		return NewBackoffFn(1, 10, NoJitter)
	case BoServerBusy:
		return NewBackoffFn(2000, 10000, EqualJitter)
	}
	return nil
}

func (t BackoffType) String() string {
	switch t {
	case BoTiKVRPC:
		return "tikvRPC"
	case BoTxnLock:
		return "txnLock"
	case BoTxnLockFast:
		return "txnLockFast"
	case BoPDRPC:
		return "pdRPC"
	case BoRegionMiss:
		return "regionMiss"
	case BoUpdateLeader:
		return "updateLeader"
	case BoServerBusy:
		return "serverBusy"
	}
	return ""
}

// Maximum total sleep time(in ms) for kv/cop commands.
const (
	CopBuildTaskMaxBackoff         = 5000
	TsoMaxBackoff                  = 15000
	ScannerNextMaxBackoff          = 20000
	BatchGetMaxBackoff             = 20000
	CopNextMaxBackoff              = 20000
	GetMaxBackoff                  = 20000
	PrewriteMaxBackoff             = 20000
	CleanupMaxBackoff              = 20000
	GcOneRegionMaxBackoff          = 20000
	GcResolveLockMaxBackoff        = 100000
	DeleteRangeOneRegionMaxBackoff = 100000
	RawkvMaxBackoff                = 20000
	SplitRegionBackoff             = 20000
)

// CommitMaxBackoff is max sleep time of the 'commit' command
var CommitMaxBackoff = 41000

// Backoffer is a utility for retrying queries.
type Backoffer struct {
	ctx context.Context

	fn         map[BackoffType]func(context.Context) int
	maxSleep   int
	totalSleep int
	errors     []error
	types      []BackoffType
}

// txnStartKey is a key for transaction start_ts info in context.Context.
const txnStartKey = "_txn_start_key"

// NewBackoffer creates a Backoffer with maximum sleep time(in ms).
func NewBackoffer(ctx context.Context, maxSleep int) *Backoffer {
	return &Backoffer{
		ctx:      ctx,
		maxSleep: maxSleep,
	}
}

// Backoff sleeps a while base on the BackoffType and records the error message.
// It returns a retryable error if total sleep time exceeds maxSleep.
func (b *Backoffer) Backoff(typ BackoffType, err error) error {
	select {
	case <-b.ctx.Done():
		return err
	default:
	}

	metrics.BackoffCounter.WithLabelValues(typ.String()).Inc()
	// Lazy initialize.
	if b.fn == nil {
		b.fn = make(map[BackoffType]func(context.Context) int)
	}
	f, ok := b.fn[typ]
	if !ok {
		f = typ.createFn()
		b.fn[typ] = f
	}

	b.totalSleep += f(b.ctx)
	b.types = append(b.types, typ)

	var startTs interface{}
	if ts := b.ctx.Value(txnStartKey); ts != nil {
		startTs = ts
	}
	log.Debugf("%v, retry later(totalsleep %dms, maxsleep %dms), type: %s, txn_start_ts: %v", err, b.totalSleep, b.maxSleep, typ.String(), startTs)

	b.errors = append(b.errors, errors.Errorf("%s at %s", err.Error(), time.Now().Format(time.RFC3339Nano)))
	if b.maxSleep > 0 && b.totalSleep >= b.maxSleep {
		errMsg := fmt.Sprintf("backoffer.maxSleep %dms is exceeded, errors:", b.maxSleep)
		for i, err := range b.errors {
			// Print only last 3 errors for non-DEBUG log levels.
			if log.GetLevel() == log.DebugLevel || i >= len(b.errors)-3 {
				errMsg += "\n" + err.Error()
			}
		}
		log.Warn(errMsg)
		// Use the first backoff type to generate a MySQL error.
		return errors.New(b.types[0].String())
	}
	return nil
}

func (b *Backoffer) String() string {
	if b.totalSleep == 0 {
		return ""
	}
	return fmt.Sprintf(" backoff(%dms %v)", b.totalSleep, b.types)
}

// Clone creates a new Backoffer which keeps current Backoffer's sleep time and errors, and shares
// current Backoffer's context.
func (b *Backoffer) Clone() *Backoffer {
	return &Backoffer{
		ctx:        b.ctx,
		maxSleep:   b.maxSleep,
		totalSleep: b.totalSleep,
		errors:     b.errors,
	}
}

// Fork creates a new Backoffer which keeps current Backoffer's sleep time and errors, and holds
// a child context of current Backoffer's context.
func (b *Backoffer) Fork() (*Backoffer, context.CancelFunc) {
	ctx, cancel := context.WithCancel(b.ctx)
	return &Backoffer{
		ctx:        ctx,
		maxSleep:   b.maxSleep,
		totalSleep: b.totalSleep,
		errors:     b.errors[:len(b.errors):len(b.errors)],
	}, cancel
}

// GetContext returns the associated context.
func (b *Backoffer) GetContext() context.Context {
	return b.ctx
}

// TotalSleep returns the total sleep time of the backoffer.
func (b *Backoffer) TotalSleep() time.Duration {
	return time.Duration(b.totalSleep) * time.Millisecond
}
