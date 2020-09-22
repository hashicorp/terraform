package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"strings"
)

// Logger is an interface representing the Logger struct
type Logger interface {
	Printf(format string, args ...interface{})
}

// DefaultLogger is a default struct, which satisfies the Logger interface
type DefaultLogger struct{}

// Printf is a default Printf method
func (DefaultLogger) Printf(format string, args ...interface{}) {
	log.Printf("[DEBUG] "+format, args...)
}

// noopLogger is a default noop logger satisfies the Logger interface
type noopLogger struct{}

// Printf is a default noop method
func (noopLogger) Printf(format string, args ...interface{}) {}

// RoundTripper satisfies the http.RoundTripper interface and is used to
// customize the default http client RoundTripper
type RoundTripper struct {
	// Default http.RoundTripper
	Rt http.RoundTripper
	// Additional request headers to be set (not appended) in all client
	// requests
	headers *http.Header
	// A pointer to a map of headers to be masked in logger
	maskHeaders *map[string]struct{}
	// A custom function to format and mask JSON requests and responses
	FormatJSON func([]byte) (string, error)
	// How many times HTTP connection should be retried until giving up
	MaxRetries int
	// If Logger is not nil, then RoundTrip method will debug the JSON
	// requests and responses
	Logger Logger
}

// List of headers that contain sensitive data.
var defaultSensitiveHeaders = map[string]struct{}{
	"x-auth-token":                    {},
	"x-auth-key":                      {},
	"x-service-token":                 {},
	"x-storage-token":                 {},
	"x-account-meta-temp-url-key":     {},
	"x-account-meta-temp-url-key-2":   {},
	"x-container-meta-temp-url-key":   {},
	"x-container-meta-temp-url-key-2": {},
	"set-cookie":                      {},
	"x-subject-token":                 {},
	"authorization":                   {},
}

// GetDefaultSensitiveHeaders returns the default list of headers to be masked
func GetDefaultSensitiveHeaders() []string {
	headers := make([]string, len(defaultSensitiveHeaders))
	i := 0
	for k := range defaultSensitiveHeaders {
		headers[i] = k
		i++
	}

	return headers
}

// SetSensitiveHeaders sets the list of case insensitive headers to be masked in
// debug log
func (rt *RoundTripper) SetSensitiveHeaders(headers []string) {
	newHeaders := make(map[string]struct{}, len(headers))

	for _, h := range headers {
		newHeaders[h] = struct{}{}
	}

	// this is concurrency safe
	rt.maskHeaders = &newHeaders
}

// SetHeaders sets request headers to be set (not appended) in all client
// requests
func (rt *RoundTripper) SetHeaders(headers http.Header) {
	newHeaders := make(http.Header, len(headers))
	for k, v := range headers {
		s := make([]string, len(v))
		for i, v := range v {
			s[i] = v
		}
		newHeaders[k] = s
	}

	// this is concurrency safe
	rt.headers = &newHeaders
}

func (rt *RoundTripper) hideSensitiveHeadersData(headers http.Header) []string {
	result := make([]string, len(headers))
	headerIdx := 0

	// this is concurrency safe
	v := rt.maskHeaders
	if v == nil {
		v = &defaultSensitiveHeaders
	}
	maskHeaders := *v

	for header, data := range headers {
		v := strings.ToLower(header)
		if _, ok := maskHeaders[v]; ok {
			result[headerIdx] = fmt.Sprintf("%s: %s", header, "***")
		} else {
			result[headerIdx] = fmt.Sprintf("%s: %s", header, strings.Join(data, " "))
		}
		headerIdx++
	}

	return result
}

// formatHeaders converts standard http.Header type to a string with separated headers.
// It will hide data of sensitive headers.
func (rt *RoundTripper) formatHeaders(headers http.Header, separator string) string {
	redactedHeaders := rt.hideSensitiveHeadersData(headers)
	sort.Strings(redactedHeaders)

	return strings.Join(redactedHeaders, separator)
}

// RoundTrip performs a round-trip HTTP request and logs relevant information about it.
func (rt *RoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	defer func() {
		if request.Body != nil {
			request.Body.Close()
		}
	}()

	// for future reference, this is how to access the Transport struct:
	//tlsconfig := rt.Rt.(*http.Transport).TLSClientConfig

	// this is concurrency safe
	h := rt.headers
	if h != nil {
		for k, v := range *h {
			// Set additional request headers
			request.Header[k] = v
		}
	}

	var err error

	if rt.Logger != nil {
		rt.log().Printf("OpenStack Request URL: %s %s", request.Method, request.URL)
		rt.log().Printf("OpenStack Request Headers:\n%s", rt.formatHeaders(request.Header, "\n"))

		if request.Body != nil {
			request.Body, err = rt.logRequest(request.Body, request.Header.Get("Content-Type"))
			if err != nil {
				return nil, err
			}
		}
	}

	// this is concurrency safe
	ort := rt.Rt
	if ort == nil {
		return nil, fmt.Errorf("Rt RoundTripper is nil, aborting")
	}
	response, err := ort.RoundTrip(request)

	// If the first request didn't return a response, retry up to `max_retries`.
	retry := 1
	for response == nil {
		if retry > rt.MaxRetries {
			if rt.Logger != nil {
				rt.log().Printf("OpenStack connection error, retries exhausted. Aborting")
			}
			err = fmt.Errorf("OpenStack connection error, retries exhausted. Aborting. Last error was: %s", err)
			return nil, err
		}

		if rt.Logger != nil {
			rt.log().Printf("OpenStack connection error, retry number %d: %s", retry, err)
		}
		response, err = ort.RoundTrip(request)
		retry += 1
	}

	if rt.Logger != nil {
		rt.log().Printf("OpenStack Response Code: %d", response.StatusCode)
		rt.log().Printf("OpenStack Response Headers:\n%s", rt.formatHeaders(response.Header, "\n"))

		response.Body, err = rt.logResponse(response.Body, response.Header.Get("Content-Type"))
	}

	return response, err
}

