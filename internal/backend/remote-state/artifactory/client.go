package artifactory

import (
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/hashicorp/terraform/internal/states/remote"
	"github.com/jfrog/jfrog-client-go/artifactory"
	"github.com/jfrog/jfrog-client-go/artifactory/services"
)

const ARTIF_TFSTATE_NAME = "terraform.tfstate"

type ArtifactoryClient struct {
	nativeClient artifactory.ArtifactoryServicesManager
	repo         string
	subpath      string
}

func (c *ArtifactoryClient) Get() (*remote.Payload, error) {
	params := services.NewDownloadParams()
	params.Pattern = fmt.Sprintf("%s/%s/%s", c.repo, c.subpath, ARTIF_TFSTATE_NAME)
	params.Target = os.TempDir()

	downloaded, _, err := c.nativeClient.DownloadFiles(params)
	if err != nil {
		return nil, fmt.Errorf("failed to download state: %v", err)
	}

	if downloaded <= 0 {
		return nil, nil
	}

	tmpState := path.Join(params.Target, c.subpath, ARTIF_TFSTATE_NAME)
	output, err := ioutil.ReadFile(tmpState)
	if err != nil {
		return nil, fmt.Errorf("failed to read state: %v", err)
	}
	defer os.Remove(tmpState)

	// TODO: migrate to using X-Checksum-Md5 header from artifactory
	// needs to be exposed by go-artifactory first

	hash := md5.Sum(output)
	payload := &remote.Payload{
		Data: output,
		MD5:  hash[:md5.Size],
	}

	return payload, nil
}

func (c *ArtifactoryClient) Put(data []byte) error {
	params := services.NewUploadParams()
	params.Pattern = path.Join(os.TempDir(), c.subpath, ARTIF_TFSTATE_NAME)
	params.Target = fmt.Sprintf("%s/%s/", c.repo, c.subpath)
	params.IncludeDirs = false
	params.Flat = true

	if err := ioutil.WriteFile(params.Pattern, data, 0755); err != nil {
		return err
	}
	defer os.Remove(params.Pattern)

	_, _, err := c.nativeClient.UploadFiles(params)
	if err != nil {
		return fmt.Errorf("failed to upload state: %v", err)
	}
	return nil
}

func (c *ArtifactoryClient) Delete() error {
	params := services.NewDeleteParams()
	params.Pattern = fmt.Sprintf("%s/%s/%s", c.repo, c.subpath, ARTIF_TFSTATE_NAME)
	pathToDelete, err := c.nativeClient.GetPathsToDelete(params)
	if err != nil {
		return err
	}
	if _, err := c.nativeClient.DeleteFiles(pathToDelete); err != nil {
		return fmt.Errorf("failed to delete state: %v", err)
	}
	return nil
}
