package chef

import (
	"bytes"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

// ChefVersion that we pretend to emulate
const ChefVersion = "11.12.0"

// Body wraps io.Reader and adds methods for calculating hashes and detecting content
type Body struct {
	io.Reader
}

// AuthConfig representing a client and a private key used for encryption
//  This is embedded in the Client type
type AuthConfig struct {
	PrivateKey *rsa.PrivateKey
	ClientName string
}

// Client is vessel for public methods used against the chef-server
type Client struct {
	Auth    *AuthConfig
	BaseURL *url.URL
	client  *http.Client

	ACLs         *ACLService
	Clients      *ApiClientService
	Cookbooks    *CookbookService
	DataBags     *DataBagService
	Environments *EnvironmentService
	Nodes        *NodeService
	Roles        *RoleService
	Sandboxes    *SandboxService
	Search       *SearchService
}

// Config contains the configuration options for a chef client. This is Used primarily in the NewClient() constructor in order to setup a proper client object
type Config struct {
	// This should be the user ID on the chef server
	Name string

	// This is the plain text private Key for the user
	Key string

	// BaseURL is the chef server URL used to connect too. Is using orgs you should include your org in the url
	BaseURL string

	// When set to false (default) this will enable SSL Cert Verification. If you need to disable Cert Verification set to true
	SkipSSL bool

	// Time to wait in seconds before giving up on a request to the server
	Timeout time.Duration
}

/*
An ErrorResponse reports one or more errors caused by an API request.
Thanks to https://github.com/google/go-github
*/
type ErrorResponse struct {
	Response *http.Response // HTTP response that caused this error
}

// Buffer creates a  byte.Buffer copy from a io.Reader resets read on reader to 0,0
func (body *Body) Buffer() *bytes.Buffer {
	var b bytes.Buffer
	if body.Reader == nil {
		return &b
	}

	b.ReadFrom(body.Reader)
	_, err := body.Reader.(io.Seeker).Seek(0, 0)
	if err != nil {
		log.Fatal(err)
	}
	return &b
}

// Hash calculates the body content hash
func (body *Body) Hash() (h string) {
	b := body.Buffer()
	// empty buffs should return a empty string
	if b.Len() == 0 {
		h = HashStr("")
	}
	h = HashStr(b.String())
	return
}

// ContentType returns the content-type string of Body as detected by http.DetectContentType()
func (body *Body) ContentType() string {
	if json.Unmarshal(body.Buffer().Bytes(), &struct{}{}) == nil {
		return "application/json"
	}
	return http.DetectContentType(body.Buffer().Bytes())
}

func (r *ErrorResponse) Error() string {
	return fmt.Sprintf("%v %v: %d",
		r.Response.Request.Method, r.Response.Request.URL,
		r.Response.StatusCode)
}

// NewClient is the client generator used to instantiate a client for talking to a chef-server
// It is a simple constructor for the Client struct intended as a easy interface for issuing
// signed requests
func NewClient(cfg *Config) (*Client, error) {
	pk, err := PrivateKeyFromString([]byte(cfg.Key))
	if err != nil {
		return nil, err
	}

	baseUrl, _ := url.Parse(cfg.BaseURL)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: cfg.SkipSSL},
	}

	c := &Client{
		Auth: &AuthConfig{
			PrivateKey: pk,
			ClientName: cfg.Name,
		},
		client: &http.Client{
			Transport: tr,
			Timeout:   cfg.Timeout * time.Second,
		},
		BaseURL: baseUrl,
	}
	c.ACLs = &ACLService{client: c}
	c.Clients = &ApiClientService{client: c}
	c.Cookbooks = &CookbookService{client: c}
	c.DataBags = &DataBagService{client: c}
	c.Environments = &EnvironmentService{client: c}
	c.Nodes = &NodeService{client: c}
	c.Roles = &RoleService{client: c}
	c.Sandboxes = &SandboxService{client: c}
	c.Search = &SearchService{client: c}
	return c, nil
}

