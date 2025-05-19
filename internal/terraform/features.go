// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import "os"

// This file holds feature flags for the next release

var flagWarnOutputErrors = os.Getenv("TF_WARN_OUTPUT_ERRORS") != ""
