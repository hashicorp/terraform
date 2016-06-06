// Package cloudflare implements the CloudFlare v4 API.
package cloudflare

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

const apiURL = "https://api.cloudflare.com/client/v4"

// API holds the configuration for the current API client. A client should not
// be modified concurrently.
type API struct {
	APIKey     string
	APIEmail   string
	BaseURL    string
	headers    http.Header
	httpClient *http.Client
}

// New creates a new CloudFlare v4 API client.
func New(key, email string, opts ...Option) (*API, error) {
	if key == "" || email == "" {
		return nil, errors.New(errEmptyCredentials)
	}

	api := &API{
		APIKey:   key,
		APIEmail: email,
		BaseURL:  apiURL,
		headers:  make(http.Header),
	}

	err := api.parseOptions(opts...)
	if err != nil {
		return nil, errors.Wrap(err, "options parsing failed")
	}

	// Fall back to http.DefaultClient if the package user does not provide
	// their own.
	if api.httpClient == nil {
		api.httpClient = http.DefaultClient
	}

	return api, nil
}

// ZoneIDByName retrieves a zone's ID from the name.
func (api *API) ZoneIDByName(zoneName string) (string, error) {
	res, err := api.ListZones(zoneName)
	if err != nil {
		return "", errors.Wrap(err, "ListZones command failed")
	}
	for _, zone := range res {
		if zone.Name == zoneName {
			return zone.ID, nil
		}
	}
	return "", errors.New("Zone could not be found")
}

// makeRequest makes a HTTP request and returns the body as a byte slice,
// closing it before returnng. params will be serialized to JSON.
func (api *API) makeRequest(method, uri string, params interface{}) ([]byte, error) {
	// Replace nil with a JSON object if needed
	var reqBody io.Reader
	if params != nil {
		json, err := json.Marshal(params)
		if err != nil {
			return nil, errors.Wrap(err, "error marshalling params to JSON")
		}
		reqBody = bytes.NewReader(json)
	} else {
		reqBody = nil
	}

	resp, err := api.request(method, uri, reqBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "could not read response body")
	}

	switch resp.StatusCode {
	case http.StatusOK:
		break
	case http.StatusUnauthorized:
		return nil, errors.Errorf("HTTP status %d: invalid credentials", resp.StatusCode)
	case http.StatusForbidden:
		return nil, errors.Errorf("HTTP status %d: insufficient permissions", resp.StatusCode)
	default:
		var s string
		if body != nil {
			s = string(body)
		}
		return nil, errors.Errorf("HTTP status %d: content %q", resp.StatusCode, s)
	}

	return body, nil
}

// request makes a HTTP request to the given API endpoint, returning the raw
// *http.Response, or an error if one occurred. The caller is responsible for
// closing the response body.
func (api *API) request(method, uri string, reqBody io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, api.BaseURL+uri, reqBody)
	if err != nil {
		return nil, errors.Wrap(err, "HTTP request creation failed")
	}

	// Apply any user-defined headers first.
	req.Header = api.headers
	req.Header.Set("X-Auth-Key", api.APIKey)
	req.Header.Set("X-Auth-Email", api.APIEmail)

	resp, err := api.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "HTTP request failed")
	}

	return resp, nil
}

// ResponseInfo contains a code and message returned by the API as errors or
// informational messages inside the response.
type ResponseInfo struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Response is a template.  There will also be a result struct.  There will be a
// unique response type for each response, which will include this type.
type Response struct {
	Success  bool           `json:"success"`
	Errors   []ResponseInfo `json:"errors"`
	Messages []ResponseInfo `json:"messages"`
}

// ResultInfo contains metadata about the Response.
type ResultInfo struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	Count   int `json:"count"`
	Total   int `json:"total_count"`
}

// User describes a user account.
type User struct {
	ID            string         `json:"id"`
	Email         string         `json:"email"`
	FirstName     string         `json:"first_name"`
	LastName      string         `json:"last_name"`
	Username      string         `json:"username"`
	Telephone     string         `json:"telephone"`
	Country       string         `json:"country"`
	Zipcode       string         `json:"zipcode"`
	CreatedOn     time.Time      `json:"created_on"`
	ModifiedOn    time.Time      `json:"modified_on"`
	APIKey        string         `json:"api_key"`
	TwoFA         bool           `json:"two_factor_authentication_enabled"`
	Betas         []string       `json:"betas"`
	Organizations []Organization `json:"organizations"`
}

