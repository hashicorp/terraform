package github

import (
	"net/url"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

type Config struct {
	// will be deprecated, instead of use OrganizationKey
	Token string

	Organization string
	BaseURL      string

	// UserKey & OrganizationKey will be used for multiple operation
	// we will be using both user and organization key if needed,
	// e.g:
	//   to add a user into a team of organization, user must activate this to join team,
	// with this user key, provider is going to do this for user.
	UserKey         string
	OrganizationKey string
}

type Clients struct {
	OrgName    string
	OrgClient  *github.Client
	UserClient *github.Client
}

// Client configures and returns a fully initialized GithubClient
func (c *Config) Clients() (*Clients, error) {
	if c.Token != "" {
		c.OrganizationKey = c.Token
	}

	orgClient := github.NewClient(
		oauth2.NewClient(
			oauth2.NoContext,
			oauth2.StaticTokenSource(
				&oauth2.Token{
					AccessToken: c.OrganizationKey,
				},
			),
		),
	)
	userClient := github.NewClient(
		oauth2.NewClient(
			oauth2.NoContext,
			oauth2.StaticTokenSource(
				&oauth2.Token{
					AccessToken: c.UserKey,
				},
			),
		),
	)

	if c.BaseURL != "" {
		u, err := url.Parse(c.BaseURL)
		if err != nil {
			return nil, err
		}
		orgClient.BaseURL = u
	}

	return &Clients{
		OrgName:    c.Organization,
		OrgClient:  orgClient,
		UserClient: userClient,
	}, nil

}

func (c *Clients) Fork(owner, repository, organization string) error {
	var opt *github.RepositoryCreateForkOptions
	if organization != "" {
		opt = &github.RepositoryCreateForkOptions{Organization: organization}
	}

	_, _, err := c.UserClient.Repositories.CreateFork(owner, repository, opt)

	return err
}
