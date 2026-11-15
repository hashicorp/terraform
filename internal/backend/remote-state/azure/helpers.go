// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

// This file is copied from terraform-provider-azurerm: internal/provider/helpers.go

package azure

import (
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform/internal/backend/backendbase"
)

// logEntry avoids log entries showing up in test output
func logEntry(f string, v ...interface{}) {
	if os.Getenv("TF_LOG") == "" {
		return
	}

	if os.Getenv("TF_ACC") != "" {
		return
	}

	// Sanitize the log format string and arguments to remove sensitive values
	sanitized := sanitizeLogArgs(f, v...)
	log.Printf(sanitized...)
}

// logSafePrintf is like log.Printf but always sanitizes sensitive values from the output.
// This should be used for all debug output in the azure backend to prevent leaking
// storage account keys, SAS tokens, client secrets, OIDC tokens, etc.
func logSafePrintf(f string, v ...interface{}) {
	sanitized := sanitizeLogArgs(f, v...)
	log.Printf(sanitized...)
}

// sensitiveURLParamKeys are URL query parameter keys that may contain sensitive values
// and should be redacted when logging URLs.
var sensitiveURLParamKeys = []string{
	// SAS token parameters
	"sig",       // SAS signature
	"sv",        // SAS version (not strictly sensitive but part of SAS)
	"signed_url",
	"signed_resource",
	"sp",        // SAS permissions
	"sr",        // SAS resource
	"sip",       // SAS IP range
	"sdd",       // SAS start IP
	"sse",       // SAS expiry
	"skoid",     // SAS key object ID
	// Storage access key (when passed as query param in some scenarios)
	"key",
	"access_key",
	// OIDC / auth
	"token",
	"bearer",
	"code",
	"client_secret",
}

// sensitiveHeaderNames are HTTP header names (lowercased) that should be redacted.
var sensitiveHeaderNames = map[string]bool{
	"authorization":       true,
	"x-ms-key":            true,
	"x-ms-version":        false, // not sensitive
	"x-ms-copy-source":    true,  // may contain SAS token
	"x-ms-source-url":     true,  // may contain SAS token
	"x-ms-encryption-key": true,
}

// sasTokenRegex matches common SAS token patterns in a URL query string.
// SAS tokens in Azure Storage URLs appear as query parameters like ?sv=...&sig=...
var sasTokenRegex = regexp.MustCompile(`[?&](sv|sig|sp|sr|sip|sdd|sse|skoid|spr|se|sas)=[^&]+`)

// bearerTokenRegex matches Bearer tokens in strings (e.g., Authorization headers).
var bearerTokenRegex = regexp.MustCompile(`(?i)Bearer\s+[A-Za-z0-9\-_.~+\/]+=*`)

// jwtRegex matches JWT-style tokens (three base64url segments separated by dots).
var jwtRegex = regexp.MustCompile(`[A-Za-z0-9\-_]+\.[A-Za-z0-9\-_]+\.[A-Za-z0-9\-_]+`)

// sanitizeURL redacts sensitive query parameters from a URL string.
// This prevents SAS tokens (which are appended to request URLs) from leaking in logs.
func sanitizeURL(rawURL string) string {
	if rawURL == "" {
		return rawURL
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		// If we can't parse it, try to use regex-based sanitization
		return sanitizeURLString(rawURL)
	}

	q := u.Query()
	modified := false
	for _, key := range sensitiveURLParamKeys {
		if q.Has(key) {
			q.Set(key, "[REDACTED]")
			modified = true
		}
	}

	// Also check all values - any value that looks like a SAS token signature
	// (very long base64-like string) should be redacted
	for key, values := range q {
		for i, v := range values {
			if len(v) > 40 && isSensitiveValue(v) {
				values[i] = "[REDACTED]"
				modified = true
			}
		}
		q[key] = values
	}

	if modified {
		u.RawQuery = q.Encode()
	}

	return u.String()
}

// sanitizeURLString is a fallback that sanitizes a raw URL string using regex
// when proper URL parsing fails.
func sanitizeURLString(raw string) string {
	// Replace each sensitive query parameter value
	for _, key := range sensitiveURLParamKeys {
		pattern := regexp.MustCompile(`([?&])` + regexp.QuoteMeta(key) + `=[^&]*`)
		raw = pattern.ReplaceAllString(raw, "${1}"+key+"=[REDACTED]")
	}
	return raw
}

// sanitizeHeaders creates a copy of the given headers with sensitive values redacted.
func sanitizeHeaders(headers http.Header) http.Header {
	if headers == nil {
		return nil
	}
	sanitized := make(http.Header, len(headers))
	for k, v := range headers {
		lowerKey := strings.ToLower(k)
		if sensitiveHeaderNames[lowerKey] {
			// Redact each value
			redacted := make([]string, len(v))
			for i, val := range v {
				if val != "" {
					redacted[i] = "[REDACTED]"
				}
			}
			sanitized[k] = redacted
		} else {
			// Also redact values that contain bearer tokens
			redacted := make([]string, len(v))
			for i, val := range v {
				redacted[i] = sanitizeValue(val)
			}
			sanitized[k] = redacted
		}
	}
	return sanitized
}

// sanitizeValue replaces known sensitive patterns in a single value string.
func sanitizeValue(s string) string {
	s = bearerTokenRegex.ReplaceAllString(s, "Bearer [REDACTED]")
	s = jwtRegex.ReplaceAllString(s, "[REDACTED_JWT]")
	return s
}

