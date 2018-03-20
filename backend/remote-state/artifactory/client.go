package artifactory

import (
	"crypto/md5"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/state/remote"
	artifactory "github.com/lusis/go-artifactory/src/artifactory.v401"
)

const ARTIF_TFSTATE_NAME = "terraform.tfstate"

type ArtifactoryClient struct {
	nativeClient *artifactory.ArtifactoryClient
	userName     string
	password     string
	url          string
	repo         string
	subpath      string
}

func (c *ArtifactoryClient) Get() (*remote.Payload, error) {
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
	payload := &remote.Payload{
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
