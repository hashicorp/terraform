package github

import (
	"net/http"
	"net/url"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

type Config struct {
	Token        string
	Organization string
	BaseURL      string
}

type Organization struct {
	name    string
	Token   string
	BaseURL *url.URL
}

type Client struct {
	*github.Client

	Transport *conditionalTransport
}

type conditionalTransport struct {
	Base *oauth2.Transport

	etag         string
	LastModified string
}

func (c *conditionalTransport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	if c.LastModified != "" {
		req.Header.Set("If-Modified-Since", c.LastModified)
	} else {
		// fallback to using etag if we don't have a LastModified value
		req.Header.Set("If-None-Match", c.etag)
	}

	return c.Base.RoundTrip(req)
}

// Create and return an Organization
func (c *Config) NewOrganization() (interface{}, error) {
	var org Organization
	org.name = c.Organization
	org.Token = c.Token

	if c.BaseURL != "" {
		u, err := url.Parse(c.BaseURL)
		if err != nil {
			return nil, err
		}
		org.BaseURL = u
	}
	return &org, nil
}

// Create and return a new github client.
func (o *Organization) Client() *Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: o.Token},
	)
	tr := &oauth2.Transport{Source: ts}
	transport := &conditionalTransport{Base: tr}
	tc := &http.Client{
		Transport: transport,
	}

	client := Client{
		Client:    github.NewClient(tc),
		Transport: transport,
	}

	if o.BaseURL != nil {
		client.BaseURL = o.BaseURL
	}

	return &client
}
