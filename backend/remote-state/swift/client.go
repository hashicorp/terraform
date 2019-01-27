package swift

import (
	"bytes"
	"crypto/md5"
	"log"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/objectstorage/v1/containers"
	"github.com/gophercloud/gophercloud/openstack/objectstorage/v1/objects"

	"github.com/hashicorp/terraform/state/remote"
)

const (
	DEFAULT_NAME        = "tfstate"
	TFSTATE_SUFFIX      = ".tf"
	TFSTATE_LOCK_SUFFIX = ".lock"
)

// RemoteClient implements the Client interface for an Openstack Swift server.
type RemoteClient struct {
	name             string
	client           *gophercloud.ServiceClient
	container        string
	archive          bool
	archiveContainer string
	expireSecs       int
}

func (c *RemoteClient) Get() (*remote.Payload, error) {
	container, prefix := getContainerAndPrefix(c.container)

	log.Printf("[DEBUG] Getting object %s in container %s", prefix+c.name+TFSTATE_SUFFIX, container)
	result := objects.Download(c.client, container, prefix+c.name+TFSTATE_SUFFIX, nil)

	// Extract any errors from result
	_, err := result.Extract()
	if err != nil {
		// 404 response is to be expected if the object doesn't already exist!
		if _, ok := err.(gophercloud.ErrDefault404); ok {
			log.Println("[DEBUG] Object doesn't exist to download.")
			return nil, nil
		}
		return nil, err
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

	container, prefix := getContainerAndPrefix(c.container)

	log.Printf("[DEBUG] Putting object %s in container %s", prefix+c.name+TFSTATE_SUFFIX, container)
	reader := bytes.NewReader(data)
	createOpts := objects.CreateOpts{
		Content: reader,
	}

	if c.expireSecs != 0 {
		log.Printf("[DEBUG] ExpireSecs = %d", c.expireSecs)
		createOpts.DeleteAfter = c.expireSecs
	}

	result := objects.Create(c.client, container, prefix+c.name+TFSTATE_SUFFIX, createOpts)

	return result.Err
}

func (c *RemoteClient) Delete() error {
	container, prefix := getContainerAndPrefix(c.container)

	log.Printf("[DEBUG] Deleting object %s in container %s", prefix+c.name+TFSTATE_SUFFIX, container)
	result := objects.Delete(c.client, container, prefix+c.name+TFSTATE_SUFFIX, nil)

	if _, ok := result.Err.(gophercloud.ErrDefault404); ok {
		return nil
	}

	return result.Err
}

func (c *RemoteClient) ensureContainerExists() error {
	containerOpts := &containers.CreateOpts{}

	if c.archive {
		container, _ := getContainerAndPrefix(c.archiveContainer)

		log.Printf("[DEBUG] Creating archive container %s", container)
		result := containers.Create(c.client, container, nil)
		if result.Err != nil {
			log.Printf("[DEBUG] Error creating archive container %s: %s", container, result.Err)
			return result.Err
		}

		log.Printf("[DEBUG] Enabling Versioning on container %s", c.container)
		containerOpts.VersionsLocation = container
	}

	container, _ := getContainerAndPrefix(c.container)

	log.Printf("[DEBUG] Creating container %s", container)
	result := containers.Create(c.client, container, containerOpts)
	if result.Err != nil {
		return result.Err
	}

	return nil
}
