package udnssdk

// udnssdk - a golang sdk for the ultradns REST service.
// 2015-07-03 - jmasseo@gmail.com

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"
)

const (
	libraryVersion = "0.1"
	// DefaultTestBaseURL returns the URL for UltraDNS's test restapi endpoint
	DefaultTestBaseURL = "https://test-restapi.ultradns.com/"
	// DefaultLiveBaseURL returns the URL for UltraDNS's production restapi endpoint
	DefaultLiveBaseURL = "https://restapi.ultradns.com/"

	userAgent = "udnssdk-go/" + libraryVersion

	apiVersion = "v1"
)

// QueryInfo wraps a query request
type QueryInfo struct {
	Q       string `json:"q"`
	Sort    string `json:"sort"`
	Reverse bool   `json:"reverse"`
	Limit   int    `json:"limit"`
}

// ResultInfo wraps the list metadata for an index response
type ResultInfo struct {
	TotalCount    int `json:"totalCount"`
	Offset        int `json:"offset"`
	ReturnedCount int `json:"returnedCount"`
}

// Client wraps our general-purpose Service Client
type Client struct {
	// This is our client structure.
	HTTPClient *http.Client

	// UltraDNS makes a call to an authorization API using your username and
	// password, returning an 'Access Token' and a 'Refresh Token'.
	// Our use case does not require the refresh token, but we should implement
	// for completeness.
	AccessToken  string
	RefreshToken string
	Username     string
	Password     string
	BaseURL      string
	UserAgent    string

	// Accounts API
	Accounts *AccountsService
	// Probe Alerts API
	Alerts *AlertsService
	// Directional Pools API
	DirectionalPools *DirectionalPoolsService
	// Events API
	Events *EventsService
	// Notifications API
	Notifications *NotificationsService
	// Probes API
	Probes *ProbesService
	// Resource Record Sets API
	RRSets *RRSetsService
	// Tasks API
	Tasks *TasksService
}

// NewClient returns a new ultradns API client.
func NewClient(username, password, BaseURL string) (*Client, error) {
	accesstoken, refreshtoken, err := GetAuthTokens(username, password, BaseURL)
	if err != nil {
		return nil, err
	}
	c := &Client{
		AccessToken:  accesstoken,
		RefreshToken: refreshtoken,
		Username:     username,
		Password:     password,
		HTTPClient:   &http.Client{},
		BaseURL:      BaseURL,
		UserAgent:    userAgent,
	}
	c.Accounts = &AccountsService{client: c}
	c.Alerts = &AlertsService{client: c}
	c.DirectionalPools = &DirectionalPoolsService{client: c}
	c.Events = &EventsService{client: c}
	c.Notifications = &NotificationsService{client: c}
	c.Probes = &ProbesService{client: c}
	c.RRSets = &RRSetsService{client: c}
	c.Tasks = &TasksService{client: c}
	return c, nil
}

// newStubClient returns a new ultradns API client.
func newStubClient(username, password, BaseURL, accesstoken, refreshtoken string) (*Client, error) {
	c := &Client{
		AccessToken:  accesstoken,
		RefreshToken: refreshtoken,
		Username:     username,
		Password:     password,
		HTTPClient:   &http.Client{},
		BaseURL:      BaseURL,
		UserAgent:    userAgent,
	}
	c.Accounts = &AccountsService{client: c}
	c.Alerts = &AlertsService{client: c}
	c.DirectionalPools = &DirectionalPoolsService{client: c}
	c.Events = &EventsService{client: c}
	c.Notifications = &NotificationsService{client: c}
	c.Probes = &ProbesService{client: c}
	c.RRSets = &RRSetsService{client: c}
	c.Tasks = &TasksService{client: c}
	return c, nil
}

// NewAuthRequest creates an Authorization request to get an access and refresh token.
//
// {
//   "tokenType":"Bearer",
//   "refreshToken":"48472efcdce044c8850ee6a395c74a7872932c7112",
//    "accessToken":"b91d037c75934fc89a9f43fe4a",
//    "expiresIn":"3600",
//    "expires_in":"3600"
// }

// AuthResponse wraps the response to an auth request
type AuthResponse struct {
	TokenType    string `json:"tokenType"`
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    string `json:"expiresIn"`
}

// GetAuthTokens requests by username, password & base URL, returns the access-token & refresh-token, or a possible error
func GetAuthTokens(username, password, BaseURL string) (string, string, error) {
	res, err := http.PostForm(fmt.Sprintf("%s/%s/authorization/token", BaseURL, apiVersion), url.Values{"grant_type": {"password"}, "username": {username}, "password": {password}})

	if err != nil {
		return "", "", err
	}

	//response := &Response{Response: res}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", "", err
	}
	err = CheckAuthResponse(res, body)
	if err != nil {
		return "", "", err
	}

	var authr AuthResponse
	err = json.Unmarshal(body, &authr)
	if err != nil {
		return string(body), "JSON Decode Error", err
	}
	return authr.AccessToken, authr.RefreshToken, err
}

// NewRequest creates an API request.
// The path is expected to be a relative path and will be resolved
// according to the BaseURL of the Client. Paths should always be specified without a preceding slash.
func (c *Client) NewRequest(method, path string, payload interface{}) (*http.Request, error) {
	url := c.BaseURL + fmt.Sprintf("%s/%s", apiVersion, path)

	body := new(bytes.Buffer)
	if payload != nil {
		err := json.NewEncoder(body).Encode(payload)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("User-Agent", c.UserAgent)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.AccessToken))
	req.Header.Add("Token", fmt.Sprintf("Bearer %s", c.AccessToken))

	return req, nil
}

