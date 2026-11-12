// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package schema

import (
	"time"
)

const TimeoutKey = "e2bfb730-ecaa-11e6-8f88-34363bc7c4c0"
const TimeoutsConfigKey = "timeouts"

const (
	TimeoutCreate  = "create"
	TimeoutRead    = "read"
	TimeoutUpdate  = "update"
	TimeoutDelete  = "delete"
	TimeoutDefault = "default"
)

type ResourceTimeout struct {
	Create, Read, Update, Delete, Default *time.Duration
}
