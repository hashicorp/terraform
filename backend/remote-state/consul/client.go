package consul

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	consulapi "github.com/hashicorp/consul/api"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
)

const (
	lockSuffix     = "/.lock"
	lockInfoSuffix = "/.lockinfo"

	// The Session TTL associated with this lock.
	lockSessionTTL = "15s"

	// the delay time from when a session is lost to when the
	// lock is released by the server
	lockDelay = 5 * time.Second
	// interval between attempts to reacquire a lost lock
	lockReacquireInterval = 2 * time.Second
)

// RemoteClient is a remote client that stores data in Consul.
type RemoteClient struct {
	Client *consulapi.Client
	Path   string
	GZip   bool

	mu sync.Mutex
	// lockState is true if we're using locks
	lockState bool

	// The index of the last state we wrote.
	// If this is > 0, Put will perform a CAS to ensure that the state wasn't
	// changed during the operation. This is important even with locks, because
	// if the client loses the lock for some reason, then reacquires it, we
	// need to make sure that the state was not modified.
	modifyIndex uint64

	consulLock *consulapi.Lock
	lockCh     <-chan struct{}

	info *state.LockInfo

	// cancel our goroutine which is monitoring the lock to automatically
	// reacquire it when possible.
	monitorCancel context.CancelFunc
	monitorWG     sync.WaitGroup

	// sessionCancel cancels the Context use for session.RenewPeriodic, and is
	// called when unlocking, or before creating a new lock if the lock is
	// lost.
	sessionCancel context.CancelFunc
}

func (c *RemoteClient) Get() (*remote.Payload, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	pair, _, err := c.Client.KV().Get(c.Path, nil)
	if err != nil {
		return nil, err
	}
	if pair == nil {
		return nil, nil
	}

	c.modifyIndex = pair.ModifyIndex

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
	c.mu.Lock()
	defer c.mu.Unlock()

	payload := data
	if c.GZip {
		if compressedState, err := compressState(data); err == nil {
			payload = compressedState
		} else {
			return err
		}
	}

	kv := c.Client.KV()

	// default to doing a CAS
	verb := consulapi.KVCAS

	// Assume a 0 index doesn't need a CAS for now, since we are either
	// creating a new state or purposely overwriting one.
	if c.modifyIndex == 0 {
		verb = consulapi.KVSet
	}

	// KV.Put doesn't return the new index, so we use a single operation
	// transaction to get the new index with a single request.
	txOps := consulapi.KVTxnOps{
		&consulapi.KVTxnOp{
			Verb:  verb,
			Key:   c.Path,
			Value: payload,
			Index: c.modifyIndex,
		},
	}

	ok, resp, _, err := kv.Txn(txOps, nil)
	if err != nil {
		return err
	}

	// transaction was rolled back
	if !ok {
		return fmt.Errorf("consul CAS failed with transaction errors: %v", resp.Errors)
	}

	if len(resp.Results) != 1 {
		// this probably shouldn't happen
		return fmt.Errorf("expected on 1 response value, got: %d", len(resp.Results))
	}

	c.modifyIndex = resp.Results[0].ModifyIndex
	return nil
}

func (c *RemoteClient) Delete() error {
	c.mu.Lock()
	defer c.mu.Unlock()

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
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.lockState {
		return "", nil
	}

	c.info = info

	// These checks only are to ensure we strictly follow the specification.
	// Terraform shouldn't ever re-lock, so provide errors for the 2 possible
	// states if this is called.
	select {
	case <-c.lockCh:
		// We had a lock, but lost it.
		return "", errors.New("lost consul lock, cannot re-lock")
	default:
		if c.lockCh != nil {
			// we have an active lock already
			return "", fmt.Errorf("state %q already locked", c.Path)
		}
	}

	return c.lock()
}

