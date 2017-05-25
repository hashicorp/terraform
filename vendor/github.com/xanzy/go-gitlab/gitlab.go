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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
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
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/
type tokenType int

// List of available token type
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/
const (
	privateToken tokenType = iota
	oAuthToken
)

// AccessLevelValue represents a permission level within GitLab.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/permissions/permissions.md
type AccessLevelValue int

// List of available access levels
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/permissions/permissions.md
const (
	GuestPermissions     AccessLevelValue = 10
	ReporterPermissions  AccessLevelValue = 20
	DeveloperPermissions AccessLevelValue = 30
	MasterPermissions    AccessLevelValue = 40
	OwnerPermission      AccessLevelValue = 50
)

// NotificationLevelValue represents a notification level.
type NotificationLevelValue int

// String implements the fmt.Stringer interface.
func (l NotificationLevelValue) String() string {
	return notificationLevelNames[l]
}

// MarshalJSON implements the json.Marshaler interface.
func (l NotificationLevelValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.String())
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (l *NotificationLevelValue) UnmarshalJSON(data []byte) error {
	var raw interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	switch raw := raw.(type) {
	case float64:
		*l = NotificationLevelValue(raw)
	case string:
		*l = notificationLevelTypes[raw]
	default:
		return fmt.Errorf("json: cannot unmarshal %T into Go value of type %T", raw, *l)
	}

	return nil
}

// List of valid notification levels.
const (
	DisabledNotificationLevel NotificationLevelValue = iota
	ParticipatingNotificationLevel
	WatchNotificationLevel
	GlobalNotificationLevel
	MentionNotificationLevel
	CustomNotificationLevel
)

var notificationLevelNames = [...]string{
	"disabled",
	"participating",
	"watch",
	"global",
	"mention",
	"custom",
}

var notificationLevelTypes = map[string]NotificationLevelValue{
	"disabled":      DisabledNotificationLevel,
	"participating": ParticipatingNotificationLevel,
	"watch":         WatchNotificationLevel,
	"global":        GlobalNotificationLevel,
	"mention":       MentionNotificationLevel,
	"custom":        CustomNotificationLevel,
}

// VisibilityLevelValue represents a visibility level within GitLab.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/
type VisibilityLevelValue int

