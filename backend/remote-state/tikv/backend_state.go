package tikv

import (
	"context"
	"fmt"
	"github.com/tikv/client-go/key"
	"github.com/tikv/client-go/txnkv"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/states/statemgr"
)

const (
	keyEnvPrefix = "-env:"
)

func (b *Backend) Workspaces() ([]string, error) {
	// List our raw path
	prefix := b.data.Get("prefix").(string) + keyEnvPrefix
	keys, err := getKeys(b.txnKvClient, prefix)
	if err != nil {
		return nil, err
	}

	envs := map[string]struct{}{}
	for _, k := range keys {
		if strings.HasPrefix(k, prefix) {
			k = strings.TrimPrefix(k, prefix)

			if idx := strings.IndexRune(k, '/'); idx >= 0 {
				continue
			}

			envs[k] = struct{}{}
		}
	}

	result := make([]string, 1, len(envs)+1)
	result[0] = backend.DefaultStateName
	for k := range envs {
		result = append(result, k)
	}

	return result, nil
}

func (b *Backend) DeleteWorkspace(name string) error {
	if name == backend.DefaultStateName || name == "" {
		return fmt.Errorf("can't delete default state")
	}

	// Determine the path of the data
	path := b.path(name)

	// Delete it. We just delete it without any locking since
	// the DeleteState API is documented as such.
	tx, err := b.txnKvClient.Begin(context.TODO())
	if err != nil {
		return err
	}
	err = tx.Delete([]byte(path))
	if e := tx.Commit(context.TODO()); e != nil {
		err = multierror.Append(err, e)
	}
	return err
}

func (b *Backend) StateMgr(name string) (statemgr.Full, error) {
	// Determine the path of the data
	path := b.path(name)

	// Build the state client
	var stateMgr = &remote.State{
		Client: &RemoteClient{
			rawKvClient: b.rawKvClient,
			txnKvClient: b.txnKvClient,
			Key:         path,
			DoLock:      b.lock,
		},
	}

	if !b.lock {
		stateMgr.DisableLocks()
	}

	// the default state always exists
	if name == backend.DefaultStateName {
		return stateMgr, nil
	}

	// Grab a lock, we use this to write an empty state if one doesn't
	// exist already. We have to write an empty state as a sentinel value
	// so States() knows it exists.
	lockInfo := state.NewLockInfo()
	lockInfo.Operation = "init"
	lockId, err := stateMgr.Lock(lockInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to lock state in Consul: %s", err)
	}

	// Local helper function so we can call it multiple places
	lockUnlock := func(parent error) error {
		if err := stateMgr.Unlock(lockId); err != nil {
			return fmt.Errorf(strings.TrimSpace(errStateUnlock), lockId, err)
		}

		return parent
	}

	// Grab the value
	if err := stateMgr.RefreshState(); err != nil {
		err = lockUnlock(err)
		return nil, err
	}

	// If we have no state, we have to create an empty state
	if v := stateMgr.State(); v == nil {
		if err := stateMgr.WriteState(states.NewState()); err != nil {
			err = lockUnlock(err)
			return nil, err
		}
		if err := stateMgr.PersistState(); err != nil {
			err = lockUnlock(err)
			return nil, err
		}
	}

	// Unlock, the state should now be initialized
	if err := lockUnlock(nil); err != nil {
		return nil, err
	}

	return stateMgr, nil
}

func getKeys(txnKvClient *txnkv.Client, prefix string) ([]string, error) {
	ctx := context.TODO()
	txn, err := txnKvClient.Begin(ctx)
	if err != nil {
		return nil, err
	}

	it, err := txn.Iter(ctx, key.Key(prefix), nil)
	if err != nil {
		return nil, err
	}

	var keys []string
	prefixKey := key.Key(prefix)

	for it.Valid() {
		if !it.Key().HasPrefix(prefixKey) {
			break
		}

		keys = append(keys, string(it.Key()))

		err = it.Next(ctx)
		if err != nil {
			return nil, err
		}
	}

	return keys, nil
}

func (b *Backend) path(name string) string {
	path := b.data.Get("prefix").(string)
	if name != backend.DefaultStateName {
		path += fmt.Sprintf("%s%s", keyEnvPrefix, name)
	}

	return path
}

const errStateUnlock = `
Error unlocking TiKV state. Lock ID: %s

Error: %s

You may have to force-unlock this state in order to use it again.
The TiKV backend acquires a lock during initialization to ensure
the minimum required key/values are prepared.
`
