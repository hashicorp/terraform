package consul

import (
	"bytes"
	"compress/gzip"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	consulapi "github.com/hashicorp/consul/api"
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
	GZip   bool

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

	payload := pair.Value
	// If the payload starts with 0x1f, it's gzip, not json
	if len(pair.Value) >= 1 && pair.Value[0] == '\x1f' {
		if data, err := uncompressState(pair.Value); err == nil {
			payload = data
		} else {
			return nil, err
		}
	}

	md5 := md5.Sum(pair.Value)
	return &remote.Payload{
		Data: payload,
		MD5:  md5[:],
	}, nil
}

func (c *RemoteClient) Put(data []byte) error {
	payload := data
	if c.GZip {
		if compressedState, err := compressState(data); err == nil {
			payload = compressedState
		} else {
			return err
		}
	}

	kv := c.Client.KV()
	_, err := kv.Put(&consulapi.KVPair{
		Key:   c.Path,
		Value: payload,
	}, nil)
	return err
}

func (c *RemoteClient) Delete() error {
	kv := c.Client.KV()
	_, err := kv.Delete(c.Path, nil)
	return err
}

func (c *RemoteClient) putLockInfo(info *state.LockInfo) error {
	info.Path = c.Path
	info.Created = time.Now().UTC()

	kv := c.Client.KV()
	_, err := kv.Put(&consulapi.KVPair{
		Key:   c.Path + lockInfoSuffix,
		Value: info.Marshal(),
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
		return nil, fmt.Errorf("error unmarshaling lock info: %s", err)
	}

	return li, nil
}

func (c *RemoteClient) Lock(info *state.LockInfo) (string, error) {
	select {
	case <-c.lockCh:
		// We had a lock, but lost it.
		// Since we typically only call lock once, we shouldn't ever see this.
		return "", errors.New("lost consul lock")
	default:
		if c.lockCh != nil {
			// we have an active lock already
			return "", fmt.Errorf("state %q already locked", c.Path)
		}
	}

	if c.consulLock == nil {
		opts := &consulapi.LockOptions{
			Key: c.Path + lockSuffix,
			// only wait briefly, so terraform has the choice to fail fast or
			// retry as needed.
			LockWaitTime: time.Second,
			LockTryOnce:  true,
		}

		lock, err := c.Client.LockOpts(opts)
		if err != nil {
			return "", err
		}

		c.consulLock = lock
	}

	lockErr := &state.LockError{}

	lockCh, err := c.consulLock.Lock(make(chan struct{}))
	if err != nil {
		lockErr.Err = err
		return "", lockErr
	}

	if lockCh == nil {
		lockInfo, e := c.getLockInfo()
		if e != nil {
			lockErr.Err = e
			return "", lockErr
		}

		lockErr.Info = lockInfo
		return "", lockErr
	}

	c.lockCh = lockCh

	err = c.putLockInfo(info)
	if err != nil {
		if unlockErr := c.Unlock(info.ID); unlockErr != nil {
			err = multierror.Append(err, unlockErr)
		}

		return "", err
	}

	return info.ID, nil
}

func (c *RemoteClient) Unlock(id string) error {
	// this doesn't use the lock id, because the lock is tied to the consul client.
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

	// This is only cleanup, and will fail if the lock was immediately taken by
	// another client, so we don't report an error to the user here.
	c.consulLock.Destroy()

	kv := c.Client.KV()
	_, delErr := kv.Delete(c.Path+lockInfoSuffix, nil)
	if delErr != nil {
		err = multierror.Append(err, delErr)
	}

	return err
}

func compressState(data []byte) ([]byte, error) {
	b := new(bytes.Buffer)
	gz := gzip.NewWriter(b)
	if _, err := gz.Write(data); err != nil {
		return nil, err
	}
	if err := gz.Flush(); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func uncompressState(data []byte) ([]byte, error) {
	b := new(bytes.Buffer)
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	b.ReadFrom(gz)
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