// List of available visibility levels
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/
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
	Branches             *BranchesService
	BuildVariables       *BuildVariablesService
	Builds               *BuildsService
	Commits              *CommitsService
	DeployKeys           *DeployKeysService
	Groups               *GroupsService
	Issues               *IssuesService
	Labels               *LabelsService
	MergeRequests        *MergeRequestsService
	Milestones           *MilestonesService
	Namespaces           *NamespacesService
	Notes                *NotesService
	NotificationSettings *NotificationSettingsService
	Projects             *ProjectsService
	ProjectSnippets      *ProjectSnippetsService
	Pipelines            *PipelinesService
	Repositories         *RepositoriesService
	RepositoryFiles      *RepositoryFilesService
	Services             *ServicesService
	Session              *SessionService
	Settings             *SettingsService
	SystemHooks          *SystemHooksService
	Tags                 *TagsService
	TimeStats            *TimeStatsService
	Users                *UsersService
	Version              *VersionService
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
	c.BuildVariables = &BuildVariablesService{client: c}
	c.Builds = &BuildsService{client: c}
	c.Commits = &CommitsService{client: c}
	c.DeployKeys = &DeployKeysService{client: c}
	c.Groups = &GroupsService{client: c}
	c.Issues = &IssuesService{client: c}
	c.Labels = &LabelsService{client: c}
	c.MergeRequests = &MergeRequestsService{client: c}
	c.Milestones = &MilestonesService{client: c}
	c.Namespaces = &NamespacesService{client: c}
	c.Notes = &NotesService{client: c}
	c.NotificationSettings = &NotificationSettingsService{client: c}
	c.Projects = &ProjectsService{client: c}
	c.ProjectSnippets = &ProjectSnippetsService{client: c}
	c.Pipelines = &PipelinesService{client: c}
	c.Repositories = &RepositoriesService{client: c}
	c.RepositoryFiles = &RepositoryFilesService{client: c}
	c.Services = &ServicesService{client: c}
	c.Session = &SessionService{client: c}
	c.Settings = &SettingsService{client: c}
	c.SystemHooks = &SystemHooksService{client: c}
	c.Tags = &TagsService{client: c}
	c.TimeStats = &TimeStatsService{client: c}
	c.Users = &UsersService{client: c}
	c.Version = &VersionService{client: c}

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
func (c *Client) NewRequest(method, path string, opt interface{}, options []OptionFunc) (*http.Request, error) {
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

	for _, fn := range options {
		if err := fn(req); err != nil {
			return nil, err
		}
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
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/README.md#data-validation-and-error-reporting
type ErrorResponse struct {
	Response *http.Response
	Message  string
}

func (e *ErrorResponse) Error() string {
	path, _ := url.QueryUnescape(e.Response.Request.URL.Opaque)
	u := fmt.Sprintf("%s://%s%s", e.Response.Request.URL.Scheme, e.Response.Request.URL.Host, path)
	return fmt.Sprintf("%s %s: %d %s", e.Response.Request.Method, u, e.Response.StatusCode, e.Message)
}

// CheckResponse checks the API response for errors, and returns them if present.
func CheckResponse(r *http.Response) error {
	switch r.StatusCode {
	case 200, 201, 304:
		return nil
	}

	errorResponse := &ErrorResponse{Response: r}
	data, err := ioutil.ReadAll(r.Body)
	if err == nil && data != nil {
		var raw interface{}
		if err := json.Unmarshal(data, &raw); err != nil {
			errorResponse.Message = "failed to parse unknown error format"
		}

		errorResponse.Message = parseError(raw)
	}

	return errorResponse
}

// Format:
// {
//     "message": {
//         "<property-name>": [
//             "<error-message>",
//             "<error-message>",
//             ...
//         ],
//         "<embed-entity>": {
//             "<property-name>": [
//                 "<error-message>",
//                 "<error-message>",
//                 ...
//             ],
//         }
//     },
//     "error": "<error-message>"
// }
func parseError(raw interface{}) string {
	switch raw := raw.(type) {
	case string:
		return raw

	case []interface{}:
		var errs []string
		for _, v := range raw {
			errs = append(errs, parseError(v))
		}
		return fmt.Sprintf("[%s]", strings.Join(errs, ", "))

	case map[string]interface{}:
		var errs []string
		for k, v := range raw {
			errs = append(errs, fmt.Sprintf("{%s: %s}", k, parseError(v)))
		}
		sort.Strings(errs)
		return strings.Join(errs, ", ")

	default:
		return fmt.Sprintf("failed to parse unexpected error type: %T", raw)
	}
}

// OptionFunc can be passed to all API requests to make the API call as if you were
// another user, provided your private token is from an administrator account.
//
// GitLab docs: https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/README.md#sudo
type OptionFunc func(*http.Request) error

// WithSudo takes either a username or user ID and sets the SUDO request header
func WithSudo(uid interface{}) OptionFunc {
	return func(req *http.Request) error {
		switch uid := uid.(type) {
		case int:
			req.Header.Set("SUDO", strconv.Itoa(uid))
			return nil
		case string:
			req.Header.Set("SUDO", uid)
			return nil
		default:
			return fmt.Errorf("uid must be either a username or user ID")
		}
	}
}

// WithContext runs the request with the provided context
func WithContext(ctx context.Context) OptionFunc {
	return func(req *http.Request) error {
		*req = *req.WithContext(ctx)
		return nil
	}
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

// NotificationLevel is a helper routine that allocates a new NotificationLevelValue
// to store v and returns a pointer to it.
func NotificationLevel(v NotificationLevelValue) *NotificationLevelValue {
	p := new(NotificationLevelValue)
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
