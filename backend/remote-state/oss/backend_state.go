package oss

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/hashicorp/terraform/states"

	"github.com/aliyun/aliyun-tablestore-go-sdk/tablestore"
	"log"
	"path"
)

const (
	lockFileSuffix = ".tflock"
)

// get a remote client configured for this state
func (b *Backend) remoteClient(name string) (*RemoteClient, error) {
	if name == "" {
		return nil, errors.New("missing state name")
	}

	client := &RemoteClient{
		ossClient:            b.ossClient,
		bucketName:           b.bucketName,
		stateFile:            b.stateFile(name),
		lockFile:             b.lockFile(name),
		serverSideEncryption: b.serverSideEncryption,
		acl:                  b.acl,
		otsTable:             b.otsTable,
		otsClient:            b.otsClient,
	}
	if b.otsEndpoint != "" && b.otsTable != "" {
		table, err := b.otsClient.DescribeTable(&tablestore.DescribeTableRequest{
			TableName: b.otsTable,
		})
		if err != nil {
			return client, fmt.Errorf("Error describing table store %s: %#v", b.otsTable, err)
		}
		for _, t := range table.TableMeta.SchemaEntry {
			pkMeta := TableStorePrimaryKeyMeta{
				PKName: *t.Name,
			}
			if *t.Type == tablestore.PrimaryKeyType_INTEGER {
				pkMeta.PKType = "Integer"
			} else if *t.Type == tablestore.PrimaryKeyType_STRING {
				pkMeta.PKType = "String"
			} else if *t.Type == tablestore.PrimaryKeyType_BINARY {
				pkMeta.PKType = "Binary"
			} else {
				return client, fmt.Errorf("Unsupported PrimaryKey type: %d.", *t.Type)
			}
			client.otsTabkePK = pkMeta
			break
		}
	}

	return client, nil
}

func (b *Backend) Workspaces() ([]string, error) {
	bucket, err := b.ossClient.Bucket(b.bucketName)
	if err != nil {
		return []string{""}, fmt.Errorf("Error getting bucket: %#v", err)
	}

	var options []oss.Option
	options = append(options, oss.Prefix(b.statePrefix+"/"))
	resp, err := bucket.ListObjects(options...)

	if err != nil {
		return nil, err
	}

	result := []string{backend.DefaultStateName}
	prefix := b.statePrefix
	for _, obj := range resp.Objects {
		// we have 3 parts, the state prefix, the workspace name, and the state file: <prefix>/<worksapce-name>/<key>
		if path.Join(b.statePrefix, b.stateKey) == obj.Key {
			// filter the default workspace
			continue
		}

		parts := strings.Split(strings.TrimPrefix(obj.Key, prefix+"/"), "/")
		if len(parts) > 0 && parts[0] != "" {
			result = append(result, parts[0])
		}
	}

	sort.Strings(result[1:])
	return result, nil
}

func (b *Backend) DeleteWorkspace(name string) error {
	if name == backend.DefaultStateName || name == "" {
		return fmt.Errorf("can't delete default state")
	}

	client, err := b.remoteClient(name)
	if err != nil {
		return err
	}
	return client.Delete()
}

func (b *Backend) StateMgr(name string) (state.State, error) {
	client, err := b.remoteClient(name)
	if err != nil {
		return nil, err
	}
	stateMgr := &remote.State{Client: client}

	// Check to see if this state already exists.
	existing, err := b.Workspaces()
	if err != nil {
		return nil, err
	}

	log.Printf("[DEBUG] Current workspace name: %s. All workspaces:%#v", name, existing)

	exists := false
	for _, s := range existing {
		if s == name {
			exists = true
			break
		}
	}
	// We need to create the object so it's listed by States.
	if !exists {
		// take a lock on this state while we write it
		lockInfo := state.NewLockInfo()
		lockInfo.Operation = "init"
		lockId, err := client.Lock(lockInfo)
		if err != nil {
			return nil, fmt.Errorf("Failed to lock OSS state: %s", err)
		}

		// Local helper function so we can call it multiple places
		lockUnlock := func(e error) error {
			if err := stateMgr.Unlock(lockId); err != nil {
				return fmt.Errorf(strings.TrimSpace(stateUnlockError), lockId, err)
			}
			return e
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

	}
	return stateMgr, nil
}

func (b *Backend) stateFile(name string) string {
	if name == backend.DefaultStateName {
		return path.Join(b.statePrefix, b.stateKey)
	}
	return path.Join(b.statePrefix, name, b.stateKey)
}

func (b *Backend) lockFile(name string) string {
	return b.stateFile(name) + lockFileSuffix
}

const stateUnlockError = `
Error unlocking Alibaba Cloud OSS state file:

Lock ID: %s
Error message: %#v

You may have to force-unlock this state in order to use it again.
The Alibaba Cloud backend acquires a lock during initialization to ensure the initial state file is created.
`
