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

package store

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"
)

// ExecDetails contains execution detail information.
type ExecDetails struct {
	CalleeAddress string
	ProcessTime   time.Duration
	WaitTime      time.Duration
	BackoffTime   time.Duration
	RequestCount  int
	TotalKeys     int64
	ProcessedKeys int64
	CommitDetail  *CommitDetails
}

// CommitDetails contains commit detail information.
type CommitDetails struct {
	GetCommitTsTime   time.Duration
	PrewriteTime      time.Duration
	CommitTime        time.Duration
	LocalLatchTime    time.Duration
	TotalBackoffTime  time.Duration
	ResolveLockTime   int64
	WriteKeys         int
	WriteSize         int
	PrewriteRegionNum int32
	TxnRetry          int
}

// String implements the fmt.Stringer interface.
func (d ExecDetails) String() string {
	parts := make([]string, 0, 6)
	if d.ProcessTime > 0 {
		parts = append(parts, fmt.Sprintf("process_time:%vs", d.ProcessTime.Seconds()))
	}
	if d.WaitTime > 0 {
		parts = append(parts, fmt.Sprintf("wait_time:%vs", d.WaitTime.Seconds()))
	}
	if d.BackoffTime > 0 {
		parts = append(parts, fmt.Sprintf("backoff_time:%vs", d.BackoffTime.Seconds()))
	}
	if d.RequestCount > 0 {
		parts = append(parts, fmt.Sprintf("request_count:%d", d.RequestCount))
	}
	if d.TotalKeys > 0 {
		parts = append(parts, fmt.Sprintf("total_keys:%d", d.TotalKeys))
	}
	if d.ProcessedKeys > 0 {
		parts = append(parts, fmt.Sprintf("processed_keys:%d", d.ProcessedKeys))
	}
	commitDetails := d.CommitDetail
	if commitDetails != nil {
		if commitDetails.PrewriteTime > 0 {
			parts = append(parts, fmt.Sprintf("prewrite_time:%vs", commitDetails.PrewriteTime.Seconds()))
		}
		if commitDetails.CommitTime > 0 {
			parts = append(parts, fmt.Sprintf("commit_time:%vs", commitDetails.CommitTime.Seconds()))
		}
		if commitDetails.GetCommitTsTime > 0 {
			parts = append(parts, fmt.Sprintf("get_commit_ts_time:%vs", commitDetails.GetCommitTsTime.Seconds()))
		}
		if commitDetails.TotalBackoffTime > 0 {
			parts = append(parts, fmt.Sprintf("total_backoff_time:%vs", commitDetails.TotalBackoffTime.Seconds()))
		}
		resolveLockTime := atomic.LoadInt64(&commitDetails.ResolveLockTime)
		if resolveLockTime > 0 {
			parts = append(parts, fmt.Sprintf("resolve_lock_time:%vs", time.Duration(resolveLockTime).Seconds()))
		}
		if commitDetails.LocalLatchTime > 0 {
			parts = append(parts, fmt.Sprintf("local_latch_wait_time:%vs", commitDetails.LocalLatchTime.Seconds()))
		}
		if commitDetails.WriteKeys > 0 {
			parts = append(parts, fmt.Sprintf("write_keys:%d", commitDetails.WriteKeys))
		}
		if commitDetails.WriteSize > 0 {
			parts = append(parts, fmt.Sprintf("write_size:%d", commitDetails.WriteSize))
		}
		prewriteRegionNum := atomic.LoadInt32(&commitDetails.PrewriteRegionNum)
		if prewriteRegionNum > 0 {
			parts = append(parts, fmt.Sprintf("prewrite_region:%d", prewriteRegionNum))
		}
		if commitDetails.TxnRetry > 0 {
			parts = append(parts, fmt.Sprintf("txn_retry:%d", commitDetails.TxnRetry))
		}
	}
	return strings.Join(parts, " ")
}
