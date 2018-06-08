package terraform

import (
	"strings"
)

// PathCacheKey returns a cache key for a module path.
func PathCacheKey(path []string) string {
	return strings.Join(path, "|")
}