func (c *Client) get(path string, v interface{}) (*Response, error) {
	return c.Do("GET", path, nil, v)
}

func (c *Client) post(path string, payload, v interface{}) (*Response, error) {
	return c.Do("POST", path, payload, v)
}

func (c *Client) put(path string, payload, v interface{}) (*Response, error) {
	return c.Do("PUT", path, payload, v)
}

func (c *Client) delete(path string, payload interface{}) (*Response, error) {
	return c.Do("DELETE", path, payload, nil)
}

// Do sends an API request and returns the API response.
// The API response is JSON decoded and stored in the value pointed by v,
// or returned as an error if an API error has occurred.
// If v implements the io.Writer interface, the raw response body will be written to v,
// without attempting to decode it.
func (c *Client) Do(method, path string, payload, v interface{}) (*Response, error) {
	req, err := c.NewRequest(method, path, payload)
	if err != nil {
		return nil, err
	}
	log.Printf("[DEBUG] HTTP Request: %+v\n", req)
	res, err := c.HTTPClient.Do(req)
	log.Printf("[DEBUG] HTTP Response: %+v\n", res)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	origresponse := &Response{Response: res}

	var nres *http.Response
	nres = res
	if res.StatusCode == 202 {
		// This is a deferred task.
		tid := TaskID(res.Header.Get("X-Task-Id"))
		log.Printf("[DEBUG] Received Async Task %+v..  will retry...\n", tid)
		// TODO: Sane Configuration for timeouts / retries
		timeout := 5
		waittime := 5 * time.Second
		i := 0
		breakmeout := false
		for i < timeout || breakmeout {
			myt, statusres, err := c.Tasks.Find(tid)
			if err != nil {
				return origresponse, err
			}
			log.Printf("[DEBUG] Task ID: %+v Retry: %d Status Code: %s\n", tid, i, myt.TaskStatusCode)
			switch myt.TaskStatusCode {
			case "COMPLETE":
				// Yay
				tres, err := c.Tasks.FindResultByTask(myt)
				if err != nil {
					return origresponse, err
				}
				nres = tres.Response
				breakmeout = true
			case "PENDING", "IN_PROCESS":
				i = i + 1
				time.Sleep(waittime)
				continue
			case "ERROR":
				return statusres, err

			}
		}
	}
	response := &Response{Response: nres}

	err = CheckResponse(nres)
	if err != nil {
		return response, err
	}

	if v != nil {
		if w, ok := v.(io.Writer); ok {
			io.Copy(w, res.Body)
		} else {
			err = json.NewDecoder(res.Body).Decode(v)
		}
	}

	return response, err
}

// A Response represents an API response.
type Response struct {
	*http.Response
}

// ErrorResponse represents an error caused by an API request.
// Example:
// {"errorCode":60001,"errorMessage":"invalid_grant:Invalid username & password combination.","error":"invalid_grant","error_description":"60001: invalid_grant:Invalid username & password combination."}
type ErrorResponse struct {
	Response         *http.Response // HTTP response that caused this error
	ErrorCode        int            `json:"errorCode"`    //  error code
	ErrorMessage     string         `json:"errorMessage"` // human-readable message
	ErrorStr         string         `json:"error"`
	ErrorDescription string         `json:"error_description"`
}

// ErrorResponseList wraps an HTTP response that has a list of errors
type ErrorResponseList struct {
	Response  *http.Response // HTTP response that caused this error
	Responses []ErrorResponse
}

// Error implements the error interface.
func (r ErrorResponse) Error() string {
	return fmt.Sprintf("%v %v: %d %d %v",
		r.Response.Request.Method, r.Response.Request.URL,
		r.Response.StatusCode, r.ErrorCode, r.ErrorMessage)
}

func (r ErrorResponseList) Error() string {
	return fmt.Sprintf("%v %v: %d %d %v",
		r.Response.Request.Method, r.Response.Request.URL,
		r.Response.StatusCode, r.Responses[0].ErrorCode, r.Responses[0].ErrorMessage)
}

// CheckAuthResponse checks the API response for errors, and returns them if so
func CheckAuthResponse(r *http.Response, body []byte) error {
	if code := r.StatusCode; 200 <= code && code <= 299 {
		return nil
	}

	// Attempt marshaling to ErrorResponse
	var er ErrorResponse
	err := json.Unmarshal(body, &er)
	if err == nil {
		er.Response = r
		return er
	}

	// Attempt marshaling to ErrorResponseList
	var ers []ErrorResponse
	err = json.Unmarshal(body, &ers)
	if err == nil {
		return &ErrorResponseList{Response: r, Responses: ers}
	}

	return fmt.Errorf("Response had non-successful status: %d, but could not extract error from body: %+v", r.StatusCode, body)
}

// CheckResponse checks the API response for errors, and returns them if present.
// A response is considered an error if the status code is different than 2xx. Specific requests
// may have additional requirements, but this is sufficient in most of the cases.
func CheckResponse(r *http.Response) error {
	if code := r.StatusCode; 200 <= code && code <= 299 {
		return nil
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}

	// Attempt marshaling to ErrorResponse
	var er ErrorResponse
	err = json.Unmarshal(body, &er)
	if err == nil {
		er.Response = r
		return er
	}

	// Attempt marshaling to ErrorResponseList
	var ers []ErrorResponse
	err = json.Unmarshal(body, &ers)
	if err == nil {
		return &ErrorResponseList{Response: r, Responses: ers}
	}

	return fmt.Errorf("Response had non-successful status: %d, but could not extract error from body: %+v", r.StatusCode, body)
}
