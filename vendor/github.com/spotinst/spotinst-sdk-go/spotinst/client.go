package spotinst

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

// DefaultTransport returns a new http.Transport with the same default values
// as http.DefaultTransport, but with idle connections and KeepAlives disabled.
func defaultTransport() *http.Transport {
	transport := defaultPooledTransport()
	transport.DisableKeepAlives = true
	transport.MaxIdleConnsPerHost = -1
	return transport
}

// DefaultPooledTransport returns a new http.Transport with similar default
// values to http.DefaultTransport. Do not use this for transient transports as
// it can leak file descriptors over time. Only use this for transports that
// will be re-used for the same host(s).
func defaultPooledTransport() *http.Transport {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 10 * time.Second,
		DisableKeepAlives:   false,
		MaxIdleConnsPerHost: 1,
	}
	return transport
}

// defaultHttpClient returns a new http.Client with similar default values to
// http.Client, but with a non-shared Transport, idle connections disabled, and
// KeepAlives disabled.
func defaultHttpClient() *http.Client {
	return &http.Client{
		Transport: defaultTransport(),
	}
}

// defaultHttpPooledClient returns a new http.Client with the same default values
// as http.Client, but with a shared Transport. Do not use this function
// for transient clients as it can leak file descriptors over time. Only use
// this for clients that will be re-used for the same host(s).
func defaultHttpPooledClient() *http.Client {
	return &http.Client{
		Transport: defaultPooledTransport(),
	}
}

// DefaultConfig returns a default configuration for the client. By default this
// will pool and reuse idle connections to API. If you have a long-lived
// client object, this is the desired behavior and should make the most efficient
// use of the connections to API. If you don't reuse a client object , which
// is not recommended, then you may notice idle connections building up over
// time. To avoid this, use the DefaultNonPooledConfig() instead.
func defaultPooledConfig() *clientConfig {
	return defaultConfig(defaultPooledTransport)
}

// DefaultNonPooledConfig returns a default configuration for the client which
// does not pool connections. This isn't a recommended configuration because it
// will reconnect to API on every request, but this is useful to avoid the
// accumulation of idle connections if you make many client objects during the
// lifetime of your application.
func defaultNonPooledConfig() *clientConfig {
	return defaultConfig(defaultTransport)
}

// defaultConfig returns the default configuration for the client, using the
// given function to make the transport.
func defaultConfig(transportFn func() *http.Transport) *clientConfig {
	config := &clientConfig{
		apiAddress:   DefaultAPIAddress,
		oauthAddress: DefaultOAuthAddress,
		scheme:       DefaultScheme,
		userAgent:    DefaultUserAgent,
		contentType:  DefaultContentType,
		httpClient: &http.Client{
			Transport: transportFn(),
		},
	}

	return config
}

// Client provides a client to the API
type Client struct {
	config              *clientConfig
	AwsGroupService     AwsGroupService
	HealthCheckService  HealthCheckService
	SubscriptionService SubscriptionService
}

// NewClient returns a new client
func NewClient(opts ...ClientOptionFunc) (*Client, error) {
	config := defaultPooledConfig()

	for _, o := range opts {
		o(config)
	}

	client := &Client{config: config}
	client.AwsGroupService = &AwsGroupServiceOp{client}
	client.HealthCheckService = &HealthCheckServiceOp{client}
	client.SubscriptionService = &SubscriptionServiceOp{client}

	// Should we request a new access token/refresh token pair?
	if creds := config.credentials; creds != nil && creds.Token == "" {
		accessToken, _, err := client.obtainOAuthTokens(
			creds.Email,
			creds.Password,
			creds.ClientID,
			creds.ClientSecret,
		)
		if err != nil {
			return nil, err
		}
		config.credentials.Token = accessToken
	}

	return client, nil
}

type oauthTokenPair struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

// obtainOAuthTokens obtains a new access token/refresh token pair.
func (c *Client) obtainOAuthTokens(username, password, clientID, clientSecret string) (string, string, error) {
	u := url.URL{
		Scheme: c.config.scheme,
		Host:   c.config.oauthAddress,
		Path:   "/token",
	}
	res, err := http.PostForm(u.String(), url.Values{
		"grant_type":    {"password"},
		"username":      {username},
		"password":      {password},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
	})
	_, resp, err := requireOK(0, res, err)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	pair, err := oauthTokensFromHttpResponse(resp)
	if err != nil {
		return "", "", err
	}
	return pair.AccessToken, pair.RefreshToken, nil
}

func oauthTokensFromJSON(in []byte) (*oauthTokenPair, error) {
	var rw responseWrapper
	if err := json.Unmarshal(in, &rw); err != nil {
		return nil, err
	}
	if len(rw.Response.Items) == 0 {
		return nil, fmt.Errorf("invalid or malformed response")
	}
	out := new(oauthTokenPair)
	for _, i := range rw.Response.Items {
		if err := json.Unmarshal(i, out); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func oauthTokensFromHttpResponse(resp *http.Response) (*oauthTokenPair, error) {
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return oauthTokensFromJSON(body)
}

// newRequest is used to create a new request
func (c *Client) newRequest(method, path string) *request {
	r := &request{
		config: c.config,
		method: method,
		url: &url.URL{
			Scheme: c.config.scheme,
			Host:   c.config.apiAddress,
			Path:   path,
		},
		params: make(map[string][]string),
		header: make(http.Header),
	}
	if creds := c.config.credentials; creds != nil {
		if token := creds.Token; token != "" {
			r.header.Set("Authorization", "Bearer "+token)
		}
	}
	return r
}

// doRequest runs a request with our client
func (c *Client) doRequest(r *request) (time.Duration, *http.Response, error) {
	req, err := r.toHTTP()
	if err != nil {
		return 0, nil, err
	}
	c.dumpRequest(req)
	start := time.Now()
	resp, err := c.config.httpClient.Do(req)
	diff := time.Now().Sub(start)
	c.dumpResponse(resp)
	return diff, resp, err
}

// errorf logs to the error log.
func (c *Client) errorf(format string, args ...interface{}) {
	if c.config.errorlog != nil {
		c.config.errorlog.Printf(format, args...)
	}
}

// infof logs informational messages.
func (c *Client) infof(format string, args ...interface{}) {
	if c.config.infolog != nil {
		c.config.infolog.Printf(format, args...)
	}
}

// tracef logs to the trace log.
func (c *Client) tracef(format string, args ...interface{}) {
	if c.config.tracelog != nil {
		c.config.tracelog.Printf(format, args...)
	}
}

// dumpRequest dumps the given HTTP request to the trace log.
func (c *Client) dumpRequest(r *http.Request) {
	if c.config.tracelog != nil && r != nil {
		out, err := httputil.DumpRequestOut(r, true)
		if err == nil {
			c.tracef("%s\n", string(out))
		}
	}
}

// dumpResponse dumps the given HTTP response to the trace log.
func (c *Client) dumpResponse(resp *http.Response) {
	if c.config.tracelog != nil && resp != nil {
		out, err := httputil.DumpResponse(resp, true)
		if err == nil {
			c.tracef("%s\n", string(out))
		}
	}
}
