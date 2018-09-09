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
	userName, err := getArtifactoryVar("username", conf)
	if err != nil {
		return nil, err
	}

	password, err := getArtifactoryVar("password", conf)
	if err != nil {
		return nil, err
	}

	url, err := getArtifactoryVar("url", conf)
	if err != nil {
		return nil, err
	}

	repo, err := getArtifactoryVar("repo", conf)
	if err != nil {
		return nil, err
	}

	subpath, err := getArtifactoryVar("subpath", conf)
	if err != nil {
		return nil, err
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

func getArtifactoryVar(vr string, conf map[string]string) (string, error) {
	envvr := fmt.Sprintf("ARTIFACTORY_%v", strings.ToUpper(vr))

	val, ok := conf[vr]
	if !ok {
		val = os.Getenv(envvr)
		if val == "" {
			return val, fmt.Errorf(
				"missing '%v' configuration or %v environment variable",
				vr,
				envvr,
			)
		}
	}

	return val, nil
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
