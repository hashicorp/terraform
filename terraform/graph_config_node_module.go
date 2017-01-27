package terraform

import (
	"fmt"
	"strings"
)

func modulePrefixStr(p []string) string {
	parts := make([]string, 0, len(p)*2)
	for _, p := range p[1:] {
		parts = append(parts, "module", p)
	}

	return strings.Join(parts, ".")
}

func modulePrefixList(result []string, prefix string) []string {
	if prefix != "" {
		for i, v := range result {
			result[i] = fmt.Sprintf("%s.%s", prefix, v)
		}
	}

	return result
}