// logRequest will log the HTTP Request details.
// If the body is JSON, it will attempt to be pretty-formatted.
func (rt *RoundTripper) logRequest(original io.ReadCloser, contentType string) (io.ReadCloser, error) {
	// Handle request contentType
	if strings.HasPrefix(contentType, "application/json") || (strings.HasPrefix(contentType, "application/") && strings.HasSuffix(contentType, "-json-patch")) {
		var bs bytes.Buffer
		defer original.Close()

		_, err := io.Copy(&bs, original)
		if err != nil {
			return nil, err
		}

		debugInfo, err := rt.formatJSON()(bs.Bytes())
		if err != nil {
			rt.log().Printf("%s", err)
		}
		rt.log().Printf("OpenStack Request Body: %s", debugInfo)

		return ioutil.NopCloser(strings.NewReader(bs.String())), nil
	}

	rt.log().Printf("Not logging because OpenStack request body isn't JSON")
	return original, nil
}

// logResponse will log the HTTP Response details.
// If the body is JSON, it will attempt to be pretty-formatted.
func (rt *RoundTripper) logResponse(original io.ReadCloser, contentType string) (io.ReadCloser, error) {
	if strings.HasPrefix(contentType, "application/json") {
		var bs bytes.Buffer
		defer original.Close()

		_, err := io.Copy(&bs, original)
		if err != nil {
			return nil, err
		}

		debugInfo, err := rt.formatJSON()(bs.Bytes())
		if err != nil {
			rt.log().Printf("%s", err)
		}
		if debugInfo != "" {
			rt.log().Printf("OpenStack Response Body: %s", debugInfo)
		}

		return ioutil.NopCloser(strings.NewReader(bs.String())), nil
	}

	rt.log().Printf("Not logging because OpenStack response body isn't JSON")
	return original, nil
}

func (rt *RoundTripper) formatJSON() func([]byte) (string, error) {
	// this is concurrency safe
	f := rt.FormatJSON
	if f == nil {
		return FormatJSON
	}
	return f
}

func (rt *RoundTripper) log() Logger {
	// this is concurrency safe
	l := rt.Logger
	if l == nil {
		// noop is used, when logger pointer has been set to nil
		return &noopLogger{}
	}
	return l
}

// FormatJSON is a default function to pretty-format a JSON body.
// It will also mask known fields which contain sensitive information.
func FormatJSON(raw []byte) (string, error) {
	var rawData interface{}

	err := json.Unmarshal(raw, &rawData)
	if err != nil {
		return string(raw), fmt.Errorf("unable to parse OpenStack JSON: %s", err)
	}

	data, ok := rawData.(map[string]interface{})
	if !ok {
		pretty, err := json.MarshalIndent(rawData, "", "  ")
		if err != nil {
			return string(raw), fmt.Errorf("unable to re-marshal OpenStack JSON: %s", err)
		}

		return string(pretty), nil
	}

	// Mask known password fields
	if v, ok := data["auth"].(map[string]interface{}); ok {
		// v2 auth methods
		if v, ok := v["passwordCredentials"].(map[string]interface{}); ok {
			v["password"] = "***"
		}
		if v, ok := v["token"].(map[string]interface{}); ok {
			v["id"] = "***"
		}
		// v3 auth methods
		if v, ok := v["identity"].(map[string]interface{}); ok {
			if v, ok := v["password"].(map[string]interface{}); ok {
				if v, ok := v["user"].(map[string]interface{}); ok {
					v["password"] = "***"
				}
			}
			if v, ok := v["application_credential"].(map[string]interface{}); ok {
				v["secret"] = "***"
			}
			if v, ok := v["token"].(map[string]interface{}); ok {
				v["id"] = "***"
			}
		}
	}

	// Mask EC2 access id and body hash
	if v, ok := data["credentials"].(map[string]interface{}); ok {
		var access string
		if s, ok := v["access"]; ok {
			access, _ = s.(string)
			v["access"] = "***"
		}
		if _, ok := v["body_hash"]; ok {
			v["body_hash"] = "***"
		}
		if v, ok := v["headers"].(map[string]interface{}); ok {
			if _, ok := v["Authorization"]; ok {
				if s, ok := v["Authorization"].(string); ok {
					v["Authorization"] = strings.Replace(s, access, "***", -1)
				}
			}
		}
	}

	// Ignore the huge catalog output
	if v, ok := data["token"].(map[string]interface{}); ok {
		if _, ok := v["catalog"]; ok {
			v["catalog"] = "***"
		}
	}

	pretty, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return string(raw), fmt.Errorf("unable to re-marshal OpenStack JSON: %s", err)
	}

	return string(pretty), nil
}
