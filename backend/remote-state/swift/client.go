package swift

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/objectstorage/v1/containers"
	"github.com/gophercloud/gophercloud/openstack/objectstorage/v1/objects"
	"github.com/gophercloud/gophercloud/pagination"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
)

const (
	TFSTATE_NAME = "tfstate.tf"

	consistencyTimeout = 15

	// Suffix that will be appended to state file paths
	// when locking
	lockSuffix = ".lock"

	// The TTL associated with this lock.
	lockTTL = 15
)

// RemoteClient implements the Client interface for an Openstack Swift server.
// Implements "state/remote".ClientLocker
type RemoteClient struct {
	client           *gophercloud.ServiceClient
	container        string
	archive          bool
	archiveContainer string
	expireSecs       int
	objectName       string

	mu sync.Mutex
	// lockState is true if we're using locks
	lockState bool

	info *state.LockInfo

	// lockCancel cancels the Context use for lockRenewPeriodic, and is
	// called when unlocking, or before creating a new lock if the lock is
	// lost.
	lockCancel context.CancelFunc
}

func (c *RemoteClient) ListObjectsNames(prefix string, delim string) ([]string, error) {
	if err := c.ensureContainerExists(); err != nil {
		return nil, err
	}

	// List our raw path
	listOpts := objects.ListOpts{
		Full:      false,
		Prefix:    prefix,
		Delimiter: delim,
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
	payload, err := c.get(c.container, c.objectName)

	// 404 response is to be expected if the object doesn't already exist!
	if _, ok := err.(gophercloud.ErrDefault404); ok {
		log.Println("[DEBUG] Object doesn't exist to download.")
		return nil, nil
	}

	return payload, err
}

// swift is eventually constistent. Consistency
// is ensured by the Get func which will always try
// to retrieve the most recent object
func (c *RemoteClient) Put(data []byte) error {
	if c.expireSecs != 0 {
		log.Printf("[DEBUG] ExpireSecs = %d", c.expireSecs)
		return c.put(c.container, c.objectName, data, c.expireSecs, "")
	}

	return c.put(c.container, c.objectName, data, -1, "")

}

func (c *RemoteClient) Delete() error {
	return c.delete(c.container, c.objectName)
}

func (c *RemoteClient) Lock(info *state.LockInfo) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.lockState {
		return "", nil
	}

	log.Printf("[DEBUG] Acquiring Lock %#v on %s/%s", info, c.container, c.objectName)

	// This check only is to ensure we strictly follow the specification.
	// Terraform shouldn't ever re-lock, so provide errors for the possible
	// states if this is called.
	if c.info != nil {
		// we have an active lock already
		return "", fmt.Errorf("state %q already locked", c.lockFilePath())
	}

	// update the path we're using
	info.Path = c.lockFilePath()

	if err := c.writeLockInfo(info, lockTTL, "*"); err != nil {
		return "", err
	}

	log.Printf("[DEBUG] Acquired Lock %s on %s", info.ID, c.objectName)

	c.info = info

	ctx, cancel := context.WithCancel(context.Background())
	c.lockCancel = cancel

	// keep the lock renewed
	go c.lockRenewPeriodic(ctx, info)

	return info.ID, nil
}

func (c *RemoteClient) Unlock(id string) error {
	c.mu.Lock()

	if !c.lockState {
		return nil
	}

	defer func() {
		// The periodic lock renew is canceled
		// the lockCancel func may not be nil in most usecases
		// but can typically be nil when using a second client
		// to ForceUnlock the state based on the same lock Id
		if c.lockCancel != nil {
			c.lockCancel()
		}
		c.info = nil
		c.mu.Unlock()
	}()

	log.Printf("[DEBUG] Releasing Lock %s on %s", id, c.objectName)

	info, err := c.lockInfo()
	if err != nil {
		return c.lockError(fmt.Errorf("failed to retrieve lock info: %s", err), nil)
	}

	c.info = info

	// conflicting lock
	if info.ID != id {
		return c.lockError(fmt.Errorf("lock id %q does not match existing lock", id), info)
	}

	return c.delete(c.container, c.lockFilePath())
}

