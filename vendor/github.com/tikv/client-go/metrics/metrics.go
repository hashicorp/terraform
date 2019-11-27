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

package metrics

import "github.com/prometheus/client_golang/prometheus"

// Client metrics.
var (
	TxnCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "tikv",
			Subsystem: "client_go",
			Name:      "txn_total",
			Help:      "Counter of created txns.",
		})

	TxnHistogram = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "tikv",
			Subsystem: "client_go",
			Name:      "txn_durations_seconds",
			Help:      "Bucketed histogram of processing txn",
			Buckets:   prometheus.ExponentialBuckets(0.0005, 2, 20),
		},
	)

	SnapshotCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "tikv",
			Subsystem: "client_go",
			Name:      "snapshot_total",
			Help:      "Counter of snapshots.",
		})

	TxnCmdHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "tikv",
			Subsystem: "client_go",
			Name:      "txn_cmd_duration_seconds",
			Help:      "Bucketed histogram of processing time of txn cmds.",
			Buckets:   prometheus.ExponentialBuckets(0.0005, 2, 20),
		}, []string{"type"})

	BackoffCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "tikv",
			Subsystem: "client_go",
			Name:      "backoff_total",
			Help:      "Counter of backoff.",
		}, []string{"type"})

	BackoffHistogram = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "tikv",
			Subsystem: "client_go",
			Name:      "backoff_seconds",
			Help:      "total backoff seconds of a single backoffer.",
			Buckets:   prometheus.ExponentialBuckets(0.0005, 2, 20),
		})

	SendReqHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "tikv",
			Subsystem: "client_go",
			Name:      "request_seconds",
			Help:      "Bucketed histogram of sending request duration.",
			Buckets:   prometheus.ExponentialBuckets(0.0005, 2, 20),
		}, []string{"type", "store"})

	LockResolverCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "tikv",
			Subsystem: "client_go",
			Name:      "lock_resolver_actions_total",
			Help:      "Counter of lock resolver actions.",
		}, []string{"type"})

	RegionErrorCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "tikv",
			Subsystem: "client_go",
			Name:      "region_err_total",
			Help:      "Counter of region errors.",
		}, []string{"type"})

	TxnWriteKVCountHistogram = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "tikv",
			Subsystem: "client_go",
			Name:      "txn_write_kv_num",
			Help:      "Count of kv pairs to write in a transaction.",
			Buckets:   prometheus.ExponentialBuckets(1, 2, 21),
		})

	TxnWriteSizeHistogram = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "tikv",
			Subsystem: "client_go",
			Name:      "txn_write_size_bytes",
			Help:      "Size of kv pairs to write in a transaction.",
			Buckets:   prometheus.ExponentialBuckets(1, 2, 21),
		})

	RawkvCmdHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "tikv",
			Subsystem: "client_go",
			Name:      "rawkv_cmd_seconds",
			Help:      "Bucketed histogram of processing time of rawkv cmds.",
			Buckets:   prometheus.ExponentialBuckets(0.0005, 2, 20),
		}, []string{"type"})

	RawkvSizeHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "tikv",
			Subsystem: "client_go",
			Name:      "rawkv_kv_size_bytes",
			Help:      "Size of key/value to put, in bytes.",
			Buckets:   prometheus.ExponentialBuckets(1, 2, 21),
		}, []string{"type"})

	TxnRegionsNumHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "tikv",
			Subsystem: "client_go",
			Name:      "txn_regions_num",
			Help:      "Number of regions in a transaction.",
			Buckets:   prometheus.ExponentialBuckets(1, 2, 20),
		}, []string{"type"})

	LoadSafepointCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "tikv",
			Subsystem: "client_go",
			Name:      "load_safepoint_total",
			Help:      "Counter of load safepoint.",
		}, []string{"type"})

	SecondaryLockCleanupFailureCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "tikv",
			Subsystem: "client_go",
			Name:      "lock_cleanup_task_total",
			Help:      "failure statistic of secondary lock cleanup task.",
		}, []string{"type"})

	RegionCacheCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "tikv",
			Subsystem: "client_go",
			Name:      "region_cache_operations_total",
			Help:      "Counter of region cache.",
		}, []string{"type", "result"})

	// PendingBatchRequests indicates the number of requests pending in the batch channel.
	PendingBatchRequests = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "tikv",
			Subsystem: "client_go",
			Name:      "pending_batch_requests",
			Help:      "Pending batch requests",
		})

	BatchWaitDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "tikv",
			Subsystem: "client_go",
			Name:      "batch_wait_duration",
			// Min bucket is [0, 1ns).
			Buckets: prometheus.ExponentialBuckets(1, 2, 30),
			Help:    "batch wait duration",
		})

	TSFutureWaitDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "tikv",
			Subsystem: "pdclient",
			Name:      "ts_future_wait_seconds",
			Help:      "Bucketed histogram of seconds cost for waiting timestamp future.",
			Buckets:   prometheus.ExponentialBuckets(0.000005, 2, 18), // 5us ~ 128 ms
		})

	LocalLatchWaitTimeHistogram = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "tidb",
			Subsystem: "tikvclient",
			Name:      "local_latch_wait_seconds",
			Help:      "Wait time of a get local latch.",
			Buckets:   prometheus.ExponentialBuckets(0.0005, 2, 20),
		})
)

// RetLabel returns "ok" when err == nil and "err" when err != nil.
// This could be useful when you need to observe the operation result.
func RetLabel(err error) string {
	if err == nil {
		return "ok"
	}
	return "err"
}

func init() {
	prometheus.MustRegister(TxnCounter)
	prometheus.MustRegister(SnapshotCounter)
	prometheus.MustRegister(TxnHistogram)
	prometheus.MustRegister(TxnCmdHistogram)
	prometheus.MustRegister(BackoffCounter)
	prometheus.MustRegister(BackoffHistogram)
	prometheus.MustRegister(SendReqHistogram)
	prometheus.MustRegister(LockResolverCounter)
	prometheus.MustRegister(RegionErrorCounter)
	prometheus.MustRegister(TxnWriteKVCountHistogram)
	prometheus.MustRegister(TxnWriteSizeHistogram)
	prometheus.MustRegister(RawkvCmdHistogram)
	prometheus.MustRegister(RawkvSizeHistogram)
	prometheus.MustRegister(TxnRegionsNumHistogram)
	prometheus.MustRegister(LoadSafepointCounter)
	prometheus.MustRegister(SecondaryLockCleanupFailureCounter)
	prometheus.MustRegister(RegionCacheCounter)
	prometheus.MustRegister(PendingBatchRequests)
	prometheus.MustRegister(BatchWaitDuration)
	prometheus.MustRegister(TSFutureWaitDuration)
	prometheus.MustRegister(LocalLatchWaitTimeHistogram)
}
