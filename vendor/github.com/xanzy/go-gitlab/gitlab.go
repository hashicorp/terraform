//
// Copyright 2015, Sander van Harmelen
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package gitlab

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/go-querystring/query"
)

const (
	libraryVersion = "0.1.1"
	defaultBaseURL = "https://gitlab.com/api/v3/"
	userAgent      = "go-gitlab/" + libraryVersion
)

// tokenType represents a token type within GitLab.
//
// GitLab API docs: http://doc.gitlab.com/ce/api/
type tokenType int

// List of available token type
//
// GitLab API docs: http://doc.gitlab.com/ce/api/
const (
	privateToken tokenType = iota
	oAuthToken
)

// AccessLevelValue represents a permission level within GitLab.
//
// GitLab API docs: http://doc.gitlab.com/ce/permissions/permissions.html
type AccessLevelValue int

// List of available access levels
//
// GitLab API docs: http://doc.gitlab.com/ce/permissions/permissions.html
const (
	GuestPermissions     AccessLevelValue = 10
	ReporterPermissions  AccessLevelValue = 20
	DeveloperPermissions AccessLevelValue = 30
	MasterPermissions    AccessLevelValue = 40
	OwnerPermission      AccessLevelValue = 50
)

// NotificationLevelValue represents a notification level within Gitlab.
//
// GitLab API docs: http://doc.gitlab.com/ce/api/
type NotificationLevelValue int

// List of available notification levels
//
// GitLab API docs: http://doc.gitlab.com/ce/api/
const (
	DisabledNotifications NotificationLevelValue = iota
	ParticipatingNotifications
	WatchNotifications
	GlobalNotifications
	MentionNotifications
)

// VisibilityLevelValue represents a visibility level within GitLab.
//
// GitLab API docs: http://doc.gitlab.com/ce/api/
type VisibilityLevelValue int

// List of available visibility levels
//
// GitLab API docs: http://doc.gitlab.com/ce/api/
const (
	PrivateVisibility  VisibilityLevelValue = 0
	InternalVisibility VisibilityLevelValue = 10
	PublicVisibility   VisibilityLevelValue = 20
)

// A Client manages communication with the GitLab API.
type Client struct {
	// HTTP client used to communicate with the API.
	client *http.Client

	// Base URL for API requests. Defaults to the public GitLab API, but can be
	// set to a domain endpoint to use with aself hosted GitLab server. baseURL
	// should always be specified with a trailing slash.
	baseURL *url.URL

	// token type used to make authenticated API calls.
	tokenType tokenType

	// token used to make authenticated API calls.
	token string

	// User agent used when communicating with the GitLab API.
	UserAgent string

	// Services used for talking to different parts of the GitLab API.
	Branches        *BranchesService
	Builds          *BuildsService
	Commits         *CommitsService
	DeployKeys      *DeployKeysService
	Groups          *GroupsService
	Issues          *IssuesService
	Labels          *LabelsService
	MergeRequests   *MergeRequestsService
	Milestones      *MilestonesService
	Namespaces      *NamespacesService
	Notes           *NotesService
	Projects        *ProjectsService
	ProjectSnippets *ProjectSnippetsService
	Repositories    *RepositoriesService
	RepositoryFiles *RepositoryFilesService
	Services        *ServicesService
	Session         *SessionService
	Settings        *SettingsService
	SystemHooks     *SystemHooksService
	Tags            *TagsService
	Users           *UsersService
}

// ListOptions specifies the optional parameters to various List methods that
// support pagination.
type ListOptions struct {
	// For paginated result sets, page of results to retrieve.
	Page int `url:"page,omitempty" json:"page,omitempty"`

	// For paginated result sets, the number of results to include per page.
	PerPage int `url:"per_page,omitempty" json:"per_page,omitempty"`
}

// NewClient returns a new GitLab API client. If a nil httpClient is
// provided, http.DefaultClient will be used. To use API methods which require
// authentication, provide a valid private token.
func NewClient(httpClient *http.Client, token string) *Client {
	return newClient(httpClient, privateToken, token)
}

// NewOAuthClient returns a new GitLab API client. If a nil httpClient is
// provided, http.DefaultClient will be used. To use API methods which require
// authentication, provide a valid oauth token.
func NewOAuthClient(httpClient *http.Client, token string) *Client {
	return newClient(httpClient, oAuthToken, token)
}

