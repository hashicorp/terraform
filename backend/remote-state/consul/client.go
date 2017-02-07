package consul

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"time"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/errwrap"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
)

const (
	lockSuffix     = "/.lock"
	lockInfoSuffix = "/.lockinfo"
)

// RemoteClient is a remote client that stores data in Consul.
type RemoteClient struct {
	Client *consulapi.Client
	Path   string

	consulLock *consulapi.Lock
	lockCh     <-chan struct{}
}

func (c *RemoteClient) Get() (*remote.Payload, error) {
	pair, _, err := c.Client.KV().Get(c.Path, nil)
	if err != nil {
		return nil, err
	}
	if pair == nil {
		return nil, nil
	}

	md5 := md5.Sum(pair.Value)
	return &remote.Payload{
		Data: pair.Value,
		MD5:  md5[:],
	}, nil
}

func (c *RemoteClient) Put(data []byte) error {
	kv := c.Client.KV()
	_, err := kv.Put(&consulapi.KVPair{
		Key:   c.Path,
		Value: data,
	}, nil)
	return err
}

func (c *RemoteClient) Delete() error {
	kv := c.Client.KV()
	_, err := kv.Delete(c.Path, nil)
	return err
}

func (c *RemoteClient) putLockInfo(info string) error {
	li := &state.LockInfo{
		Path:    c.Path,
		Created: time.Now().UTC(),
		Info:    info,
	}

	js, err := json.Marshal(li)
	if err != nil {
		return err
	}

	kv := c.Client.KV()
	_, err = kv.Put(&consulapi.KVPair{
		Key:   c.Path + lockInfoSuffix,
		Value: js,
	}, nil)

	return err
}

func (c *RemoteClient) getLockInfo() (*state.LockInfo, error) {
	path := c.Path + lockInfoSuffix
	pair, _, err := c.Client.KV().Get(path, nil)
	if err != nil {
		return nil, err
	}
	if pair == nil {
		return nil, nil
	}

	li := &state.LockInfo{}
	err = json.Unmarshal(pair.Value, li)
	if err != nil {
		return nil, errwrap.Wrapf("error unmarshaling lock info: {{err}}", err)
	}

	return li, nil
}

func (c *RemoteClient) Lock(info string) error {
	select {
	case <-c.lockCh:
		// We had a lock, but lost it.
		// Since we typically only call lock once, we shouldn't ever see this.
		return errors.New("lost consul lock")
	default:
		if c.lockCh != nil {
			// we have an active lock already
			return nil
		}
	}

	if c.consulLock == nil {
		opts := &consulapi.LockOptions{
			Key: c.Path + lockSuffix,
			// We currently don't procide any options to block terraform and
			// retry lock acquisition, but we can wait briefly in case the
			// lock is about to be freed.
			LockWaitTime: time.Second,
			LockTryOnce:  true,
		}

		lock, err := c.Client.LockOpts(opts)
		if err != nil {
			return nil
		}

		c.consulLock = lock
	}

	lockCh, err := c.consulLock.Lock(make(chan struct{}))
	if err != nil {
		return err
	}

	if lockCh == nil {
		lockInfo, e := c.getLockInfo()
		if e != nil {
			return e
		}
		return lockInfo.Err()
	}

	c.lockCh = lockCh

	err = c.putLockInfo(info)
	if err != nil {
		err = multierror.Append(err, c.Unlock())
		return err
	}

	return nil
}

func (c *RemoteClient) Unlock() error {
	if c.consulLock == nil || c.lockCh == nil {
		return nil
	}

	select {
	case <-c.lockCh:
		return errors.New("consul lock was lost")
	default:
	}

	err := c.consulLock.Unlock()
	c.lockCh = nil

	kv := c.Client.KV()
	_, delErr := kv.Delete(c.Path+lockInfoSuffix, nil)
	if delErr != nil {
		err = multierror.Append(err, delErr)
	}

	return err
}
