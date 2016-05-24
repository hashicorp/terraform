package azurerm

import "strings"

func String(value interface{}) *string {
	s := value.(string)
	return &s
}

func Int(value interface{}) *int {
	i := value.(int)
	return &i
}

func isOneOf(haystack []string, needle interface{}) bool {
	n := strings.ToLower(needle.(string))
	for _, h := range haystack {
		if h == n {
			return true
		}
	}

	return false
}
