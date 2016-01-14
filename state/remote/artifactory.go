package remote

import (
	"crypto/md5"
	"fmt"
	"os"
	"strings"

	artifactory "github.com/lusis/go-artifactory/src/artifactory.v401"
)

const ARTIF_TFSTATE_NAME = "terraform.tfstate"

func artifactoryFactory(conf map[string]string) (Client, error) {
	userName, ok := conf["username"]
	if !ok {
		userName = os.Getenv("ARTIFACTORY_USERNAME")
		if userName == "" {
			return nil, fmt.Errorf(
				"missing 'username' configuration or ARTIFACTORY_USERNAME environment variable")
		}
	}
	password, ok := conf["password"]
	if !ok {
		password = os.Getenv("ARTIFACTORY_PASSWORD")
		if password == "" {
			return nil, fmt.Errorf(
				"missing 'password' configuration or ARTIFACTORY_PASSWORD environment variable")
		}
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

	clientConf := &artifactory.ClientConfig{
		BaseURL:  url,
		Username: userName,
		Password: password,
	}
	nativeClient := artifactory.NewClient(clientConf)

	return &ArtifactoryClient{
		nativeClient: &nativeClient,
		userName:     userName,
		password:     password,
		url:          url,
		repo:         repo,
		subpath:      subpath,
	}, nil

}

type ArtifactoryClient struct {
	nativeClient *artifactory.ArtifactoryClient
	userName     string
	password     string
	url          string
	repo         string
	subpath      string
}

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

func (c *ArtifactoryClient) Put(data []byte) error {
	p := fmt.Sprintf("%s/%s/%s", c.repo, c.subpath, ARTIF_TFSTATE_NAME)
	if _, err := c.nativeClient.Put(p, string(data), make(map[string]string)); err == nil {
		return nil
	} else {
		return fmt.Errorf("Failed to upload state: %v", err)
	}
}

func (c *ArtifactoryClient) Delete() error {
	p := fmt.Sprintf("%s/%s/%s", c.repo, c.subpath, ARTIF_TFSTATE_NAME)
	err := c.nativeClient.Delete(p)
	return err
}