// magicRequestDecoder performs a request on an endpoint, and decodes the response into the passed in Type
func (c *Client) magicRequestDecoder(method, path string, body io.Reader, v interface{}) error {
	req, err := c.NewRequest(method, path, body)
	if err != nil {
		return err
	}

	debug("Request: %+v \n", req)
	res, err := c.Do(req, v)
	if res != nil {
		defer res.Body.Close()
	}
	debug("Response: %+v \n", res)
	if err != nil {
		return err
	}
	return err
}

// NewRequest returns a signed request  suitable for the chef server
func (c *Client) NewRequest(method string, requestUrl string, body io.Reader) (*http.Request, error) {
	relativeUrl, err := url.Parse(requestUrl)
	if err != nil {
		return nil, err
	}
	u := c.BaseURL.ResolveReference(relativeUrl)

	// NewRequest uses a new value object of body
	req, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return nil, err
	}

	// parse and encode Querystring Values
	values := req.URL.Query()
	req.URL.RawQuery = values.Encode()
	debug("Encoded url %+v", u)

	myBody := &Body{body}

	if body != nil {
		// Detect Content-type
		req.Header.Set("Content-Type", myBody.ContentType())
	}

	// Calculate the body hash
	req.Header.Set("X-Ops-Content-Hash", myBody.Hash())

	// don't have to check this works, signRequest only emits error when signing hash is not valid, and we baked that in
	c.Auth.SignRequest(req)
	return req, nil
}

// CheckResponse receives a pointer to a http.Response and generates an Error via unmarshalling
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

// Do is used either internally via our magic request shite or a user may use it
func (c *Client) Do(req *http.Request, v interface{}) (*http.Response, error) {
	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	// BUG(fujin) tightly coupled
	err = CheckResponse(res) // <--
	if err != nil {
		return res, err
	}

	if v != nil {
		if w, ok := v.(io.Writer); ok {
			io.Copy(w, res.Body)
		} else {
			err = json.NewDecoder(res.Body).Decode(v)
			if err != nil {
				return res, err
			}
		}
	}
	return res, nil
}

// SignRequest modifies headers of an http.Request
func (ac AuthConfig) SignRequest(request *http.Request) error {
	// sanitize the path for the chef-server
	// chef-server doesn't support '//' in the Hash Path.
	var endpoint string
	if request.URL.Path != "" {
		endpoint = path.Clean(request.URL.Path)
		request.URL.Path = endpoint
	} else {
		endpoint = request.URL.Path
	}

	request.Header.Set("Method", request.Method)
	request.Header.Set("Hashed Path", HashStr(endpoint))
	request.Header.Set("Accept", "application/json")
	request.Header.Set("X-Chef-Version", ChefVersion)
	request.Header.Set("X-Ops-Timestamp", time.Now().UTC().Format(time.RFC3339))
	request.Header.Set("X-Ops-UserId", ac.ClientName)
	request.Header.Set("X-Ops-Sign", "algorithm=sha1;version=1.0")

	// To validate the signature it seems to be very particular
	var content string
	for _, key := range []string{"Method", "Hashed Path", "X-Ops-Content-Hash", "X-Ops-Timestamp", "X-Ops-UserId"} {
		content += fmt.Sprintf("%s:%s\n", key, request.Header.Get(key))
	}
	content = strings.TrimSuffix(content, "\n")
	// generate signed string of headers
	// Since we've gone through additional validation steps above,
	// we shouldn't get an error at this point
	signature, err := GenerateSignature(ac.PrivateKey, content)
	if err != nil {
		return err
	}

	// TODO: THIS IS CHEF PROTOCOL SPECIFIC
	// Signature is made up of n 60 length chunks
	base64sig := Base64BlockEncode(signature, 60)

	// roll over the auth slice and add the apropriate header
	for index, value := range base64sig {
		request.Header.Set(fmt.Sprintf("X-Ops-Authorization-%d", index+1), string(value))
	}

	return nil
}

// PrivateKeyFromString parses an RSA private key from a string
func PrivateKeyFromString(key []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(key)
	if block == nil {
		return nil, fmt.Errorf("block size invalid for '%s'", string(key))
	}
	rsaKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return rsaKey, nil
}
