package formatters

import "strconv"

func InstanceMemoryLimit(limit int64) string {
	if limit == -1 {
		return "unlimited"
	}

	return strconv.FormatInt(limit, 10) + "M"
}
