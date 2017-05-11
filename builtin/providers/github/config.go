package github

import (
	"net/url"

	"github.com/google/go-github/github"
	"github.com/hashicorp/terraform/helper/logging"
	"golang.org/x/oauth2"
)

type Config struct {
	Token        string
	Organization string
	BaseURL      string
}

type Organization struct {
	name   string
	client *github.Client
}

// Client configures and returns a fully initialized GithubClient
func (c *Config) Client() (interface{}, error) {
	var org Organization
	org.name = c.Organization
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: c.Token},
	)
	tc := oauth2.NewClient(oauth2.NoContext, ts)

	tc.Transport = logging.NewTransport("Github", tc.Transport)

	org.client = github.NewClient(tc)
	if c.BaseURL != "" {
		u, err := url.Parse(c.BaseURL)
		if err != nil {
			return nil, err
		}
		org.client.BaseURL = u
	}

	return &org, nil
}
