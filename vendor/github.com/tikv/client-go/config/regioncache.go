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

import "time"

// RegionCache contains the configurations for region cache.
type RegionCache struct {
	BTreeDegree int
	CacheTTL    time.Duration
}

// DefaultRegionCache returns the default region cache config.
func DefaultRegionCache() RegionCache {
	return RegionCache{
		BTreeDegree: 32,
		CacheTTL:    10 * time.Minute,
	}
}
