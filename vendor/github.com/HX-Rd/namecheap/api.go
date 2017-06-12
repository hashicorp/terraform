package namecheap

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/hashicorp/go-cleanhttp"
)

// Client provides a client to the Namecheap API
type Client struct {
	// Access Token
	Token string

	// ApiUser
	ApiUser string

	// Username
	Username string

	// URL to the DO API to use
	URL string

	// IP that is whitelisted
	Ip string

	// HttpClient is the client to use. A client with
	// default values will be used if not provided.
	Http *http.Client
}

// NewClient returns a new dnsimple client,
// requires an authorization token. You can generate
// an OAuth token by visiting the Apps & API section
// of the Namecheap control panel for your account.
func NewClient(username string, apiuser string, token string, ip string, useSandbox bool) (*Client, error) {
	url := "https://api.namecheap.com/xml.response"
	if useSandbox {
		url = "https://api.sandbox.namecheap.com/xml.response"
	}
	client := Client{
		Token:    token,
		ApiUser:  apiuser,
		Username: username,
		Ip:       ip,
		URL:      url,
		Http:     cleanhttp.DefaultClient(),
	}
	return &client, nil
}

// Creates a new request with the params
func (c *Client) NewRequest(body map[string]string) (*http.Request, error) {
	u, err := url.Parse(c.URL)

	if err != nil {
		return nil, fmt.Errorf("Error parsing base URL: %s", err)
	}

	body["Username"] = c.Username
	body["ApiKey"] = c.Token
	body["ApiUser"] = c.ApiUser
	body["ClientIp"] = c.Ip

	rBody := encodeBody(body)

	fmt.Printf("The body: %s\n", rBody)

	if err != nil {
		return nil, fmt.Errorf("Error encoding request body: %s", err)
	}

	// Build the request
	req, err := http.NewRequest("POST", u.String(), bytes.NewBufferString(rBody))
	if err != nil {
		return nil, fmt.Errorf("Error creating request: %s", err)
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(rBody)))

	return req, nil

}

func decode(reader io.Reader, obj interface{}) error {
	decoder := xml.NewDecoder(reader)
	err := decoder.Decode(&obj)
	if err != nil {
		return err
	}
	return nil
}

func encodeBody(body map[string]string) string {
	data := url.Values{}
	for key, val := range body {
		data.Set(key, val)
	}
	return data.Encode()
}