// UserResponse wraps a response containing User accounts.
type UserResponse struct {
	Response
	Result User `json:"result"`
}

// Owner describes the resource owner.
type Owner struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	OwnerType string `json:"owner_type"`
}

// DNSRecord represents a DNS record in a zone.
type DNSRecord struct {
	ID         string      `json:"id,omitempty"`
	Type       string      `json:"type,omitempty"`
	Name       string      `json:"name,omitempty"`
	Content    string      `json:"content,omitempty"`
	Proxiable  bool        `json:"proxiable,omitempty"`
	Proxied    bool        `json:"proxied,omitempty"`
	TTL        int         `json:"ttl,omitempty"`
	Locked     bool        `json:"locked,omitempty"`
	ZoneID     string      `json:"zone_id,omitempty"`
	ZoneName   string      `json:"zone_name,omitempty"`
	CreatedOn  time.Time   `json:"created_on,omitempty"`
	ModifiedOn time.Time   `json:"modified_on,omitempty"`
	Data       interface{} `json:"data,omitempty"` // data returned by: SRV, LOC
	Meta       interface{} `json:"meta,omitempty"`
	Priority   int         `json:"priority,omitempty"`
}

// DNSRecordResponse represents the response from the DNS endpoint.
type DNSRecordResponse struct {
	Response
	Result DNSRecord `json:"result"`
}

// DNSListResponse represents the response from the list DNS records endpoint.
type DNSListResponse struct {
	Response
	Result []DNSRecord `json:"result"`
}

// KeylessSSL represents Keyless SSL configuration.
type KeylessSSL struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Host        string    `json:"host"`
	Port        int       `json:"port"`
	Status      string    `json:"success"`
	Enabled     bool      `json:"enabled"`
	Permissions []string  `json:"permissions"`
	CreatedOn   time.Time `json:"created_on"`
	ModifiedOn  time.Time `json:"modifed_on"`
}

// KeylessSSLResponse represents the response from the Keyless SSL endpoint.
type KeylessSSLResponse struct {
	Response
	Result []KeylessSSL `json:"result"`
}

// CustomPage represents a custom page configuration.
type CustomPage struct {
	CreatedOn      string    `json:"created_on"`
	ModifiedOn     time.Time `json:"modified_on"`
	URL            string    `json:"url"`
	State          string    `json:"state"`
	RequiredTokens []string  `json:"required_tokens"`
	PreviewTarget  string    `json:"preview_target"`
	Description    string    `json:"description"`
}

// CustomPageResponse represents the response from the custom pages endpoint.
type CustomPageResponse struct {
	Response
	Result []CustomPage `json:"result"`
}

// WAFPackage represents a WAF package configuration.
type WAFPackage struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	ZoneID        string `json:"zone_id"`
	DetectionMode string `json:"detection_mode"`
	Sensitivity   string `json:"sensitivity"`
	ActionMode    string `json:"action_mode"`
}

// WAFPackagesResponse represents the response from the WAF packages endpoint.
type WAFPackagesResponse struct {
	Response
	Result     []WAFPackage `json:"result"`
	ResultInfo ResultInfo   `json:"result_info"`
}

// WAFRule represents a WAF rule.
type WAFRule struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Priority    string `json:"priority"`
	PackageID   string `json:"package_id"`
	Group       struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"group"`
	Mode         string   `json:"mode"`
	DefaultMode  string   `json:"default_mode"`
	AllowedModes []string `json:"allowed_modes"`
}

// WAFRulesResponse represents the response from the WAF rule endpoint.
type WAFRulesResponse struct {
	Response
	Result     []WAFRule  `json:"result"`
	ResultInfo ResultInfo `json:"result_info"`
}

// PurgeCacheRequest represents the request format made to the purge endpoint.
type PurgeCacheRequest struct {
	Everything bool     `json:"purge_everything,omitempty"`
	Files      []string `json:"files,omitempty"`
	Tags       []string `json:"tags,omitempty"`
}

// PurgeCacheResponse represents the response from the purge endpoint.
type PurgeCacheResponse struct {
	Response
}

// IPRanges contains lists of IPv4 and IPv6 CIDRs
type IPRanges struct {
	IPv4CIDRs []string `json:"ipv4_cidrs"`
	IPv6CIDRs []string `json:"ipv6_cidrs"`
}

// IPsResponse is the API response containing a list of IPs
type IPsResponse struct {
	Response
	Result IPRanges `json:"result"`
}
