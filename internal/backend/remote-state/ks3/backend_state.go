package ks3

import (
	"fmt"
	"log"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/remote"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/wilac-pv/ksyun-ks3-go-sdk/ks3"
)

// Define file suffix
const (
	stateFileSuffix = ".tfstate"
	lockFileSuffix  = ".tflock"
)

// Workspaces returns a list of names for the workspaces
func (b *Backend) Workspaces() ([]string, error) {
	c, err := b.client("ksyun")
	if err != nil {
		return nil, err
	}

	obs, err := c.listObjects(b.workspaceKeyPrefix)
	log.Printf("[DEBUG] list all workspaces, objects: %v, error: %v", obs, err)
	if err != nil {
		return nil, err
	}

	ws := []string{backend.DefaultStateName}
	for _, vv := range obs {
		// <name>.tfstate
		if !strings.HasSuffix(vv.Key, stateFileSuffix) {
			continue
		}
		// default worksapce
		if path.Join(b.workspaceKeyPrefix, b.key) == vv.Key {
			continue
		}

		// deal with <prefix>/<workspace>/<name>.state
		if space := b.getWorkspace(vv); space != "" {
			ws = append(ws, space)
		}
	}

	sort.Strings(ws[1:])
	log.Printf("[DEBUG] list all workspaces, workspaces: %v", ws)

	return ws, nil
}

// DeleteWorkspace deletes the named workspaces. The "default" state cannot be deleted.
func (b *Backend) DeleteWorkspace(name string, _ bool) error {
	log.Printf("[DEBUG] delete workspace, workspace: %v", name)

	if name == backend.DefaultStateName || name == "" {
		return fmt.Errorf("default state is not allow to delete")
	}

	c, err := b.client(name)
	if err != nil {
		return err
	}

	return c.Delete()
}

// StateMgr manage the state, if the named state not exists, a new file will created
func (b *Backend) StateMgr(name string) (statemgr.Full, error) {
	log.Printf("[DEBUG] state manager, current workspace: %v", name)

	c, err := b.client(name)
	if err != nil {
		return nil, err
	}
	stateMgr := &remote.State{Client: c}

	ws, err := b.Workspaces()
	if err != nil {
		return nil, err
	}

	exists := false
	for _, candidate := range ws {
		if candidate == name {
			exists = true
			break
		}
	}

	if !exists {
		log.Printf("[DEBUG] workspace %v not exists", name)

		// take a lock on this state while we write it
		lockInfo := statemgr.NewLockInfo()
		lockInfo.Operation = "init"
		lockId, err := c.Lock(lockInfo)
		if err != nil {
			return nil, fmt.Errorf("Failed to lock cos state: %s", err)
		}

		// Local helper function so we can call it multiple places
		lockUnlock := func(e error) error {
			if err := stateMgr.Unlock(lockId); err != nil {
				return fmt.Errorf(unlockErrMsg, err, lockId)
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
			if err := stateMgr.PersistState(nil); err != nil {
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

// client returns a remoteClient for the named state.
func (b *Backend) client(name string) (*remoteClient, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("state name not allow to be empty")
	}
	bucketClass, err := b.ks3Client.Bucket(b.bucket)
	if err != nil {
		return nil, err
	}
	ld, err := lockDurationParse(b.lockDuration)
	if err != nil {
		return nil, err
	}
	return &remoteClient{
		ks3Context:   b.ks3Context,
		ks3Client:    b.ks3Client,
		tagClient:    b.tagClient,
		bucketName:   b.bucket,
		bucket:       bucketClass,
		stateFile:    b.stateFile(name),
		lockFile:     b.lockFile(name),
		encrypt:      b.encrypt,
		acl:          b.acl,
		lockDuration: ld,
	}, nil
}

func (b *Backend) getWorkspace(object ks3.ObjectProperties) string {
	// <prefix>/<worksapce>/<key>
	prefix := strings.TrimRight(b.workspaceKeyPrefix, "/") + "/"
	parts := strings.Split(strings.TrimPrefix(object.Key, prefix), "/")
	if len(parts) > 0 && parts[0] != "" {
		return parts[0]
	}
	return ""
}

// stateFile returns state file path by name
func (b *Backend) stateFile(name string) string {
	if name == backend.DefaultStateName {
		return path.Join(b.workspaceKeyPrefix, b.key)
	}
	return path.Join(b.workspaceKeyPrefix, name, b.key)
}

// lockFile returns lock file path by name
func (b *Backend) lockFile(name string) string {
	return b.stateFile(name) + lockFileSuffix
}

func lockDurationParse(d string) (time.Duration, error) {
	switch d {
	case "0":
		return 0, nil
	case "-1":
		return 9999 * time.Hour, nil
	default:
		suffix := d[len(d)-1:]
		switch suffix {
		case "h":
			length := d[:len(d)-1]
			t, err := strconv.Atoi(length)
			if err != nil {
				return -1, err
			}
			return time.Duration(t) * time.Hour, nil
		case "m":
			length := d[:len(d)-1]
			t, err := strconv.Atoi(length)
			if err != nil {
				return -1, err
			}
			return time.Duration(t) * time.Minute, nil
		}
	}

	return 0, fmt.Errorf("lock time parse unexpected error")
}

// unlockErrMsg is error msg for unlock failed
const unlockErrMsg = `
Unlocking the state file on Ksyun ks3 backend failed:

Error message: %v
Lock ID (gen): %s

You may have to force-unlock this state in order to use it again.
The Ksyun backend acquires a lock during initialization
to ensure the initial state file is created.
`
