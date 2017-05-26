package inmem

import (
	"crypto/md5"
	"errors"
	"time"

	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
)

// RemoteClient is a remote client that stores data in memory for testing.
type RemoteClient struct {
	Data []byte
	MD5  []byte

	LockInfo *state.LockInfo
}

func (c *RemoteClient) Get() (*remote.Payload, error) {
	if c.Data == nil {
		return nil, nil
	}

	return &remote.Payload{
		Data: c.Data,
		MD5:  c.MD5,
	}, nil
}

func (c *RemoteClient) Put(data []byte) error {
	md5 := md5.Sum(data)

	c.Data = data
	c.MD5 = md5[:]
	return nil
}

func (c *RemoteClient) Delete() error {
	c.Data = nil
	c.MD5 = nil
	return nil
}

func (c *RemoteClient) Lock(info *state.LockInfo) (string, error) {
	lockErr := &state.LockError{
		Info: &state.LockInfo{},
	}

	if c.LockInfo != nil {
		lockErr.Err = errors.New("state locked")
		// make a copy of the lock info to avoid any testing shenanigans
		*lockErr.Info = *c.LockInfo
		return "", lockErr
	}

	info.Created = time.Now().UTC()
	c.LockInfo = info

	return c.LockInfo.ID, nil
}

func (c *RemoteClient) Unlock(id string) error {
	if c.LockInfo == nil {
		return errors.New("state not locked")
	}

	lockErr := &state.LockError{
		Info: &state.LockInfo{},
	}
	if id != c.LockInfo.ID {
		lockErr.Err = errors.New("invalid lock id")
		*lockErr.Info = *c.LockInfo
		return lockErr
	}

	c.LockInfo = nil
	return nil
}
