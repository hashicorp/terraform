package artifactory

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// Request represents an artifactory http request
type Request struct {
	Verb        string
	Path        string
	ContentType string
	Accept      string
	QueryParams map[string]string
	Body        io.Reader
}

// HTTPRequest performs an HTTP request to artifactory
func (c *Client) HTTPRequest(ar Request) ([]byte, error) {
	options := make(map[string]string)
	if ar.ContentType != "" {
		options["content-type"] = ar.ContentType
	}
	for q, p := range ar.QueryParams {
		options[q] = p
	}
	r, err := c.makeRequest(ar.Verb, ar.Path, options, ar.Body)
	if err != nil {
		var data bytes.Buffer
		return data.Bytes(), err
	}

	return c.parseResponse(r)
}

// HTTPRequestWithResponse performs an HTTP request to artifactory and returns
// the http response
func (c *Client) HTTPRequestWithResponse(ar Request) (*http.Response, error) {
	options := make(map[string]string)
	if ar.ContentType != "" {
		options["content-type"] = ar.ContentType
	}
	for q, p := range ar.QueryParams {
		options[q] = p
	}

	return c.makeRequest(ar.Verb, ar.Path, options, ar.Body)
}

// Get performs an http GET to artifactory
func (c *Client) Get(path string, options map[string]string) ([]byte, error) {
	r, err := c.makeRequest("GET", path, options, nil)
	if err != nil {
		var data bytes.Buffer
		return data.Bytes(), err
	}

	return c.parseResponse(r)
}

// Post performs an http POST to artifactory
func (c *Client) Post(path string, data []byte, options map[string]string) ([]byte, error) {
	body := bytes.NewReader(data)
	r, err := c.makeRequest("POST", path, options, body)
	if err != nil {
		var data bytes.Buffer
		return data.Bytes(), err
	}

	return c.parseResponse(r)
}

// Put performs an http PUT to artifactory
func (c *Client) Put(path string, data []byte, options map[string]string) ([]byte, error) {
	body := bytes.NewReader(data)
	r, err := c.makeRequest("PUT", path, options, body)
	if err != nil {
		var data bytes.Buffer
		return data.Bytes(), err
	}

	return c.parseResponse(r)
}

// Delete performs an http DELETE to artifactory
func (c *Client) Delete(path string) error {
	r, err := c.makeRequest("DELETE", path, make(map[string]string), nil)
	if err != nil {
		return err
	}

	_, err = c.parseResponse(r)

	return err
}

func (c *Client) makeRequest(method string, path string, options map[string]string, body io.Reader) (*http.Response, error) {
	qs := url.Values{}
	var contentType string
	for q, p := range options {
		if q == "content-type" {
			contentType = p
			delete(options, q)
		} else {
			qs.Add(q, p)
		}
	}

	baseReqPath := strings.TrimSuffix(c.Config.BaseURL, "/") + path
	if os.Getenv("ARTIFACTORY_DEBUG") != "" {
		log.Printf("Final URL: %s", baseReqPath)
	}
	u, err := url.Parse(baseReqPath)
	if err != nil {
		return nil, err
	}
	if len(options) != 0 {
		u.RawQuery = qs.Encode()
	}
	buf := new(bytes.Buffer)
	if body != nil {
		_, _ = buf.ReadFrom(body)
	}
	req, _ := http.NewRequest(method, u.String(), bytes.NewReader(buf.Bytes()))
	if body != nil {
		h := sha1.New()
		_, _ = h.Write(buf.Bytes())
		chkSum := h.Sum(nil)
		req.Header.Add("X-Checksum-Sha1", fmt.Sprintf("%x", chkSum))
	}
	req.Header.Add("user-agent", "artifactory-go."+Version.String())
	req.Header.Add("X-Result-Detail", "info, properties")
	if contentType != "" {
		req.Header.Add("Content-Type", contentType)
	} else {
		req.Header.Add("Content-Type", "application/json")
	}
	if c.Config.AuthMethod == "basic" {
		req.SetBasicAuth(c.Config.Username, c.Config.Password)
	} else {
		req.Header.Add("X-JFrog-Art-Api", c.Config.Token)
	}
	if os.Getenv("ARTIFACTORY_DEBUG") != "" {
		log.Printf("Headers: %#v", req.Header)
		if len(buf.Bytes()) > 0 {
			log.Printf("Body: %#v", buf.String())
		}
	}

	r, err := c.Client.Do(req)

	return r, err
}

func (c *Client) parseResponse(r *http.Response) ([]byte, error) {
	defer func() { _ = r.Body.Close() }()
	data, err := ioutil.ReadAll(r.Body)
	if r.StatusCode < 200 || r.StatusCode > 299 {
		var ej ErrorsJSON
		uerr := json.Unmarshal(data, &ej)
		if uerr != nil {
			emsg := fmt.Sprintf("Unable to parse error json. Non-2xx code returned: %d. Message follows:\n%s", r.StatusCode, string(data))
			return data, errors.New(emsg)
		}
		// here we catch the {"error":"foo"} oddity in things like security/apiKey
		if ej.Error != "" {
			return data, errors.New(ej.Error)
		}
		var emsgs []string
		for _, i := range ej.Errors {
			emsgs = append(emsgs, i.Message)
		}
		return data, errors.New(strings.Join(emsgs, "\n"))
	}
	return data, err
}
