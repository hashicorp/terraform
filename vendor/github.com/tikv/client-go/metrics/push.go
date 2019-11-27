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

package metrics

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	log "github.com/sirupsen/logrus"
)

// PushMetrics pushes metrics to Prometheus Pushgateway.
// Note:
// * Normally, you need to start a goroutine to push metrics: `go
//   PushMetrics(...)`
// * `instance` should be global identical -- NO 2 processes share a same
//   `instance`.
// * `job` is used to distinguish different workloads, DO NOT use too many `job`
//   labels since there are grafana panels that groups by `job`.
func PushMetrics(ctx context.Context, addr string, interval time.Duration, job, instance string) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		err := push.New(addr, job).Grouping("instance", instance).Gatherer(prometheus.DefaultGatherer).Push()
		if err != nil {
			log.Errorf("cannot push metrics to prometheus pushgateway: %v", err)
		}
	}
}
