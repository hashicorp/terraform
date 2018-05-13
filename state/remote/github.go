package remote

import (
	"crypto/md5"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"

	"github.com/google/go-github/github"
)

// GithubClient is a remote client that stores data in a GitHub
// repository.
type GithubClient struct {
	Owner      string
	Token      string
	Repository string
	Branch     string
	StatePath  string
	LockPath   string
	client     *github.Client
}

// githubFactory returns initializes and returns a GithubClient struct.
func githubFactory(conf map[string]string) (Client, error) {
	owner, ok := conf["owner"]
	if !ok {
		return nil, fmt.Errorf("missing 'owner' configuration")
	}
	repository, ok := conf["repository"]
	if !ok {
		return nil, fmt.Errorf("missing 'repository' configuration")
	}
	token, ok := conf["token"]
	if !ok {
		token = os.Getenv("GITHUB_TOKEN")
		if token == "" {
			return nil, fmt.Errorf("missing 'token' configuration")
		}
	}
	branch, ok := conf["branch"]
	if !ok {
		branch = "master"
	}
	statePath, ok := conf["state_path"]
	if !ok {
		statePath = "terraform.tfstate"
	}
	lockPath, ok := conf["lock_path"]
	if !ok {
		lockPath = "terraform.tfstate.lock"
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(oauth2.NoContext, ts)
	client := github.NewClient(tc)

	baseURL := conf["base_url"]
	if baseURL != "" {
		u, err := url.Parse(baseURL)
		if err != nil {
			return nil, err
		}
		client.BaseURL = u
	}

	ghc := &GithubClient{
		Owner:      owner,
		Token:      token,
		Repository: repository,
		Branch:     branch,
		StatePath:  statePath,
		LockPath:   lockPath,
		client:     client,
	}
	return ghc, nil
}

// Get returns state from the github backend.
func (c *GithubClient) Get() (*Payload, error) {
	s := c.client.Repositories
	file_content, dc, _, err := s.GetContents(context.Background(),
		c.Owner, c.Repository, c.StatePath,
		&github.RepositoryContentGetOptions{Ref: c.Branch})
	if isStatusNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if dc != nil {
		return nil, errors.New("State file appears to be a directory, uh-oh...")
	}

	contents, err := file_content.GetContent()
	if err != nil {
		return nil, err
	}
	data := []byte(contents)
	md5 := md5.Sum(data)

	return &Payload{
		Data: data,
		MD5:  md5[:],
	}, nil
}

// Put writes state to the github backend.
func (c *GithubClient) Put(data []byte) error {
	err := c.tryPut(data)
	for err != nil && isStatusConflict(err) {
		err = c.tryPut(data)
	}
	return err
}

// tryPut tries once to write state to the github backend.
func (c *GithubClient) tryPut(data []byte) error {
	s := c.client.Repositories

	fc, _, _, err := s.GetContents(context.Background(), c.Owner, c.Repository, c.StatePath,
		&github.RepositoryContentGetOptions{Ref: c.Branch})
	if err != nil && !isStatusNotFound(err) { // an absent file is ok
		return err
	}

	content := github.RepositoryContentFileOptions{
		Branch:  github.String(c.Branch),
		Message: github.String("Save state in file in github backend"),
		Content: data,
	}
	if fc != nil {
		content.SHA = fc.SHA
	}

	_, _, err = s.UpdateFile(context.Background(), c.Owner, c.Repository, c.StatePath, &content)

	return err
}

// Delete removes the state from the github backend (deletes the file).
func (c *GithubClient) Delete() error {

	err := c.tryDelete()
	for err != nil && isStatusConflict(err) {
		err = c.tryDelete()
	}
	return err
}

// tryDelete tries once to remove the state from the github backend.
func (c *GithubClient) tryDelete() error {
	s := c.client.Repositories

	fc, _, _, err := s.GetContents(context.Background(), c.Owner, c.Repository, c.StatePath,
		&github.RepositoryContentGetOptions{Ref: c.Branch})
	if isStatusNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}

	content := github.RepositoryContentFileOptions{
		Branch:  github.String(c.Branch),
		Message: github.String("Deleting state file from github backend"),
		SHA:     fc.SHA,
	}

	_, _, err = s.DeleteFile(context.Background(), c.Owner, c.Repository, c.StatePath, &content)
	if isStatusNotFound(err) {
		return nil
	}

	return err
}

// Lock locks the github back end by creating a lock file.  Because
// the sha of the file to be created is nil the file must not exist.
// If the file does exist then the UpdateFile will fail.
func (c *GithubClient) Lock(info string) error {
	s := c.client.Repositories

	content := github.RepositoryContentFileOptions{
		Branch:  github.String(c.Branch),
		Message: github.String("Locking Terraform's github backend"),
		Content: []byte("Locked"),
	}

	_, _, err := s.UpdateFile(context.Background(), c.Owner, c.Repository, c.LockPath, &content)
	if err != nil {
		return fmt.Errorf("Unable to lock Terraform's github backend: %v", err)
	}

	return nil
}

func (c *GithubClient) Unlock() error {
	s := c.client.Repositories

	fc, _, _, err := s.GetContents(context.Background(), c.Owner, c.Repository, c.LockPath,
		&github.RepositoryContentGetOptions{Ref: c.Branch})
	if isStatusNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}

	content := github.RepositoryContentFileOptions{
		Branch:  github.String(c.Branch),
		Message: github.String("Unlocking Terraform's github backend"),
		SHA:     fc.SHA,
	}

	_, _, err = s.DeleteFile(context.Background(), c.Owner, c.Repository, c.LockPath, &content)
	if isStatusNotFound(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("Unable to unlock Terraform's github backend: %v", err)
	}

	return nil
}

func isStatusNotFound(err error) bool {
	if err, ok := err.(*github.ErrorResponse); ok && err.Response.StatusCode == http.StatusNotFound {
		return true
	}
	return false
}

func isStatusConflict(err error) bool {
	if err, ok := err.(*github.ErrorResponse); ok && err.Response.StatusCode == http.StatusConflict {
		return true
	}
	return false
}
