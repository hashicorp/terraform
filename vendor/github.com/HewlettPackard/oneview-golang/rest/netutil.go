package rest

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/HewlettPackard/oneview-golang/utils"
	"github.com/docker/machine/libmachine/log"
)

var (
	codes = map[int]bool{
		http.StatusOK:                  true,
		http.StatusCreated:             true,
		http.StatusAccepted:            true,
		http.StatusNoContent:           true,
		http.StatusBadRequest:          false,
		http.StatusNotFound:            false,
		http.StatusNotAcceptable:       false,
		http.StatusConflict:            false,
		http.StatusInternalServerError: false,
	}

	// TODO: this should have a real cert
	tr = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	// get a client
	client = &http.Client{Transport: tr}
)

// Options for REST call
type Options struct {
	Headers map[string]string
	Query   map[string]interface{}
}

// Client - generic REST api client
type Client struct {
	Method
	User       string
	Password   string
	Domain     string
	APIKey     string
	APIVersion int
	SSLVerify  bool
	Endpoint   string
	Option     Options
}

// NewClient - get a new network client
func (c *Client) NewClient(user, key, endpoint string) *Client {
	return &Client{User: user, APIKey: key, Endpoint: endpoint, Option: Options{}}
}

// isOkStatus - check the return status of the response
func (c *Client) isOkStatus(code int) bool {
	return codes[code]
}

// SetQueryString - set the query strings to use
func (c *Client) SetQueryString(query map[string]interface{}) {
	// TODO: uuencode the query String
	c.Option.Query = query
}

// GetQueryString - get a query string for url
func (c *Client) GetQueryString(u *url.URL) {
	if len(c.Option.Query) == 0 {
		return
	}
	parameters := url.Values{}
	for k, v := range c.Option.Query {
		if val, ok := v.([]string); ok {
			for _, va := range val {
				parameters.Add(k, va)
			}
		} else {
			parameters.Add(k, v.(string))
		}
		u.RawQuery = parameters.Encode()
	}
	return
}

// SetAuthHeaderOptins - set the Headers Options
func (c *Client) SetAuthHeaderOptions(headers map[string]string) {
	c.Option.Headers = headers
}

// RestAPICall - general rest method caller
func (c *Client) RestAPICall(method Method, path string, options interface{}) ([]byte, error) {
	log.Debugf("RestAPICall %s - %s%s", method, utils.Sanatize(c.Endpoint), path)

	var (
		Url *url.URL
		err error
		req *http.Request
	)

	Url, err = url.Parse(utils.Sanatize(c.Endpoint))
	if err != nil {
		return nil, err
	}
	Url.Path += path

	// Manage the query string
	c.GetQueryString(Url)

	log.Debugf("*** url => %s", Url.String())
	log.Debugf("*** method => %s", method.String())

	// parse url
	reqUrl, err := url.Parse(Url.String())
	if err != nil {
		return nil, fmt.Errorf("Error with request: %v - %q", Url, err)
	}

	// handle options
	if options != nil {
		OptionsJSON, err := json.Marshal(options)
		if err != nil {
			return nil, err
		}
		log.Debugf("*** options => %+v", bytes.NewBuffer(OptionsJSON))
		req, err = http.NewRequest(method.String(), reqUrl.String(), bytes.NewBuffer(OptionsJSON))
	} else {
		req, err = http.NewRequest(method.String(), reqUrl.String(), nil)
	}

	if err != nil {
		return nil, fmt.Errorf("Error with request: %v - %q", Url, err)
	}

	// setup proxy
	proxyUrl, err := http.ProxyFromEnvironment(req)
	if err != nil {
		return nil, fmt.Errorf("Error with proxy: %v - %q", proxyUrl, err)
	}
	if proxyUrl != nil {
		tr.Proxy = http.ProxyURL(proxyUrl)
		log.Debugf("*** proxy => %+v", tr.Proxy)
	}

	// build the auth headerU
	for k, v := range c.Option.Headers {
		log.Debugf("Headers -> %s -> %+v\n", k, v)
		req.Header.Add(k, v)
	}

	// req.SetBasicAuth(c.User, c.APIKey)
	req.Method = fmt.Sprintf("%s", method.String())

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// TODO: CLeanup Later
	// DEBUGGING WHILE WE WORK
	// DEBUGGING WHILE WE WORK
	// fmt.Printf("METHOD --> %+v\n",method)
	log.Debugf("REQ    --> %+v\n", req)
	log.Debugf("RESP   --> %+v\n", resp)
	log.Debugf("ERROR  --> %+v\n", err)
	// DEBUGGING WHILE WE WORK

	data, err := ioutil.ReadAll(resp.Body)

	if !c.isOkStatus(resp.StatusCode) {
		type apiErr struct {
			Err string `json:"details"`
		}
		var outErr apiErr
		json.Unmarshal(data, &outErr)
		return nil, fmt.Errorf("Error in response: %s\n Response Status: %s", outErr.Err, resp.Status)
	}

	if err != nil {
		return nil, err
	}

	return data, nil
}
