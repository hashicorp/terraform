package datastore

import (
	"bytes"
	"compress/zlib"
	"context"
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"

	"cloud.google.com/go/datastore"
)

const (
	lockAcquireTimeout = 2 * time.Second

	kindTerraformState     = "TerraformState"
	kindTerraformStateLock = "TerraformStateLock"
)

type entityState struct {
	Data []byte `datastore:",noindex"`
}

// Corresponds to state.LockInfo
type entityLock struct {
	// Provided by the caller.
	ID        string
	Operation string
	Info      string `datastore:",noindex"`
	Who       string
	Version   string
	Created   time.Time

	// Unused. Exists so we can type convert to and from from state.LockInfo.
	Path string `datastore:",noindex,omitempty"`
}

// RemoteClient is a remote client that stores data in Google Datastore.
type RemoteClient struct {
	ds      *datastore.Client
	key     *datastore.Key
	lockKey *datastore.Key
}

func newRemoteClient(c *datastore.Client, namespace, workspace string) *RemoteClient {
	k := &datastore.Key{Kind: kindTerraformState, Namespace: namespace, Name: workspace}
	lk := &datastore.Key{Kind: kindTerraformStateLock, Namespace: namespace, Name: workspace, Parent: k}
	return &RemoteClient{ds: c, key: k, lockKey: lk}
}

// Get state from Datastore.
func (c *RemoteClient) Get() (*remote.Payload, error) {
	e := &entityState{}
	if err := c.ds.Get(context.TODO(), c.key, e); err != nil {
		// Remote clients are expected to return a nil payload and error when
		// asked to get a non-existent state.
		if err == datastore.ErrNoSuchEntity {
			return nil, nil
		}
		return nil, fmt.Errorf("cannot get Google Datastore key %s: %v", c.key, err)
	}
	d, err := decompress(e.Data)
	if err != nil {
		return nil, fmt.Errorf("cannot decompress zlib compressed state from Google Datastore: %v", err)
	}
	h := md5.Sum(d)
	return &remote.Payload{Data: d, MD5: h[:]}, nil
}

func decompress(p []byte) ([]byte, error) {
	z, err := zlib.NewReader(bytes.NewReader(p))
	if err != nil {
		return nil, err
	}
	defer z.Close()
	return ioutil.ReadAll(z)
}

// Put state in Datastore.
func (c *RemoteClient) Put(p []byte) error {
	// We compress state in order to avoid Datastore's 1MB blob size limit.
	if _, err := c.ds.Put(context.TODO(), c.key, &entityState{Data: compress(p)}); err != nil {
		return fmt.Errorf("cannot put Google Datastore key %s: %v", c.key, err)
	}
	return nil
}

func compress(p []byte) []byte {
	b := &bytes.Buffer{}
	z := zlib.NewWriter(b)
	z.Write(p)
	z.Close()
	return b.Bytes()
}

// Delete state from Datatstore.
func (c *RemoteClient) Delete() error {
	if err := c.ds.Delete(context.TODO(), c.key); err != nil {
		return fmt.Errorf("cannot delete Google Datastore key %s: %v", c.key, err)
	}
	return nil
}

// Lock state in Datastore.
func (c *RemoteClient) Lock(info *state.LockInfo) (string, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), lockAcquireTimeout)
	defer cancel()

	_, err := c.ds.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
		existing := &entityLock{}
		if err := tx.Get(c.lockKey, existing); err != datastore.ErrNoSuchEntity {
			if err != nil {
				return fmt.Errorf("cannot determine lock state of Google Datastore key %v: %v", c.lockKey, err)
			}
			return &state.LockError{
				Err:  fmt.Errorf("lock already taken on Google Datastore key %v", c.lockKey),
				Info: (*state.LockInfo)(existing),
			}
		}
		if _, err := tx.Put(c.lockKey, (*entityLock)(info)); err != nil {
			return fmt.Errorf("cannot take lock on Google Datastore key %v: %v", c.lockKey, err)
		}
		return nil
	})

	return info.ID, err
}

// Unlock state in Datastore.
func (c *RemoteClient) Unlock(id string) error {
	ctx, cancel := context.WithTimeout(context.TODO(), lockAcquireTimeout)
	defer cancel()

	_, err := c.ds.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
		existing := &entityLock{}
		if err := tx.Get(c.lockKey, existing); err != nil {
			return fmt.Errorf("cannot determine lock state of Google Datastore key %v: %v", c.lockKey, err)
		}
		if existing.ID != id {
			return &state.LockError{
				Err:  fmt.Errorf("lock taken by another party on Google Datastore key %v", c.lockKey),
				Info: (*state.LockInfo)(existing),
			}
		}
		if err := tx.Delete(c.lockKey); err != nil {
			return fmt.Errorf("cannot relinquish lock of Google Datastore key %v: %v", c.lockKey, err)
		}
		return nil
	})
	return err
}
