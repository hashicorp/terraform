package metadata

import (
	"net/http"
	"strings"
)

// ParseFromHeaders parses the meta data from the headers
func ParseFromHeaders(headers http.Header) map[string]string {
	metaData := make(map[string]string, 0)
	for k, v := range headers {
		key := strings.ToLower(k)
		prefix := "x-ms-meta-"
		if !strings.HasPrefix(key, prefix) {
			continue
		}

		key = strings.TrimPrefix(key, prefix)
		metaData[key] = v[0]
	}
	return metaData
}
