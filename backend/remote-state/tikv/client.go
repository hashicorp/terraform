package tikv

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hashicorp/go-multierror"
	"github.com/tikv/client-go/rawkv"
	"sync"
	"time"

	_ "github.com/tikv/client-go/config"
	"github.com/tikv/client-go/txnkv"
	"github.com/tikv/client-go/txnkv/kv"

	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
)

const (
	lockAcquireTimeout = 2 * time.Second
	lockInfoSuffix     = ".lockinfo"
)

// RemoteClient is a remote client that will store data in etcd.
type RemoteClient struct {
	DoLock bool
	Key    string

	rawKvClient *rawkv.Client
	txnKvClient *txnkv.Client
	info        *state.LockInfo
	mu          sync.Mutex
}

func (c *RemoteClient) Get() (*remote.Payload, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	tx, err := c.txnKvClient.Begin(context.TODO())
	if err != nil {
		return nil, err
	}
	defer func() {
		tx.Commit(context.TODO())
	}()
	res, err := tx.Get(context.TODO(), []byte(c.Key))
	if err != nil {
		if kv.IsErrNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	if res == nil {
		return nil, nil
	}

	payload := res
	md5Sum := md5.Sum(payload)

	return &remote.Payload{
		Data: payload,
		MD5:  md5Sum[:],
	}, nil
}

func (c *RemoteClient) Put(data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	tx, err := c.txnKvClient.Begin(context.TODO())
	if err != nil {
		return err
	}
	err = tx.Set([]byte(c.Key), []byte(data))
	if e := tx.Commit(context.TODO()); e != nil {
		err = multierror.Append(err, e)
	}
	return err
}

func (c *RemoteClient) Delete() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	tx, err := c.txnKvClient.Begin(context.TODO())
	if err != nil {
		return err
	}
	err = tx.Delete([]byte(c.Key))
	if e := tx.Commit(context.TODO()); e != nil {
		err = multierror.Append(err, e)
	}
	return err
}

func (c *RemoteClient) Lock(info *state.LockInfo) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.DoLock {
		return "", nil
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
	tx, err := c.txnKvClient.Begin(context.TODO())
	if err != nil {
		return &state.LockError{Err: err}
	}
	err = tx.Delete([]byte(c.Key + lockInfoSuffix))
	if e := tx.Commit(context.TODO()); e != nil {
		err = multierror.Append(err, e)
	}
	if err != nil {
		return &state.LockError{Err: err}
	}
	return nil
}

func (c *RemoteClient) getLockInfo(tx *txnkv.Transaction) (*state.LockInfo, error) {
	res, err := tx.Get(context.TODO(), []byte(c.Key+lockInfoSuffix))
	if err != nil {
		if kv.IsErrNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	li := &state.LockInfo{}
	err = json.Unmarshal(res, li)
	if err != nil {
		return nil, fmt.Errorf("Error unmarshaling lock info: %s.", err)
	}

	return li, nil
}

func (c *RemoteClient) putLockInfo(tx *txnkv.Transaction, info *state.LockInfo) error {
	//c.info.Path = c.etcdMutex.Key()
	c.info.Created = time.Now().UTC()
	err := tx.Set([]byte(c.Key+lockInfoSuffix), []byte(string(c.info.Marshal())))
	return err
}

func (c *RemoteClient) lock() (string, error) {
	tx, err := c.txnKvClient.Begin(context.TODO())
	if err != nil {
		return "", &state.LockError{Err: err}
	}

	resp, err := tx.Get(context.TODO(), []byte(c.Key+lockInfoSuffix))
	if err != nil && !kv.IsErrNotFound(err) {
		return "", &state.LockError{Err: err}
	}
	if resp != nil {
		lockInfo, err := c.getLockInfo(tx)
		if err == nil {
			err = errors.New("lock is conflict")
		}
		if e := tx.Commit(context.TODO()); e != nil {
			err = multierror.Append(err, e)
		}
		return "", &state.LockError{Info: lockInfo, Err: err}
	}
	err = c.putLockInfo(tx, c.info)
	if e := tx.Commit(context.TODO()); e != nil {
		err = multierror.Append(err, e)
	}
	if err != nil {
		return "", &state.LockError{Info: c.info, Err: err}
	}

	return c.info.ID, nil
}

func (c *RemoteClient) unlock(id string) error {
	return c.deleteLockInfo(c.info)
}
