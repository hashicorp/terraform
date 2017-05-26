// +build go1.8

package request

import (
	"net/http"
)

// NoBody is a http.NoBody reader instructing Go HTTP client to not include
// and body in the HTTP request.
var NoBody = http.NoBody
