package artifactory

import (
	"crypto/md5"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
	artifactory "github.com/lusis/go-artifactory/src/artifactory.v401"
)

const ARTIF_TFSTATE_NAME = "terraform.tfstate"
const ARTIF_TFLOCK_NAME = "terraform.lock"

type ArtifactoryClient struct {
	nativeClient       *artifactory.ArtifactoryClient
	lockNativeClient   *artifactory.ArtifactoryClient
	unlockNativeClient *artifactory.ArtifactoryClient
	userName           string
	password           string
	url                string
	repo               string
	subpath            string
	lockUserName       string
	lockPassword       string
	unlockUserName     string
	unlockPassword     string
	lockUrl            string
	lockRepo           string
	lockSubpath        string
	lockID             string
	jsonLockInfo       []byte
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

func (c *ArtifactoryClient) Lock(info *state.LockInfo) (string, error) {
	if c.lockUrl == "" {
		return "", nil
	}
	c.lockID = ""

	jsonLockInfo := info.Marshal()
	p := fmt.Sprintf("%s/%s/%s", c.lockRepo, c.lockSubpath, ARTIF_TFLOCK_NAME)
	if _, err := c.lockNativeClient.Put(p, string(jsonLockInfo), make(map[string]string)); err == nil {
		c.lockID = info.ID
		c.jsonLockInfo = jsonLockInfo
		return info.ID, nil
	} else {
		return "", fmt.Errorf("Failed to lock: %v", err)
	}
}

func (c *ArtifactoryClient) Unlock(id string) error {
	if c.lockUrl == "" {
		return nil
	}
	p := fmt.Sprintf("%s/%s/%s", c.lockRepo, c.lockSubpath, ARTIF_TFLOCK_NAME)
	var err error
	if c.unlockUserName == "" {
		err = c.lockNativeClient.Delete(p)
	} else {
		err = c.unlockNativeClient.Delete(p)
	}
	return err
}
