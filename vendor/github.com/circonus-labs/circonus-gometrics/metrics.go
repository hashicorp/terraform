// Copyright 2016 Circonus, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package circonusgometrics

// SetMetricTags sets the tags for the named metric and flags a check update is needed
func (m *CirconusMetrics) SetMetricTags(name string, tags []string) bool {
	return m.check.AddMetricTags(name, tags, false)
}

// AddMetricTags appends tags to any existing tags for the named metric and flags a check update is needed
func (m *CirconusMetrics) AddMetricTags(name string, tags []string) bool {
	return m.check.AddMetricTags(name, tags, true)
}
