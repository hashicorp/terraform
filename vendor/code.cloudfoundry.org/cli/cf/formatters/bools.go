package formatters

import (
	. "code.cloudfoundry.org/cli/cf/i18n"
)

func Allowed(allowed bool) string {
	if allowed {
		return T("allowed")
	}
	return T("disallowed")
}