func newClient(httpClient *http.Client, tokenType tokenType, token string) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	c := &Client{client: httpClient, tokenType: tokenType, token: token, UserAgent: userAgent}
	if err := c.SetBaseURL(defaultBaseURL); err != nil {
		// should never happen since defaultBaseURL is our constant
		panic(err)
	}

	c.Branches = &BranchesService{client: c}
	c.Builds = &BuildsService{client: c}
	c.Commits = &CommitsService{client: c}
	c.DeployKeys = &DeployKeysService{client: c}
	c.Groups = &GroupsService{client: c}
	c.Issues = &IssuesService{client: c}
	c.Labels = &LabelsService{client: c}
	c.MergeRequests = &MergeRequestsService{client: c}
	c.Milestones = &MilestonesService{client: c}
	c.Notes = &NotesService{client: c}
	c.Namespaces = &NamespacesService{client: c}
	c.Projects = &ProjectsService{client: c}
	c.ProjectSnippets = &ProjectSnippetsService{client: c}
	c.Repositories = &RepositoriesService{client: c}
	c.RepositoryFiles = &RepositoryFilesService{client: c}
	c.Services = &ServicesService{client: c}
	c.Session = &SessionService{client: c}
	c.Settings = &SettingsService{client: c}
	c.SystemHooks = &SystemHooksService{client: c}
	c.Tags = &TagsService{client: c}
	c.Users = &UsersService{client: c}

	return c
}

// BaseURL return a copy of the baseURL.
func (c *Client) BaseURL() *url.URL {
	u := *c.baseURL
	return &u
}

// SetBaseURL sets the base URL for API requests to a custom endpoint. urlStr
// should always be specified with a trailing slash.
func (c *Client) SetBaseURL(urlStr string) error {
	// Make sure the given URL end with a slash
	if !strings.HasSuffix(urlStr, "/") {
		urlStr += "/"
	}

	var err error
	c.baseURL, err = url.Parse(urlStr)
	return err
}

// NewRequest creates an API request. A relative URL path can be provided in
// urlStr, in which case it is resolved relative to the base URL of the Client.
// Relative URL paths should always be specified without a preceding slash. If
// specified, the value pointed to by body is JSON encoded and included as the
// request body.
func (c *Client) NewRequest(method, path string, opt interface{}) (*http.Request, error) {
	u := *c.baseURL
	// Set the encoded opaque data
	u.Opaque = c.baseURL.Path + path

	if opt != nil {
		q, err := query.Values(opt)
		if err != nil {
			return nil, err
		}
		u.RawQuery = q.Encode()

	}

	req := &http.Request{
		Method:     method,
		URL:        &u,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
		Host:       u.Host,
	}

	if method == "POST" || method == "PUT" {
		bodyBytes, err := json.Marshal(opt)
		if err != nil {
			return nil, err
		}
		bodyReader := bytes.NewReader(bodyBytes)

		u.RawQuery = ""
		req.Body = ioutil.NopCloser(bodyReader)
		req.ContentLength = int64(bodyReader.Len())
		req.Header.Set("Content-Type", "application/json")
	}

	req.Header.Set("Accept", "application/json")

	switch c.tokenType {
	case privateToken:
		req.Header.Set("PRIVATE-TOKEN", c.token)
	case oAuthToken:
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}

	return req, nil
}

// Response is a GitLab API response. This wraps the standard http.Response
// returned from GitLab and provides convenient access to things like
// pagination links.
type Response struct {
	*http.Response

	// These fields provide the page values for paginating through a set of
	// results.  Any or all of these may be set to the zero value for
	// responses that are not part of a paginated set, or for which there
	// are no additional pages.

	NextPage  int
	PrevPage  int
	FirstPage int
	LastPage  int
}

// newResponse creats a new Response for the provided http.Response.
func newResponse(r *http.Response) *Response {
	response := &Response{Response: r}
	response.populatePageValues()
	return response
}

// populatePageValues parses the HTTP Link response headers and populates the
// various pagination link values in the Reponse.
func (r *Response) populatePageValues() {
	if links, ok := r.Response.Header["Link"]; ok && len(links) > 0 {
		for _, link := range strings.Split(links[0], ",") {
			segments := strings.Split(strings.TrimSpace(link), ";")

			// link must at least have href and rel
			if len(segments) < 2 {
				continue
			}

			// ensure href is properly formatted
			if !strings.HasPrefix(segments[0], "<") || !strings.HasSuffix(segments[0], ">") {
				continue
			}

			// try to pull out page parameter
			url, err := url.Parse(segments[0][1 : len(segments[0])-1])
			if err != nil {
				continue
			}
			page := url.Query().Get("page")
			if page == "" {
				continue
			}

			for _, segment := range segments[1:] {
				switch strings.TrimSpace(segment) {
				case `rel="next"`:
					r.NextPage, _ = strconv.Atoi(page)
				case `rel="prev"`:
					r.PrevPage, _ = strconv.Atoi(page)
				case `rel="first"`:
					r.FirstPage, _ = strconv.Atoi(page)
				case `rel="last"`:
					r.LastPage, _ = strconv.Atoi(page)
				}

			}
		}
	}
}

