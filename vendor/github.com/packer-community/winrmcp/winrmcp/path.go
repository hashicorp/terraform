package winrmcp

import (
	"fmt"
	"strings"
)

func winPath(path string) string {
	if len(path) == 0 {
		return path
	}

	if strings.Contains(path, " ") {
		path = fmt.Sprintf("'%s'", strings.Trim(path, "'\""))
	}

	return strings.Replace(path, "/", "\\", -1)
}
