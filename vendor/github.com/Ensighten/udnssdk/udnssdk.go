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
	"time"

	"golang.org/x/oauth2"

	"github.com/Ensighten/udnssdk/passwordcredentials"
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
	Config     *passwordcredentials.Config

	BaseURL   string
	UserAgent string

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
	ctx := oauth2.NoContext
	conf := NewConfig(username, password, BaseURL)

	c := &Client{
		HTTPClient: conf.Client(ctx),
		BaseURL:    BaseURL,
		UserAgent:  userAgent,
		Config:     conf,
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
func newStubClient(username, password, BaseURL, clientID, clientSecret string) (*Client, error) {
	c := &Client{
		HTTPClient: &http.Client{},
		BaseURL:    BaseURL,
		UserAgent:  userAgent,
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

	return req, nil
}

func (c *Client) get(path string, v interface{}) (*http.Response, error) {
	return c.Do("GET", path, nil, v)
}

func (c *Client) post(path string, payload, v interface{}) (*http.Response, error) {
	return c.Do("POST", path, payload, v)
}

func (c *Client) put(path string, payload, v interface{}) (*http.Response, error) {
	return c.Do("PUT", path, payload, v)
}

func (c *Client) delete(path string, payload interface{}) (*http.Response, error) {
	return c.Do("DELETE", path, payload, nil)
}

// Do sends an API request and returns the API response.
// The API response is JSON decoded and stored in the value pointed by v,
// or returned as an error if an API error has occurred.
// If v implements the io.Writer interface, the raw response body will be written to v,
// without attempting to decode it.
func (c *Client) Do(method, path string, payload, v interface{}) (*http.Response, error) {
	hc := c.HTTPClient
	req, err := c.NewRequest(method, path, payload)
	if err != nil {
		return nil, err
	}
	log.Printf("[DEBUG] HTTP Request: %+v\n", req)
	r, err := hc.Do(req)
	log.Printf("[DEBUG] HTTP Response: %+v\n", r)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	if r.StatusCode == 202 {
		// This is a deferred task.
		tid := TaskID(r.Header.Get("X-Task-Id"))
		log.Printf("[DEBUG] Received Async Task %+v..  will retry...\n", tid)
		// TODO: Sane Configuration for timeouts / retries
		timeout := 5
		waittime := 5 * time.Second
		i := 0
		breakmeout := false
		for i < timeout || breakmeout {
			t, _, err := c.Tasks.Find(tid)
			if err != nil {
				return nil, err
			}
			log.Printf("[DEBUG] Task ID: %+v Retry: %d Status Code: %s\n", tid, i, t.TaskStatusCode)
			switch t.TaskStatusCode {
			case "COMPLETE":
				// Yay
				resp, err := c.Tasks.FindResultByTask(t)
				if err != nil {
					return nil, err
				}
				r = resp
				breakmeout = true
			case "PENDING", "IN_PROCESS":
				i = i + 1
				time.Sleep(waittime)
				continue
			case "ERROR":
				return nil, err
			}
		}
	}

	err = CheckResponse(r)
	if err != nil {
		return r, err
	}

	if v != nil {
		if w, ok := v.(io.Writer); ok {
			io.Copy(w, r.Body)
		} else {
			err = json.NewDecoder(r.Body).Decode(v)
			// err = json.Unmarshal(r.Body, v)
		}
	}

	return r, err
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
