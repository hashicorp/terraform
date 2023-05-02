// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package jsonchecks

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/checks"
)

type checkStatus []byte

func checkStatusForJSON(s checks.Status) checkStatus {
	if ret, ok := checkStatuses[s]; ok {
		return ret
	}
	panic(fmt.Sprintf("unsupported check status %#v", s))
}

func (s checkStatus) MarshalJSON() ([]byte, error) {
	return []byte(s), nil
}

var checkStatuses = map[checks.Status]checkStatus{
	checks.StatusPass:    checkStatus(`"pass"`),
	checks.StatusFail:    checkStatus(`"fail"`),
	checks.StatusError:   checkStatus(`"error"`),
	checks.StatusUnknown: checkStatus(`"unknown"`),
}
