// Copyright 2019 PingCAP, Inc.
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

package config

// Raw is rawkv configurations.
type Raw struct {
	// MaxScanLimit is the maximum scan limit for rawkv Scan.
	MaxScanLimit int

	// MaxBatchPutSize is the maximum size limit for rawkv each batch put request.
	MaxBatchPutSize int

	// BatchPairCount is the maximum limit for rawkv each batch get/delete request.
	BatchPairCount int
}

// DefaultRaw returns default rawkv configuration.
func DefaultRaw() Raw {
	return Raw{
		MaxScanLimit:    10240,
		MaxBatchPutSize: 16 * 1024,
		BatchPairCount:  512,
	}
}
