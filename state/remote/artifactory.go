package remote

import (
	"crypto/md5"
	"fmt"
	"os"
	"strings"

	artifactory "github.com/lusis/go-artifactory/artifactory.v54"
)

// ARTIF_TFSTATE_NAME is the constant for the terraform state file stored in artifactory
const ARTIF_TFSTATE_NAME = "terraform.tfstate"

func artifactoryFactory(conf map[string]string) (Client, error) {
	var (
		userName   string
		password   string
		authmethod string
		token      string
	)
	token, hasToken := conf["token"]
	if !hasToken {
		token = os.Getenv("ARTIFACTORY_TOKEN")
		if token == "" {
			authmethod = "basic"
		} else {
			authmethod = "token"
		}
	} else {
		authmethod = "token"
	}
	if authmethod == "basic" {
		u, ok := conf["username"]
		if !ok {
			u = os.Getenv("ARTIFACTORY_USERNAME")
			if u == "" {
				return nil, fmt.Errorf(
					"missing 'username' configuration or ARTIFACTORY_USERNAME environment variable")
			}

		}
		p, ok := conf["password"]
		if !ok {
			p = os.Getenv("ARTIFACTORY_PASSWORD")
			if p == "" {
				return nil, fmt.Errorf(
					"missing 'password' configuration or ARTIFACTORY_PASSWORD environment variable")
			}
		}
		userName = u
		password = p
	}
	url, ok := conf["url"]
	if !ok {
		url = os.Getenv("ARTIFACTORY_URL")
		if url == "" {
			return nil, fmt.Errorf(
				"missing 'url' configuration or ARTIFACTORY_URL environment variable")
		}
	}
	repo, ok := conf["repo"]
	if !ok {
		return nil, fmt.Errorf(
			"missing 'repo' configuration")
	}
	subpath, ok := conf["subpath"]
	if !ok {
		return nil, fmt.Errorf(
			"missing 'subpath' configuration")
	}

	clientConf := artifactory.ClientConfig{
		BaseURL: url,
	}
	if authmethod == "token" {
		clientConf.Token = token
	} else {
		clientConf.Username = userName
		clientConf.Password = password
	}
	clientConf.AuthMethod = authmethod
	nativeClient := artifactory.NewClient(&clientConf)

	return &ArtifactoryClient{
		nativeClient: &nativeClient,
		userName:     userName,
		password:     password,
		token:        token,
		url:          url,
		repo:         repo,
		subpath:      subpath,
	}, nil

}

// ArtifactoryClient is a wrapper around artifactory.Client for terraform to use
type ArtifactoryClient struct {
	nativeClient *artifactory.Client
	userName     string
	password     string
	token        string
	url          string
	repo         string
	subpath      string
}

// Get gets a resource from Artifactory
func (c *ArtifactoryClient) Get() (*Payload, error) {
	p := fmt.Sprintf("%s/%s/%s", c.repo, c.subpath, ARTIF_TFSTATE_NAME)
	output, err := c.nativeClient.Get(p, make(map[string]string))
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return nil, nil
		}
		return nil, err
	}

	// TODO: migrate to using X-Checksum-Md5 header from artifactory
	// needs to be exposed by go-artifactory first

	hash := md5.Sum(output)
	payload := &Payload{
		Data: output,
		MD5:  hash[:md5.Size],
	}

	// If there was no data, then return nil
	if len(payload.Data) == 0 {
		return nil, nil
	}

	return payload, nil
}

// Put puts a resource in artifactory
func (c *ArtifactoryClient) Put(data []byte) error {
	p := fmt.Sprintf("%s/%s/%s", c.repo, c.subpath, ARTIF_TFSTATE_NAME)
	_, err := c.nativeClient.Put(p, data, make(map[string]string))
	return fmt.Errorf("Failed to upload state: %v", err)
}

// Delete deletes a resource in artifactory
func (c *ArtifactoryClient) Delete() error {
	p := fmt.Sprintf("%s/%s/%s", c.repo, c.subpath, ARTIF_TFSTATE_NAME)
	err := c.nativeClient.Delete(p)
	return err
}