// Do sends an API request and returns the API response. The API response is
// JSON decoded and stored in the value pointed to by v, or returned as an
// error if an API error has occurred. If v implements the io.Writer
// interface, the raw response body will be written to v, without attempting to
// first decode it.
func (c *Client) Do(req *http.Request, v interface{}) (*Response, error) {
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	response := newResponse(resp)

	err = CheckResponse(resp)
	if err != nil {
		// even though there was an error, we still return the response
		// in case the caller wants to inspect it further
		return response, err
	}

	if v != nil {
		if w, ok := v.(io.Writer); ok {
			_, err = io.Copy(w, resp.Body)
		} else {
			err = json.NewDecoder(resp.Body).Decode(v)
		}
	}
	return response, err
}

// Helper function to accept and format both the project ID or name as project
// identifier for all API calls.
func parseID(id interface{}) (string, error) {
	switch v := id.(type) {
	case int:
		return strconv.Itoa(v), nil
	case string:
		return v, nil
	default:
		return "", fmt.Errorf("invalid ID type %#v, the ID must be an int or a string", id)
	}
}

// An ErrorResponse reports one or more errors caused by an API request.
//
// GitLab API docs:
// http://doc.gitlab.com/ce/api/README.html#data-validation-and-error-reporting
type ErrorResponse struct {
	Response *http.Response // HTTP response that caused this error
	Message  string         `json:"message"` // error message
	Errors   []Error        `json:"errors"`  // more detail on individual errors
}

func (r *ErrorResponse) Error() string {
	path, _ := url.QueryUnescape(r.Response.Request.URL.Opaque)
	ru := fmt.Sprintf("%s://%s%s", r.Response.Request.URL.Scheme, r.Response.Request.URL.Host, path)

	return fmt.Sprintf("%v %s: %d %v %+v",
		r.Response.Request.Method, ru, r.Response.StatusCode, r.Message, r.Errors)
}

// An Error reports more details on an individual error in an ErrorResponse.
// These are the possible validation error codes:
//
//     missing:
//         resource does not exist
//     missing_field:
//         a required field on a resource has not been set
//     invalid:
//         the formatting of a field is invalid
//     already_exists:
//         another resource has the same valid as this field
//
// GitLab API docs:
// http://doc.gitlab.com/ce/api/README.html#data-validation-and-error-reporting
type Error struct {
	Resource string `json:"resource"` // resource on which the error occurred
	Field    string `json:"field"`    // field on which the error occurred
	Code     string `json:"code"`     // validation error code
}

func (e *Error) Error() string {
	return fmt.Sprintf("%v error caused by %v field on %v resource",
		e.Code, e.Field, e.Resource)
}

// CheckResponse checks the API response for errors, and returns them if
// present.  A response is considered an error if it has a status code outside
// the 200 range.  API error responses are expected to have either no response
// body, or a JSON response body that maps to ErrorResponse.  Any other
// response body will be silently ignored.
func CheckResponse(r *http.Response) error {
	if c := r.StatusCode; 200 <= c && c <= 299 {
		return nil
	}
	errorResponse := &ErrorResponse{Response: r}
	data, err := ioutil.ReadAll(r.Body)
	if err == nil && data != nil {
		json.Unmarshal(data, errorResponse)
	}
	return errorResponse
}

// Bool is a helper routine that allocates a new bool value
// to store v and returns a pointer to it.
func Bool(v bool) *bool {
	p := new(bool)
	*p = v
	return p
}

// Int is a helper routine that allocates a new int32 value
// to store v and returns a pointer to it, but unlike Int32
// its argument value is an int.
func Int(v int) *int {
	p := new(int)
	*p = v
	return p
}

// String is a helper routine that allocates a new string value
// to store v and returns a pointer to it.
func String(v string) *string {
	p := new(string)
	*p = v
	return p
}

// AccessLevel is a helper routine that allocates a new AccessLevelValue
// to store v and returns a pointer to it.
func AccessLevel(v AccessLevelValue) *AccessLevelValue {
	p := new(AccessLevelValue)
	*p = v
	return p
}

// VisibilityLevel is a helper routine that allocates a new VisibilityLevelValue
// to store v and returns a pointer to it.
func VisibilityLevel(v VisibilityLevelValue) *VisibilityLevelValue {
	p := new(VisibilityLevelValue)
	*p = v
	return p
}