// the lock implementation.
// Only to be called while holding Client.mu
func (c *RemoteClient) lock() (string, error) {
	// We create a new session here, so it can be canceled when the lock is
	// lost or unlocked.
	lockSession, err := c.createSession()
	if err != nil {
		return "", err
	}

	opts := &consulapi.LockOptions{
		Key:     c.Path + lockSuffix,
		Session: lockSession,

		// only wait briefly, so terraform has the choice to fail fast or
		// retry as needed.
		LockWaitTime: time.Second,
		LockTryOnce:  true,

		// Don't let the lock monitor give up right away, as it's possible the
		// session is still OK. While the session is refreshed at a rate of
		// TTL/2, the lock monitor is an idle blocking request and is more
		// susceptible to being closed by a lower network layer.
		MonitorRetries: 5,
		//
		// The delay between lock monitor retries.
		// While the session has a 15s TTL plus a 5s wait period on a lost
		// lock, if we can't get our lock back in 10+ seconds something is
		// wrong so we're going to drop the session and start over.
		MonitorRetryTime: 2 * time.Second,
	}

	c.consulLock, err = c.Client.LockOpts(opts)
	if err != nil {
		return "", err
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

	err = c.putLockInfo(c.info)
	if err != nil {
		if unlockErr := c.unlock(c.info.ID); unlockErr != nil {
			err = multierror.Append(err, unlockErr)
		}

		return "", err
	}

	// Start a goroutine to monitor the lock state.
	// If we lose the lock to due communication issues with the consul agent,
	// attempt to immediately reacquire the lock. Put will verify the integrity
	// of the state by using a CAS operation.
	ctx, cancel := context.WithCancel(context.Background())
	c.monitorCancel = cancel
	c.monitorWG.Add(1)
	go func() {
		defer c.monitorWG.Done()
		select {
		case <-c.lockCh:
			log.Println("[ERROR] lost consul lock")
			for {
				c.mu.Lock()
				// We lost our lock, so we need to cancel the session too.
				// The CancelFunc is only replaced while holding Client.mu, so
				// this is safe to call here. This will be replaced by the
				// lock() call below.
				c.sessionCancel()

				c.consulLock = nil
				_, err := c.lock()
				c.mu.Unlock()

				if err != nil {
					// We failed to get the lock, keep trying as long as
					// terraform is running. There may be changes in progress,
					// so there's no use in aborting. Either we eventually
					// reacquire the lock, or a Put will fail on a CAS.
					log.Printf("[ERROR] could not reacquire lock: %s", err)
					time.Sleep(lockReacquireInterval)

					select {
					case <-ctx.Done():
						return
					default:
					}
					continue
				}

				// if the error was nil, the new lock started a new copy of
				// this goroutine.
				return
			}

		case <-ctx.Done():
			return
		}
	}()

	if testLockHook != nil {
		testLockHook()
	}

	return c.info.ID, nil
}

// called after a lock is acquired
var testLockHook func()

func (c *RemoteClient) createSession() (string, error) {
	// create the context first. Even if the session creation fails, we assume
	// that the CancelFunc is always callable.
	ctx, cancel := context.WithCancel(context.Background())
	c.sessionCancel = cancel

	session := c.Client.Session()
	se := &consulapi.SessionEntry{
		Name:      consulapi.DefaultLockSessionName,
		TTL:       lockSessionTTL,
		LockDelay: lockDelay,
	}

	id, _, err := session.Create(se, nil)
	if err != nil {
		return "", err
	}

	log.Println("[INFO] created consul lock session", id)

	// keep the session renewed
	// we need an adapter to convert the session Done() channel to a
	// non-directional channel to satisfy the RenewPeriodic signature.
	done := make(chan struct{})
	go func() {
		<-ctx.Done()
		close(done)
	}()
	go session.RenewPeriodic(lockSessionTTL, id, nil, done)

	return id, nil
}

func (c *RemoteClient) Unlock(id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.lockState {
		return nil
	}

	return c.unlock(id)
}

// the unlock implementation.
// Only to be called while holding Client.mu
func (c *RemoteClient) unlock(id string) error {
	// this doesn't use the lock id, because the lock is tied to the consul client.
	if c.consulLock == nil || c.lockCh == nil {
		return nil
	}

	// cancel our monitoring goroutine
	c.monitorCancel()

	defer func() {
		c.consulLock = nil

		// The consul session is only used for this single lock, so cancel it
		// after we unlock.
		// The session is only created and replaced holding Client.mu, so the
		// CancelFunc must be non-nil.
		c.sessionCancel()
	}()

	select {
	case <-c.lockCh:
		return errors.New("consul lock was lost")
	default:
	}

	kv := c.Client.KV()

	var errs error

	if _, err := kv.Delete(c.Path+lockInfoSuffix, nil); err != nil {
		errs = multierror.Append(errs, err)
	}

	if err := c.consulLock.Unlock(); err != nil {
		errs = multierror.Append(errs, err)
	}

	// the monitoring goroutine may be in a select on the lockCh, so we need to
	// wait for it to return before changing the value.
	c.monitorWG.Wait()
	c.lockCh = nil

	// This is only cleanup, and will fail if the lock was immediately taken by
	// another client, so we don't report an error to the user here.
	c.consulLock.Destroy()

	return errs
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
