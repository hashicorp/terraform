package etcd

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	etcdv3 "github.com/coreos/etcd/clientv3"
	etcdv3sync "github.com/coreos/etcd/clientv3/concurrency"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
)

const (
	lockAcquireTimeout = 2 * time.Second
	lockInfoSuffix     = ".lockinfo"
)

// RemoteClient is a remote client that will store data in etcd.
type RemoteClient struct {
	Client *etcdv3.Client
	DoLock bool
	Key    string

	etcdMutex   *etcdv3sync.Mutex
	etcdSession *etcdv3sync.Session
	info        *state.LockInfo
	mu          sync.Mutex
	modRevision int64
}

func (c *RemoteClient) Get() (*remote.Payload, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	res, err := c.Client.KV.Get(context.TODO(), c.Key)
	if err != nil {
		return nil, err
	}
	if res.Count == 0 {
		return nil, nil
	}
	if res.Count >= 2 {
		return nil, fmt.Errorf("Expected a single result but got %d.", res.Count)
	}

	c.modRevision = res.Kvs[0].ModRevision

	payload := res.Kvs[0].Value
	md5 := md5.Sum(payload)

	return &remote.Payload{
		Data: payload,
		MD5:  md5[:],
	}, nil
}

func (c *RemoteClient) Put(data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	res, err := etcdv3.NewKV(c.Client).Txn(context.TODO()).If(
		etcdv3.Compare(etcdv3.ModRevision(c.Key), "=", c.modRevision),
	).Then(
		etcdv3.OpPut(c.Key, string(data)),
		etcdv3.OpGet(c.Key),
	).Commit()

	if err != nil {
		return err
	}
	if !res.Succeeded {
		return fmt.Errorf("The transaction did not succeed.")
	}
	if len(res.Responses) != 2 {
		return fmt.Errorf("Expected two responses but got %d.", len(res.Responses))
	}

	c.modRevision = res.Responses[1].GetResponseRange().Kvs[0].ModRevision
	return nil
}

func (c *RemoteClient) Delete() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, err := c.Client.KV.Delete(context.TODO(), c.Key)
	return err
}

func (c *RemoteClient) Lock(info *state.LockInfo) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.DoLock {
		return "", nil
	}
	if c.etcdSession != nil {
		return "", fmt.Errorf("state %q already locked", c.Key)
	}

	c.info = info
	return c.lock()
}

func (c *RemoteClient) Unlock(id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.DoLock {
		return nil
	}

	return c.unlock(id)
}

func (c *RemoteClient) deleteLockInfo(info *state.LockInfo) error {
	res, err := c.Client.KV.Delete(context.TODO(), c.Key+lockInfoSuffix)
	if err != nil {
		return err
	}
	if res.Deleted == 0 {
		return fmt.Errorf("No keys deleted for %s when deleting lock info.", c.Key+lockInfoSuffix)
	}
	return nil
}

func (c *RemoteClient) getLockInfo() (*state.LockInfo, error) {
	res, err := c.Client.KV.Get(context.TODO(), c.Key+lockInfoSuffix)
	if err != nil {
		return nil, err
	}
	if res.Count == 0 {
		return nil, nil
	}

	li := &state.LockInfo{}
	err = json.Unmarshal(res.Kvs[0].Value, li)
	if err != nil {
		return nil, fmt.Errorf("Error unmarshaling lock info: %s.", err)
	}

	return li, nil
}

func (c *RemoteClient) putLockInfo(info *state.LockInfo) error {
	c.info.Path = c.etcdMutex.Key()
	c.info.Created = time.Now().UTC()

	_, err := c.Client.KV.Put(context.TODO(), c.Key+lockInfoSuffix, string(c.info.Marshal()))
	return err
}

func (c *RemoteClient) lock() (string, error) {
	session, err := etcdv3sync.NewSession(c.Client)
	if err != nil {
		return "", nil
	}

	ctx, cancel := context.WithTimeout(context.TODO(), lockAcquireTimeout)
	defer cancel()

	mutex := etcdv3sync.NewMutex(session, c.Key)
	if err1 := mutex.Lock(ctx); err1 != nil {
		lockInfo, err2 := c.getLockInfo()
		if err2 != nil {
			return "", &state.LockError{Err: err2}
		}
		return "", &state.LockError{Info: lockInfo, Err: err1}
	}

	c.etcdMutex = mutex
	c.etcdSession = session

	err = c.putLockInfo(c.info)
	if err != nil {
		if unlockErr := c.unlock(c.info.ID); unlockErr != nil {
			err = multierror.Append(err, unlockErr)
		}
		return "", err
	}

	return c.info.ID, nil
}

func (c *RemoteClient) unlock(id string) error {
	if c.etcdMutex == nil {
		return nil
	}

	var errs error

	if err := c.deleteLockInfo(c.info); err != nil {
		errs = multierror.Append(errs, err)
	}
	if err := c.etcdMutex.Unlock(context.TODO()); err != nil {
		errs = multierror.Append(errs, err)
	}
	if err := c.etcdSession.Close(); err != nil {
		errs = multierror.Append(errs, err)
	}

	c.etcdMutex = nil
	c.etcdSession = nil

	return errs
}
