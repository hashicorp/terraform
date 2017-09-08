package etcd

import (
	"context"
	"fmt"
	"sort"
	"strings"

	etcdv3 "github.com/coreos/etcd/clientv3"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/hashicorp/terraform/terraform"
)

const (
	keyEnvPrefix = "-env:"
)

func (b *Backend) States() ([]string, error) {
	client, err := b.rawClient()
	if err != nil {
		return nil, err
	}

	prefix := b.determineKey("")
	res, err := client.Get(context.TODO(), prefix, etcdv3.WithPrefix(), etcdv3.WithKeysOnly())
	if err != nil {
		return nil, err
	}

	result := make([]string, 1, len(res.Kvs)+1)
	result[0] = backend.DefaultStateName
	for _, kv := range res.Kvs {
		result = append(result, strings.TrimPrefix(string(kv.Key), prefix))
	}
	sort.Strings(result[1:])

	return result, nil
}

func (b *Backend) DeleteState(name string) error {
	if name == backend.DefaultStateName || name == "" {
		return fmt.Errorf("Can't delete default state.")
	}

	client, err := b.rawClient()
	if err != nil {
		return err
	}

	path := b.determineKey(name)

	_, err = client.Delete(context.TODO(), path)
	return err
}

func (b *Backend) State(name string) (state.State, error) {
	client, err := b.rawClient()
	if err != nil {
		return nil, err
	}

	var stateMgr state.State = &remote.State{
		Client: &RemoteClient{
			Client: client,
			DoLock: b.lock,
			Key:    b.determineKey(name),
		},
	}

	if !b.lock {
		stateMgr = &state.LockDisabled{Inner: stateMgr}
	}

	lockInfo := state.NewLockInfo()
	lockInfo.Operation = "init"
	lockId, err := stateMgr.Lock(lockInfo)
	if err != nil {
		return nil, fmt.Errorf("Failed to lock state in etcd: %s.", err)
	}

	lockUnlock := func(parent error) error {
		if err := stateMgr.Unlock(lockId); err != nil {
			return fmt.Errorf(strings.TrimSpace(errStateUnlock), lockId, err)
		}
		return parent
	}

	if err := stateMgr.RefreshState(); err != nil {
		err = lockUnlock(err)
		return nil, err
	}

	if v := stateMgr.State(); v == nil {
		if err := stateMgr.WriteState(terraform.NewState()); err != nil {
			err = lockUnlock(err)
			return nil, err
		}
		if err := stateMgr.PersistState(); err != nil {
			err = lockUnlock(err)
			return nil, err
		}
	}

	if err := lockUnlock(nil); err != nil {
		return nil, err
	}

	return stateMgr, nil
}

func (b *Backend) determineKey(name string) string {
	prefix := b.prefix
	if name != backend.DefaultStateName {
		prefix += fmt.Sprintf("%s%s", keyEnvPrefix, name)
	}
	return prefix
}

const errStateUnlock = `
Error unlocking etcd state. Lock ID: %s

Error: %s

You may have to force-unlock this state in order to use it again.
`
