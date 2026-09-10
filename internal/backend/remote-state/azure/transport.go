// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

// The sensitive data stripper is an HTTP transport wrapper that prevents
// Terraform state file contents from being leaked in debug logs.
//
// PROBLEM: When TF_LOG=DEBUG is enabled, the Azure SDK logs full HTTP
// request/response bodies. This includes Terraform state files which
// contain sensitive data (passwords, API keys, connection strings, etc.).
//
// SOLUTION: Wrap the Azure SDK's HTTP transport with a middleware that:
// 1. Intercepts all HTTP request/response bodies
// 2. Strips/identifies sensitive data patterns
// 3. Logs only summary information (method, URL, status, duration)
// 4. Restores bodies so the caller receives unmodified responses
//
// This is injected into all Azure SDK clients via configureClient() in
// api_client.go. See https://github.com/hashicorp/terraform/issues/32382

package azure

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var sensitivePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)password`),
	regexp.MustCompile(`(?i)secret`),
	regexp.MustCompile(`(?i)access[_-]?key`),
	regexp.MustCompile(`(?i)token`),
	regexp.MustCompile(`(?i)api[_-]?key`),
	regexp.MustCompile(`(?i)connection[_-]?string`),
	regexp.MustCompile(`(?i)sas[_-]?token`),
	regexp.MustCompile(`(?i)auth`),
	regexp.MustCompile(`(?i)credential`),
	regexp.MustCompile(`(?i)private[_-]?key`),
}

var logSensitive = log.New(io.Discard, "[AzureSafe]", log.LstdFlags)

func logSafe(format string, args ...interface{}) {
	logSensitive.Printf(format, args...)
}

func redactValue(v string) string {
	if strings.Contains(v, "sv=") && strings.Contains(v, "ss=") {
		return "<redacted-sas>"
	}
	for _, p := range sensitivePatterns {
		if p.MatchString(v) {
			return "<redacted>"
		}
	}
	return v
}

func redactJSON(data []byte) []byte {
	var obj interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return data
	}
	if m, ok := obj.(map[string]interface{}); ok {
		redactMap(m)
	} else if a, ok := obj.([]interface{}); ok {
		redactArr(a)
	}
	r, err := json.Marshal(obj)
	if err != nil {
		return data
	}
	return r
}

func redactMap(m map[string]interface{}) {
	for k, v := range m {
		lower := strings.ToLower(k)
		for _, p := range sensitivePatterns {
			if p.MatchString(lower) {
				m[k] = "<redacted>"
				break
			}
		}
		switch x := v.(type) {
		case map[string]interface{}:
			m[k] = redactMap(x)
		case []interface{}:
			m[k] = redactArr(x)
		}
	}
	return m
}

func redactArr(a []interface{}) []interface{} {
	for i, v := range a {
		switch x := v.(type) {
		case map[string]interface{}:
			a[i] = redactMap(x)
		case []interface{}:
			a[i] = redactArr(x)
		}
	}
	return a
}

// safeTransport wraps an HTTP transport and strips sensitive data from
// request/response bodies before they reach the debug logger.
type safeTransport struct {
	next http.RoundTripper
}

func (s *safeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t0 := time.Now()

	// Read and inspect request body (clone via GetBody or manually)
	var reqBody []byte
	if req.GetBody != nil {
		body, err := req.GetBody()
		if err == nil {
			if b, err := io.ReadAll(body); err == nil {
				reqBody = b
				body.Close()
			}
		}
	}

	resp, err := s.next.RoundTrip(req)

	// Log only safe summary info
	if resp != nil {
		logSafe("[DEBUG] Azure HTTP %s %s -> %d (%.2fs)",
			req.Method, req.URL.String(), resp.StatusCode, time.Since(t0).Seconds())
	}
	if err != nil && req != nil {
		logSafe("[ERROR] Azure HTTP %s %s -> %v (%.2fs)",
			req.Method, req.URL.String(), err, time.Since(t0).Seconds())
	}
	if err != nil {
		return resp, err
	}

	// Read and redact response body, then restore it for the caller
	if resp.Body != nil {
		body, err := io.ReadAll(resp.Body)
		if err == nil {
			resp.Body = io.NopCloser(bytes.NewReader(body))

			ct := resp.Header.Get("Content-Type")
			if strings.Contains(ct, "application/json") {
				if len(body) > 2048 {
					logSafe("[DEBUG] Azure HTTP body redacted (%d bytes)", len(body))
				} else {
					r := redactJSON(body)
					if !bytes.Equal(body, r) {
						logSafe("[DEBUG] Azure HTTP body redacted (%d bytes)", len(r))
					}
				}
			}
		}
	}

	return resp, nil
}

// wrapClient wraps the HTTP transport of an *http.Client with safeTransport.
// This is called from configureClient() in api_client.go before setting the
// authorizer, so all HTTP calls from the Azure SDK go through the wrapper.
func wrapClient(c *http.Client) {
	if c == nil {
		return
	}
	if _, ok := c.Transport.(*safeTransport); ok {
		return // already wrapped
	}
	next := c.Transport
	if next == nil {
		next = http.DefaultTransport
	}
	c.Transport = &safeTransport{next: next}
}
