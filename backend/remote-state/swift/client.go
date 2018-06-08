package swift

import (
	"bytes"
	"crypto/md5"
	"log"
	"os"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/objectstorage/v1/containers"
	"github.com/gophercloud/gophercloud/openstack/objectstorage/v1/objects"

	"github.com/hashicorp/terraform/state/remote"
)

const (
	TFSTATE_NAME      = "tfstate.tf"
	TFSTATE_LOCK_NAME = "tfstate.lock"
)

// RemoteClient implements the Client interface for an Openstack Swift server.
type RemoteClient struct {
	client           *gophercloud.ServiceClient
	container        string
	archive          bool
	archiveContainer string
	expireSecs       int
}

func (c *RemoteClient) Get() (*remote.Payload, error) {
	log.Printf("[DEBUG] Getting object %s in container %s", TFSTATE_NAME, c.container)
	result := objects.Download(c.client, c.container, TFSTATE_NAME, nil)

	// Extract any errors from result
	_, err := result.Extract()

	// 404 response is to be expected if the object doesn't already exist!
	if _, ok := err.(gophercloud.ErrDefault404); ok {
		log.Println("[DEBUG] Object doesn't exist to download.")
		return nil, nil
	}

	bytes, err := result.ExtractContent()
	if err != nil {
		return nil, err
	}

	hash := md5.Sum(bytes)
	payload := &remote.Payload{
		Data: bytes,
		MD5:  hash[:md5.Size],
	}

	return payload, nil
}

func (c *RemoteClient) Put(data []byte) error {
	if err := c.ensureContainerExists(); err != nil {
		return err
	}

	log.Printf("[DEBUG] Putting object %s in container %s", TFSTATE_NAME, c.container)
	reader := bytes.NewReader(data)
	createOpts := objects.CreateOpts{
		Content: reader,
	}

	if c.expireSecs != 0 {
		log.Printf("[DEBUG] ExpireSecs = %d", c.expireSecs)
		createOpts.DeleteAfter = c.expireSecs
	}

	result := objects.Create(c.client, c.container, TFSTATE_NAME, createOpts)

	return result.Err
}

func (c *RemoteClient) Delete() error {
	log.Printf("[DEBUG] Deleting object %s in container %s", TFSTATE_NAME, c.container)
	result := objects.Delete(c.client, c.container, TFSTATE_NAME, nil)
	return result.Err
}

func (c *RemoteClient) ensureContainerExists() error {
	containerOpts := &containers.CreateOpts{}

	if c.archive {
		log.Printf("[DEBUG] Creating archive container %s", c.archiveContainer)
		result := containers.Create(c.client, c.archiveContainer, nil)
		if result.Err != nil {
			log.Printf("[DEBUG] Error creating archive container %s: %s", c.archiveContainer, result.Err)
			return result.Err
		}

		log.Printf("[DEBUG] Enabling Versioning on container %s", c.container)
		containerOpts.VersionsLocation = c.archiveContainer
	}

	log.Printf("[DEBUG] Creating container %s", c.container)
	result := containers.Create(c.client, c.container, containerOpts)
	if result.Err != nil {
		return result.Err
	}

	return nil
}

func multiEnv(ks []string) string {
	for _, k := range ks {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return ""
}