func (c *RemoteClient) get(container, object string) (*remote.Payload, error) {
	log.Printf("[DEBUG] Getting object %s/%s", c.container, object)
	result := objects.Download(c.client, container, object, objects.DownloadOpts{Newest: true})

	// Extract any errors from result
	_, err := result.Extract()
	if err != nil {
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

func (c *RemoteClient) put(container, object string, data []byte, deleteAfter int, ifNoneMatch string) error {
	log.Printf("[DEBUG] Writing object in %s/%s", c.container, object)
	if err := c.ensureContainerExists(); err != nil {
		return err
	}

	contentType := "application/json"
	contentLength := int64(len(data))

	createOpts := objects.CreateOpts{
		Content:       bytes.NewReader(data),
		ContentType:   contentType,
		ContentLength: int64(contentLength),
	}

	if deleteAfter >= 0 {
		createOpts.DeleteAfter = deleteAfter
	}

	if ifNoneMatch != "" {
		createOpts.IfNoneMatch = ifNoneMatch
	}

	result := objects.Create(c.client, c.container, object, createOpts)
	if result.Err != nil {
		return result.Err
	}

	return nil
}

func (c *RemoteClient) delete(container, object string) error {
	log.Printf("[DEBUG] Deleting object %s/%s", c.container, c.objectName)
	result := objects.Delete(c.client, container, object, nil)

	if result.Err != nil {
		return result.Err
	}

	// swift is eventually consistent. As delete operations are synchronous
	// we check that objects are really deleted to be consistent
	data, err := c.get(container, object)
	if _, ok := err.(gophercloud.ErrDefault404); ok {
		log.Println("[DEBUG] Object to delete doesn't exist.")
		return nil
	}

	if err != nil {
		return err
	}

	if data != nil {
		return fmt.Errorf("object %s/%s hasnt been properly deleted. Data is not null: %#v", c.container, c.objectName, data)
	}

	return nil
}

func (c *RemoteClient) writeLockInfo(info *state.LockInfo, deleteAfter int, ifNoneMatch string) error {
	err := c.put(c.container, c.lockFilePath(), info.Marshal(), deleteAfter, ifNoneMatch)

	if httpErr, ok := err.(gophercloud.ErrUnexpectedResponseCode); ok && httpErr.Actual == 412 {
		log.Printf("[DEBUG] Couldn't write lock %s. One already exists.", info.ID)
		info2, err2 := c.lockInfo()
		if err2 != nil {
			return fmt.Errorf("Couldn't read lock info: %v", err2)
		}

		return c.lockError(err, info2)
	}

	if err != nil {
		return c.lockError(err, nil)
	}

	return nil
}

func (c *RemoteClient) lockError(err error, conflictingLock *state.LockInfo) *state.LockError {
	lockErr := &state.LockError{
		Err:  err,
		Info: conflictingLock,
	}

	return lockErr
}

// lockInfo reads the lock file, parses its contents and returns the parsed
// LockInfo struct.
func (c *RemoteClient) lockInfo() (*state.LockInfo, error) {
	raw, err := c.get(c.container, c.lockFilePath())
	if err != nil {
		return nil, err
	}

	info := &state.LockInfo{}

	if err := json.Unmarshal(raw.Data, info); err != nil {
		return nil, err
	}

	return info, nil
}

func (c *RemoteClient) lockRenewPeriodic(ctx context.Context, info *state.LockInfo) error {
	log.Printf("[DEBUG] Renew lock %v", info)

	ttl := lockTTL * time.Second
	waitDur := ttl / 2
	lastRenewTime := time.Now()
	var lastErr error
	for {
		if time.Since(lastRenewTime) > ttl {
			return lastErr
		}
		select {
		case <-time.After(waitDur):
			c.mu.Lock()
			// Unlock may have released the mu.Lock
			// in which case we shouldn't renew the lock
			select {
			case <-ctx.Done():
				log.Printf("[DEBUG] Stopping Periodic renew of lock %v", info)
				return nil
			default:
			}

			info2, err := c.lockInfo()
			if _, ok := err.(gophercloud.ErrDefault404); ok {
				log.Println("[DEBUG] Lock has expired trying to reacquire.")
				err = nil
			}

			if err == nil && (info2 == nil || info.ID == info2.ID) {
				info2 = info
				err = c.writeLockInfo(info, lockTTL, "")
			}

			c.mu.Unlock()

			if err != nil {
				log.Printf("[ERROR] could not reacquire lock (%v): %s", info, err)
				waitDur = time.Second
				lastErr = err
				continue
			}

			// conflicting lock
			if info2.ID != info.ID {
				return c.lockError(fmt.Errorf("lock id %q does not match existing lock %q", info.ID, info2.ID), info2)
			}

			waitDur = ttl / 2
			lastRenewTime = time.Now()

		case <-ctx.Done():
			log.Printf("[DEBUG] Stopping Periodic renew of lock %s", info.ID)
			return nil
		}
	}
}

func (c *RemoteClient) lockFilePath() string {
	return c.objectName + lockSuffix
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
