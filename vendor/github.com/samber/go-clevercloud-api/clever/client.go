package clever

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"

	"crypto/tls"
	"net/http"
	"net/url"

	"math/rand"
	"time"
)

// Get access token and secret key here: https://console.clever-cloud.com/cli-oauth
const (
	OAUTH_CONSUMER_KEY    = "T5nFjKeHH4AIlEveuGhB5S3xg8T19e"
	OAUTH_CONSUMER_SECRET = "MgVMqTr6fWlf2M0tkC2MXOnhfqBWDT"
)

type ClientConfig struct {
	Endpoint   string
	OrgId      string
	AuthToken  string
	AuthSecret string
}

type Client struct {
	httpClient *http.Client
	config     *ClientConfig
}

func init() {
	rand.Seed(42)
}

func NewClient(config *ClientConfig) (*Client, error) {
	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
	}
	t := &http.Transport{
		TLSClientConfig: tlsConf,
	}
	httpClient := &http.Client{
		Transport: t,
	}
	if config.Endpoint == "" {
		config.Endpoint = "https://api.clever-cloud.com/v2/"
	}

	client := &Client{
		httpClient: httpClient,
		config:     config,
	}

	if err := client.loadAddons(); err != nil {
		return nil, err
	}
	if err := client.loadApplicationInstances(); err != nil {
		return nil, err
	}

	return client, nil
}

func (c *Client) rawRequest(method string, path string, body []byte, headers map[string]string) ([]byte, error) {
	var err error

	req := &http.Request{
		Method: method,
		Header: http.Header{},
	}

	req.URL, err = url.Parse(path)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Body = ioutil.NopCloser(bytes.NewReader(body))
		req.ContentLength = int64(len(body))
	}

	req.Header.Add("User-Agent", "Go-CleverCloud-API")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-LIB", "github.com/samber/go-clevercloud-api")
	req.Header.Add("Authorization", "OAuth realm=\"https://api.clever-cloud.com/v2/oauth\", oauth_consumer_key=\""+OAUTH_CONSUMER_KEY+"\", oauth_token=\""+c.config.AuthToken+"\", oauth_signature_method=\"PLAINTEXT\", oauth_signature=\""+OAUTH_CONSUMER_SECRET+"&"+c.config.AuthSecret+"\", oauth_timestamp=\""+strconv.Itoa(int(time.Now().Unix()))+"\", oauth_nonce=\""+strconv.Itoa(rand.Intn(1000000))+"\"") // should be less dirty and signed with HMAC signature !!
	for k, v := range headers {
		req.Header.Add(k, v)
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode == 400 {
		return nil, &BadRequestError{body: string(resBody)}
	}
	if res.StatusCode == 403 {
		return nil, &ForbiddenError{body: string(resBody)}
	}
	if res.StatusCode == 404 {
		return nil, &NotFoundError{body: string(resBody)}
	}
	if res.StatusCode >= 500 {
		return nil, &InternalServerError{body: string(resBody)}
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP Error %i\n%s", res.StatusCode, resBody)
	}

	return resBody, nil
}

func (c *Client) jsonRequest(method string, path string, reqBody interface{}, resBody interface{}) error {
	var err error
	var reqBodyBytes []byte
	if reqBody != nil {
		if reqBodyBytes, err = json.Marshal(reqBody); err != nil {
			return err
		}
	}

	reqUrl := c.config.Endpoint + path

	resBodyBytes, err := c.rawRequest(method, reqUrl, reqBodyBytes, map[string]string{})
	if err != nil {
		return err
	}

	// reqBody can be nil to ignore the response
	if resBody != nil {
		if resBodyBytes == nil {
			return fmt.Errorf("Server did not return an JSON document: %s.", resBodyBytes)
		}
		err = json.Unmarshal(resBodyBytes, resBody)
		if err != nil {
			return fmt.Errorf("Error decoding response JSON document: %s.", err.Error())
		}
	}

	return nil
}

func (c *Client) get(path string, resBody interface{}) error {
	return c.jsonRequest("GET", path, nil, resBody)
}
func (c *Client) post(path string, reqBody interface{}, resBody interface{}) error {
	return c.jsonRequest("POST", path, reqBody, resBody)
}
func (c *Client) put(path string, reqBody interface{}, resBody interface{}) error {
	return c.jsonRequest("PUT", path, reqBody, resBody)
}
func (c *Client) delete(path string) error {
	return c.jsonRequest("DELETE", path, nil, nil)
}
