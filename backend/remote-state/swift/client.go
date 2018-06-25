package swift

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"log"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/objectstorage/v1/containers"
	"github.com/gophercloud/gophercloud/openstack/objectstorage/v1/objects"
	"github.com/gophercloud/gophercloud/pagination"

	"github.com/hashicorp/terraform/state/remote"
)

const (
	TFSTATE_LOCK_NAME = "tfstate.lock"
)

// RemoteClient implements the Client interface for an Openstack Swift server.
type RemoteClient struct {
	client           *gophercloud.ServiceClient
	container        string
	archive          bool
	archiveContainer string
	expireSecs       int
	objectName       string
}

func (c *RemoteClient) ListObjectsNames(prefix string) ([]string, error) {
	if err := c.ensureContainerExists(); err != nil {
		return nil, err
	}

	// List our raw path
	listOpts := objects.ListOpts{
		Full:   false,
		Prefix: prefix,
	}

	result := []string{}
	pager := objects.List(c.client, c.container, listOpts)
	// Define an anonymous function to be executed on each page's iteration
	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		objectList, err := objects.ExtractNames(page)
		if err != nil {
			return false, fmt.Errorf("Error extracting names from objects from page %+v", err)
		}
		for _, object := range objectList {
			result = append(result, object)
		}
		return true, nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil

}

func (c *RemoteClient) Get() (*remote.Payload, error) {
	log.Printf("[DEBUG] Getting object %s in container %s", c.objectName, c.container)
	if err := c.ensureContainerExists(); err != nil {
		return nil, err
	}

	result := objects.Download(c.client, c.container, c.objectName, nil)

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

	log.Printf("[DEBUG] Putting object %s in container %s", c.objectName, c.container)
	reader := bytes.NewReader(data)
	createOpts := objects.CreateOpts{
		Content: reader,
	}

	if c.expireSecs != 0 {
		log.Printf("[DEBUG] ExpireSecs = %d", c.expireSecs)
		createOpts.DeleteAfter = c.expireSecs
	}

	result := objects.Create(c.client, c.container, c.objectName, createOpts)

	return result.Err
}

func (c *RemoteClient) Delete() error {
	log.Printf("[DEBUG] Deleting object %s in container %s", c.objectName, c.container)
	result := objects.Delete(c.client, c.container, c.objectName, nil)
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
