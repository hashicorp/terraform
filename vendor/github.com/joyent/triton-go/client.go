package triton

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/hashicorp/errwrap"
	"github.com/joyent/triton-go/authentication"
)

const nilContext = "nil context"

// Client represents a connection to the Triton API.
type Client struct {
	client      *http.Client
	authorizer  []authentication.Signer
	apiURL      url.URL
	accountName string
}

// NewClient is used to construct a Client in order to make API
// requests to the Triton API.
//
// At least one signer must be provided - example signers include
// authentication.PrivateKeySigner and authentication.SSHAgentSigner.
func NewClient(endpoint string, accountName string, signers ...authentication.Signer) (*Client, error) {
	apiURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, errwrap.Wrapf("invalid endpoint: {{err}}", err)
	}

	if accountName == "" {
		return nil, errors.New("account name can not be empty")
	}

	httpClient := &http.Client{
		Transport:     httpTransport(false),
		CheckRedirect: doNotFollowRedirects,
	}

	return &Client{
		client:      httpClient,
		authorizer:  signers,
		apiURL:      *apiURL,
		accountName: accountName,
	}, nil
}

// InsecureSkipTLSVerify turns off TLS verification for the client connection. This
// allows connection to an endpoint with a certificate which was signed by a non-
// trusted CA, such as self-signed certificates. This can be useful when connecting
// to temporary Triton installations such as Triton Cloud-On-A-Laptop.
func (c *Client) InsecureSkipTLSVerify() {
	if c.client == nil {
		return
	}

	c.client.Transport = httpTransport(true)
}

func httpTransport(insecureSkipTLSVerify bool) *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 10 * time.Second,
		DisableKeepAlives:   true,
		MaxIdleConnsPerHost: -1,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: insecureSkipTLSVerify,
		},
	}
}

func doNotFollowRedirects(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}

func (c *Client) executeRequestURIParams(ctx context.Context, method, path string, body interface{}, query *url.Values) (io.ReadCloser, error) {
	var requestBody io.ReadSeeker
	if body != nil {
		marshaled, err := json.MarshalIndent(body, "", "    ")
		if err != nil {
			return nil, err
		}
		requestBody = bytes.NewReader(marshaled)
	}

	endpoint := c.apiURL
	endpoint.Path = path
	if query != nil {
		endpoint.RawQuery = query.Encode()
	}

	req, err := http.NewRequest(method, endpoint.String(), requestBody)
	if err != nil {
		return nil, errwrap.Wrapf("Error constructing HTTP request: {{err}}", err)
	}

	dateHeader := time.Now().UTC().Format(time.RFC1123)
	req.Header.Set("date", dateHeader)

	authHeader, err := c.authorizer[0].Sign(dateHeader)
	if err != nil {
		return nil, errwrap.Wrapf("Error signing HTTP request: {{err}}", err)
	}
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Version", "8")
	req.Header.Set("User-Agent", "triton-go Client API")

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req.WithContext(ctx))
	if err != nil {
		return nil, errwrap.Wrapf("Error executing HTTP request: {{err}}", err)
	}

	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		return resp.Body, nil
	}

	return nil, c.decodeError(resp.StatusCode, resp.Body)
}

func (c *Client) decodeError(statusCode int, body io.Reader) error {
	err := &TritonError{
		StatusCode: statusCode,
	}

	errorDecoder := json.NewDecoder(body)
	if err := errorDecoder.Decode(err); err != nil {
		return errwrap.Wrapf("Error decoding error response: {{err}}", err)
	}

	return err
}

func (c *Client) executeRequest(ctx context.Context, method, path string, body interface{}) (io.ReadCloser, error) {
	return c.executeRequestURIParams(ctx, method, path, body, nil)
}

func (c *Client) executeRequestRaw(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var requestBody io.ReadSeeker
	if body != nil {
		marshaled, err := json.MarshalIndent(body, "", "    ")
		if err != nil {
			return nil, err
		}
		requestBody = bytes.NewReader(marshaled)
	}

	endpoint := c.apiURL
	endpoint.Path = path

	req, err := http.NewRequest(method, endpoint.String(), requestBody)
	if err != nil {
		return nil, errwrap.Wrapf("Error constructing HTTP request: {{err}}", err)
	}

	dateHeader := time.Now().UTC().Format(time.RFC1123)
	req.Header.Set("date", dateHeader)

	authHeader, err := c.authorizer[0].Sign(dateHeader)
	if err != nil {
		return nil, errwrap.Wrapf("Error signing HTTP request: {{err}}", err)
	}
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Version", "8")
	req.Header.Set("User-Agent", "triton-go c API")

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req.WithContext(ctx))
	if err != nil {
		return nil, errwrap.Wrapf("Error executing HTTP request: {{err}}", err)
	}

	return resp, nil
}