// sanitizeLogArgs applies sanitizeURL to all arguments that look like URLs,
// and sanitizeValue to all string arguments.
func sanitizeLogArgs(format string, args ...interface{}) []interface{} {
	// Always sanitize the format string itself (it may contain embedded values)
	format = sanitizeValue(format)

	result := make([]interface{}, len(args))
	for i, arg := range args {
		if str, ok := arg.(string); ok {
			if isURL(str) {
				result[i] = sanitizeURL(str)
			} else {
				result[i] = sanitizeValue(str)
			}
		} else {
			result[i] = arg
		}
	}

	// Re-append format as first arg since log.Printf takes format then args
	// But log.Printf format is the first parameter, so we need to handle this differently
	// Actually log.Printf(f string, v ...interface{}) - so format is separate
	// We return [format] + args, but the caller does log.Printf(sanitized...)
	// Let's fix this: return [format] concatenated with args
	return append([]interface{}{format}, result...)
}

// isURL checks if a string looks like a URL.
func isURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

// isSensitiveValue heuristically checks if a string value appears to be a sensitive secret.
func isSensitiveValue(s string) bool {
	// Azure SAS signatures are typically 64+ chars of base64
	if len(s) >= 40 {
		// Check if it looks like a base64-encoded value (common for secrets)
		base64Like := true
		for _, c := range s {
			if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '+' || c == '/' || c == '=' || c == '-' || c == '_') {
				base64Like = false
				break
			}
		}
		if base64Like {
			return true
		}
	}
	return false
}

func decodeCertificate(clientCertificate string) ([]byte, error) {
	var pfx []byte
	if clientCertificate != "" {
		out := make([]byte, base64.StdEncoding.DecodedLen(len(clientCertificate)))
		n, err := base64.StdEncoding.Decode(out, []byte(clientCertificate))
		if err != nil {
			return pfx, fmt.Errorf("could not decode client certificate data: %v", err)
		}
		pfx = out[:n]
	}
	return pfx, nil
}

func getOidcToken(d *backendbase.SDKLikeData) (*string, error) {
	idToken := strings.TrimSpace(d.String("oidc_token"))

	if path := d.String("oidc_token_file_path"); path != "" {
		fileTokenRaw, err := os.ReadFile(path)

		if err != nil {
			return nil, fmt.Errorf("reading OIDC Token from file %q: %v", path, err)
		}

		fileToken := strings.TrimSpace(string(fileTokenRaw))

		if idToken != "" && idToken != fileToken {
			return nil, fmt.Errorf("mismatch between supplied OIDC token and supplied OIDC token file contents - please either remove one or ensure they match")
		}

		idToken = fileToken
	}

	if d.Bool("use_aks_workload_identity") && os.Getenv("AZURE_FEDERATED_TOKEN_FILE") != "" {
		path := os.Getenv("AZURE_FEDERATED_TOKEN_FILE")
		fileTokenRaw, err := os.ReadFile(os.Getenv("AZURE_FEDERATED_TOKEN_FILE"))

		if err != nil {
			return nil, fmt.Errorf("reading OIDC Token from file %q provided by AKS Workload Identity: %v", path, err)
		}

		fileToken := strings.TrimSpace(string(fileTokenRaw))

		if idToken != "" && idToken != fileToken {
			return nil, fmt.Errorf("mismatch between supplied OIDC token and OIDC token file contents provided by AKS Workload Identity - please either remove one, ensure they match, or disable use_aks_workload_identity")
		}

		idToken = fileToken
	}

	return &idToken, nil
}

func getClientId(d *backendbase.SDKLikeData) (*string, error) {
	clientId := strings.TrimSpace(d.String("client_id"))

	if path := d.String("client_id_file_path"); path != "" {
		fileClientIdRaw, err := os.ReadFile(path)

		if err != nil {
			return nil, fmt.Errorf("reading Client ID from file %q: %v", path, err)
		}

		fileClientId := strings.TrimSpace(string(fileClientIdRaw))

		if clientId != "" && clientId != fileClientId {
			return nil, fmt.Errorf("mismatch between supplied Client ID and supplied Client ID file contents - please either remove one or ensure they match")
		}

		clientId = fileClientId
	}

	if d.Bool("use_aks_workload_identity") && os.Getenv("AZURE_CLIENT_ID") != "" {
		aksClientId := os.Getenv("AZURE_CLIENT_ID")
		if clientId != "" && clientId != aksClientId {
			return nil, fmt.Errorf("mismatch between supplied Client ID and that provided by AKS Workload Identity - please remove, ensure they match, or disable use_aks_workload_identity")
		}
		clientId = aksClientId
	}

	return &clientId, nil
}

func getClientSecret(d *backendbase.SDKLikeData) (*string, error) {
	clientSecret := strings.TrimSpace(d.String("client_secret"))

	if path := d.String("client_secret_file_path"); path != "" {
		fileSecretRaw, err := os.ReadFile(path)

		if err != nil {
			return nil, fmt.Errorf("reading Client Secret from file %q: %v", path, err)
		}

		fileSecret := strings.TrimSpace(string(fileSecretRaw))

		if clientSecret != "" && clientSecret != fileSecret {
			return nil, fmt.Errorf("mismatch between supplied Client Secret and supplied Client Secret file contents - please either remove one or ensure they match")
		}

		clientSecret = fileSecret
	}

	return &clientSecret, nil
}

func getTenantId(d *backendbase.SDKLikeData) (*string, error) {
	tenantId := strings.TrimSpace(d.String("tenant_id"))

	if d.Bool("use_aks_workload_identity") && os.Getenv("AZURE_TENANT_ID") != "" {
		aksTenantId := os.Getenv("AZURE_TENANT_ID")
		if tenantId != "" && tenantId != aksTenantId {
			return nil, fmt.Errorf("mismatch between supplied Tenant ID and that provided by AKS Workload Identity - please remove, ensure they match, or disable use_aks_workload_identity")
		}
		tenantId = aksTenantId
	}

	return &tenantId, nil
}
