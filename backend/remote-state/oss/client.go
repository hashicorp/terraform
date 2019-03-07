package oss

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	uuid "github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
	"log"
	"sync"
)

type RemoteClient struct {
	ossClient            *oss.Client
	bucketName           string
	stateFile            string
	lockFile             string
	serverSideEncryption bool
	acl                  string
	doLock               bool
	info                 *state.LockInfo
	mu                   sync.Mutex
}

func (c *RemoteClient) Get() (payload *remote.Payload, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	buf, err := c.getObj(c.stateFile)
	if err != nil {
		return nil, err
	}

	// If there was no data, then return nil
	if buf == nil || len(buf.Bytes()) == 0 {
		log.Printf("[DEBUG] State %s has no data.", c.stateFile)
		return nil, nil
	}
	md5 := md5.Sum(buf.Bytes())

	payload = &remote.Payload{
		Data: buf.Bytes(),
		MD5:  md5[:],
	}
	return payload, nil
}

func (c *RemoteClient) Put(data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.putObj(c.stateFile, data)
}

func (c *RemoteClient) Delete() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.deleteObj(c.stateFile)
}

func (c *RemoteClient) Lock(info *state.LockInfo) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.doLock {
		return "", nil
	}

	bucket, err := c.ossClient.Bucket(c.bucketName)
	if err != nil {
		return "", fmt.Errorf("Error getting bucket: %#v", err)
	}

	infoJson, err := json.Marshal(info)
	if err != nil {
		return "", err
	}

	if info.ID == "" {
		lockID, err := uuid.GenerateUUID()
		if err != nil {
			return "", err
		}
		info.ID = lockID
	}

	info.Path = c.lockFile
	exist, err := bucket.IsObjectExist(info.Path)
	if err != nil {
		return "", fmt.Errorf("Error checking object %s: %#v", info.Path, err)
	}
	if !exist {
		if err := c.putObj(info.Path, infoJson); err != nil {
			return "", err
		}
	} else if _, err := c.validLock(info.ID); err != nil {
		return "", err
	}

	return info.ID, nil
}

func (c *RemoteClient) Unlock(id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.doLock {
		return nil
	}

	lockInfo, err := c.validLock(id)
	if err != nil {
		return err
	}

	if err := c.deleteObj(c.lockFile); err != nil {
		return &state.LockError{
			Info: lockInfo,
			Err:  err,
		}
	}
	return nil
}

func (c *RemoteClient) putObj(key string, data []byte) error {
	bucket, err := c.ossClient.Bucket(c.bucketName)
	if err != nil {
		return fmt.Errorf("Error getting bucket: %#v", err)
	}
	body := bytes.NewReader(data)

	var options []oss.Option
	if c.acl != "" {
		options = append(options, oss.ACL(oss.ACLType(c.acl)))
	}
	options = append(options, oss.ContentType("application/json"))
	if c.serverSideEncryption {
		options = append(options, oss.ServerSideEncryption("AES256"))
	}
	options = append(options, oss.ContentLength(int64(len(data))))

	if body != nil {
		if err := bucket.PutObject(key, body, options...); err != nil {
			return fmt.Errorf("failed to upload %s: %#v", key, err)
		}
		return nil
	}
	return nil
}

func (c *RemoteClient) getObj(key string) (*bytes.Buffer, error) {
	bucket, err := c.ossClient.Bucket(c.bucketName)
	if err != nil {
		return nil, fmt.Errorf("Error getting bucket: %#v", err)
	}

	if exist, err := bucket.IsObjectExist(key); err != nil {
		return nil, fmt.Errorf("Estimating object %s is exist got an error: %#v", key, err)
	} else if !exist {
		return nil, nil
	}

	var options []oss.Option
	output, err := bucket.GetObject(key, options...)
	if err != nil {
		return nil, fmt.Errorf("Error getting object: %#v", err)
	}

	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, output); err != nil {
		return nil, fmt.Errorf("Failed to read remote state: %s", err)
	}
	return buf, nil
}

func (c *RemoteClient) deleteObj(key string) error {
	bucket, err := c.ossClient.Bucket(c.bucketName)
	if err != nil {
		return fmt.Errorf("Error getting bucket: %#v", err)
	}

	if err := bucket.DeleteObject(key); err != nil {
		return fmt.Errorf("Error deleting object %s: %#v", key, err)
	}
	return nil
}

// lockInfo reads the lock file, parses its contents and returns the parsed
// LockInfo struct.
func (c *RemoteClient) lockInfo() (*state.LockInfo, error) {
	buf, err := c.getObj(c.lockFile)
	if err != nil {
		return nil, err
	}
	if buf == nil || len(buf.Bytes()) == 0 {
		return nil, nil
	}
	info := &state.LockInfo{}
	if err := json.Unmarshal(buf.Bytes(), info); err != nil {
		return nil, err
	}

	return info, nil
}

func (c *RemoteClient) validLock(id string) (*state.LockInfo, *state.LockError) {
	lockErr := &state.LockError{}
	lockInfo, err := c.lockInfo()
	if err != nil {
		lockErr.Err = fmt.Errorf("failed to retrieve lock info: %s", err)
		return nil, lockErr
	}
	lockErr.Info = lockInfo

	if lockInfo.ID != id {
		lockErr.Err = fmt.Errorf("lock id %q does not match existing lock", id)
		return nil, lockErr
	}
	return lockInfo, nil
}
