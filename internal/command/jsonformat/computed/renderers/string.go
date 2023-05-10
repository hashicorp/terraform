// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package renderers

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"
)

type evaluatedString struct {
	String string
	Json   interface{}

	IsMultiline bool
	IsNull      bool
}

func evaluatePrimitiveString(value interface{}, opts computed.RenderHumanOpts) evaluatedString {
	if value == nil {
		return evaluatedString{
			String: opts.Colorize.Color("[dark_gray]null[reset]"),
			IsNull: true,
		}
	}

	str := value.(string)

	if strings.HasPrefix(str, "{") || strings.HasPrefix(str, "[") {
		var jv interface{}
		if err := json.Unmarshal([]byte(str), &jv); err == nil {
			return evaluatedString{
				String: str,
				Json:   jv,
			}
		}
	}

	if strings.Contains(str, "\n") {
		return evaluatedString{
			String:      strings.TrimSpace(str),
			IsMultiline: true,
		}
	}

	return evaluatedString{
		String: str,
	}
}

func (e evaluatedString) RenderSimple() string {
	if e.IsNull {
		return e.String
	}
	return fmt.Sprintf("%q", e.String)
}
