package udnssdk

// udnssdk - a golang sdk for the ultradns REST service.
// based heavily on github.com/weppos/dnsimple
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
	libraryVersion     = "0.1"
	DefaultTestBaseURL = "https://test-restapi.ultradns.com/"
	DefaultLiveBaseURL = "https://restapi.ultradns.com/"

	userAgent = "udnssdk-go/" + libraryVersion

	apiVersion = "v1"
)

type Client struct {
	// This is our client structure.
	HttpClient *http.Client

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

	// UltraDNS has 'zones' and 'rrsets'.  We really only care about RR Sets for
	// this implementation.
	RRSets *RRSetsService
	// UltraDNS Tasks API
	Tasks *TasksService
}

// NewClient returns a new ultradns API client.
func NewClient(username, password, BaseURL string) (*Client, error) {
	accesstoken, refreshtoken, err := GetAuthTokens(username, password, BaseURL)
	if err != nil {
		return nil, err
	}
	c := &Client{AccessToken: accesstoken, RefreshToken: refreshtoken, Username: username, Password: password, HttpClient: &http.Client{}, BaseURL: BaseURL, UserAgent: userAgent}
	c.RRSets = &RRSetsService{client: c}
	c.Tasks = &TasksService{client: c}
	return c, nil
}

// NewAuthRequest creates an Authorization request to get an access and refresh token.
// {"tokenType":"Bearer","refreshToken":"48472efcdce044c8850ee6a395c74a7872932c7112","accessToken":"b91d037c75934fc89a9f43fe4a","expiresIn":"3600"
// ,"expires_in":"3600","token_type":"Bearer","refresh_token":"48472efcdce044c8850ee6a395c74a7872932c7112","access_token":"b91d037c75934fc89a9f43fe4a"}

type AuthResponse struct {
	TokenType     string `json:"tokenType"`
	RefreshToken  string `json:"refreshToken"`
	AccessToken   string `json:"accessToken"`
	ExpiresIn     string `json:"expiresIn"`
	Expires_in    string `json:"expires_in"`
	Token_type    string `json:"token_type"`
	Refresh_token string `json:"refresh_token"`
	Access_token  string `json:"access_token"`
}

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
	// BUG: Looking for an intermittant edge case causing a JSON error
	fmt.Printf("ResCode: %d Body: %s\n", res.StatusCode, body)

	err = CheckAuthResponse(res, body)
	if err != nil {
		return "", "", err
	}

	var authr AuthResponse
	//log.Printf("GetAuthTokens: %s", string(body))
	err = json.Unmarshal(body, &authr)
	if err != nil {
		return string(body), "JSON Decode Error", err
	}
	//log.Printf("Expires: %v Access T: %v Refresh T: %v\n", authr.Expires_in, authr.Access_token, authr.Refresh_token)
	//log.Printf("%+v", authr)
	return authr.Access_token, authr.Refresh_token, err
}

// NewRequest creates an API request.
// The path is expected to be a relative path and will be resolved
// according to the BaseURL of the Client. Paths should always be specified without a preceding slash.
func (client *Client) NewRequest(method, path string, payload interface{}) (*http.Request, error) {
	url := client.BaseURL + fmt.Sprintf("%s/%s", apiVersion, path)

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
	req.Header.Add("User-Agent", client.UserAgent)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", client.AccessToken))
	req.Header.Add("Token", fmt.Sprintf("Bearer %s", client.AccessToken))

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
	//log.Printf("Req: %+v\n", req)
	res, err := c.HttpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	origresponse := &Response{Response: res}

	//response := &Response{Response: res}
	//log.Printf("ReS: %+v\n", res)
	var nres *http.Response
	nres = res
	if res.StatusCode == 202 {
		// This is a deferred task.
		mytaskid := res.Header.Get("X-Task-Id")
		log.Printf("Received Async Task %+v..  will retry...\n", mytaskid)
		timeout := 5
		waittime := 5 * time.Second
		i := 0
		breakmeout := false
		for i < timeout || breakmeout {
			myt, statusres, err := c.Tasks.GetTaskStatus(mytaskid)
			if err != nil {
				return origresponse, err
			}
			log.Printf("ID %+v Retry %d Status Code %s\n", mytaskid, i, myt.TaskStatusCode)
			switch myt.TaskStatusCode {
			case "COMPLETE":
				// Yay
				tres, err := c.Tasks.GetTaskResultByURI(myt.ResultUri)
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

// An ErrorResponse represents an error caused by an API request.
// Example:
// {"errorCode":60001,"errorMessage":"invalid_grant:Invalid username & password combination.","error":"invalid_grant","error_description":"60001: invalid_grant:Invalid username & password combination."}

type ErrorResponse struct {
	Response         *http.Response // HTTP response that caused this error
	ErrorCode        int            `json:"errorCode"`    //  error code
	ErrorMessage     string         `json:"errorMessage"` // human-readable message
	ErrorStr         string         `json:"error"`
	ErrorDescription string         `json:"error_description"`
}
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

	//var er ErrorResponseList
	var er ErrorResponse
	//fmt.Printf("Body: %s\n", body)

	err := json.Unmarshal(body, &er)
	//err = json.NewDecoder(r.Body).Decode(errorResponse)
	if err != nil {
		fmt.Printf("ERROR: %+v - Body: %s", err, body)
		return err
	}
	er.Response = r
	//log.Printf("CheckAuthResponse: %d", er)

	return er

}

// CheckResponse checks the API response for errors, and returns them if present.
// A response is considered an error if the status code is different than 2xx. Specific requests
// may have additional requirements, but this is sufficient in most of the cases.

func CheckResponse(r *http.Response) error {

	if code := r.StatusCode; 200 <= code && code <= 299 {
		return nil
	}

	//errorResponse := &ErrorResponseList{Response: r}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	//var er ErrorResponseList
	var er []ErrorResponse
	err = json.Unmarshal(body, &er)
	//err = json.NewDecoder(r.Body).Decode(errorResponse)
	if err != nil {
		return err
	}
	//log.Printf("CheckResponse: %+v", er)
	x := &ErrorResponseList{Response: r, Responses: er}
	return x
}
