package obs

import (
	"errors"
	"fmt"
	"log"
	"path"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/hashicorp/terraform/states"
	"github.com/huaweicloud/golangsdk/openstack/obs"
)

// Define file suffix
const (
	stateFileSuffix = ".tfstate"
	lockFileSuffix  = ".tflock"
)

// Workspaces returns a list of names for the workspaces
func (b *Backend) Workspaces() ([]string, error) {
	const maxKeys = 1000

	params := &obs.ListObjectsInput{}
	params.Bucket = b.bucketName
	params.Prefix = b.prefix
	params.MaxKeys = maxKeys

	Objects, err := b.obsClient.ListObjects(params)
	if obsErr, ok := err.(obs.ObsError); ok {
		log.Printf("[ERROR] failed to list obs objects: %s", err)
		if obsErr.Code == ErrCodeNoSuchBucket {
			return nil, fmt.Errorf(errNoSuchBucket, err)
		}
		return nil, err
	}

	// Grab the Contents
	wss := []string{backend.DefaultStateName}
	for _, obj := range Objects.Contents {
		// skip <name>.tfstate
		if !strings.HasSuffix(obj.Key, stateFileSuffix) {
			continue
		}
		// skip default worksapce
		if path.Join(b.prefix, b.keyName) == obj.Key {
			continue
		}
		// <prefix>/<worksapce>/<key>
		prefix := strings.TrimRight(b.prefix, "/") + "/"
		parts := strings.Split(strings.TrimPrefix(obj.Key, prefix), "/")
		if len(parts) > 0 && parts[0] != "" {
			wss = append(wss, parts[0])
		}
	}

	sort.Strings(wss[1:])
	log.Printf("[DEBUG] workspaces in bucket %s: %v", b.bucketName, wss)

	return wss, nil
}

// DeleteWorkspace deletes the named workspaces. The "default" state cannot be deleted
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

// StateMgr manage the state, if the named state not exists, a new file will created
func (b *Backend) StateMgr(name string) (state.State, error) {
	client, err := b.remoteClient(name)
	if err != nil {
		return nil, err
	}

	stateMgr := &remote.State{Client: client}
	// Check to see if this state already exists.
	// If the state doesn't exist, we have to assume this is a normal create operation.
	existing, err := b.Workspaces()
	if err != nil {
		return nil, err
	}

	exists := false
	for _, s := range existing {
		if s == name {
			exists = true
			break
		}
	}

	// We need to create the object so it's listed by States.
	if !exists {
		log.Printf("[DEBUG] object %s does not exist in the workspace", name)
		// Grab the value
		if err := stateMgr.RefreshState(); err != nil {
			log.Printf("[ERROR] failed to refresh the state file: %s", err)
			return nil, err
		}

		// If we have no state, we have to create an empty state
		if v := stateMgr.State(); v == nil {
			if err := stateMgr.WriteState(states.NewState()); err != nil {
				log.Printf("[ERROR] failed to write the state file: %s", err)
				return nil, err
			}
			if err := stateMgr.PersistState(); err != nil {
				log.Printf("[ERROR] failed to persist the state file: %s", err)
				return nil, err
			}
		}
	}

	return stateMgr, nil
}

// get a remote client configured for this state
func (b *Backend) remoteClient(name string) (*RemoteClient, error) {
	if name == "" {
		return nil, errors.New("missing state name")
	}

	client := &RemoteClient{
		obsClient:  b.obsClient,
		bucketName: b.bucketName,
		stateFile:  b.statePath(name),
		lockFile:   b.lockPath(name),
		acl:        b.acl,
		encryption: b.encryption,
		kmsKeyID:   b.kmsKeyID,
	}

	return client, nil
}

// statePath returns state file path by name
func (b *Backend) statePath(name string) string {
	if name == backend.DefaultStateName {
		return path.Join(b.prefix, b.keyName)
	}

	return path.Join(b.prefix, name, b.keyName)
}

// lockPath returns lock file path by name
func (b *Backend) lockPath(name string) string {
	return b.statePath(name) + lockFileSuffix
}
