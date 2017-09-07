package gcloud

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/hashicorp/terraform/terraform"
	"google.golang.org/api/iterator"
)

func (b *Backend) States() ([]string, error) {
	workspaces := []string{backend.DefaultStateName}
	var stateRegex *regexp.Regexp
	var err error
	if b.stateDir == "" {
		stateRegex = regexp.MustCompile(`^(.+)\.tfstate$`)
	} else {
		stateRegex, err = regexp.Compile(fmt.Sprintf("^%v/(.+)\\.tfstate$", regexp.QuoteMeta(b.stateDir)))
		if err != nil {
			return []string{}, fmt.Errorf("Failed to compile regex for querying states: %v", err)
		}
	}

	bucket := b.storageClient.Bucket(b.bucketName)
	query := &storage.Query{
		Prefix: b.stateDir,
	}

	files := bucket.Objects(b.storageContext, query)
	for {
		attrs, err := files.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return []string{}, fmt.Errorf("Failed to query remote states: %v", err)
		}

		matches := stateRegex.FindStringSubmatch(attrs.Name)
		if len(matches) == 2 && matches[1] != backend.DefaultStateName {
			workspaces = append(workspaces, matches[1])
		}
	}

	sort.Strings(workspaces[1:])
	return workspaces, nil
}

func (b *Backend) DeleteState(name string) error {
	if name == backend.DefaultStateName || name == "" {
		return fmt.Errorf("Can't delete default state")
	}

	client, err := b.remoteClient(name)
	if err != nil {
		return fmt.Errorf("Failed to create Google Storage client: %v", err)
	}

	err = client.Delete()
	if err != nil {
		return fmt.Errorf("Failed to delete state file %v: %v", client.stateFileURL(), err)
	}

	return nil
}

// get a remote client configured for this state
func (b *Backend) remoteClient(name string) (*RemoteClient, error) {
	if name == "" {
		return nil, errors.New("Missing state name")
	}

	client := &RemoteClient{
		storageContext: b.storageContext,
		storageClient:  b.storageClient,
		bucketName:     b.bucketName,
		stateFilePath:  b.stateFileName(name),
		lockFilePath:   b.lockFileName(name),
	}

	return client, nil
}

func (b *Backend) State(name string) (state.State, error) {
	client, err := b.remoteClient(name)
	if err != nil {
		return nil, fmt.Errorf("Failed to create Google Storage client: %v", err)
	}

	stateMgr := &remote.State{Client: client}
	lockInfo := state.NewLockInfo()
	lockInfo.Operation = "init"
	lockId, err := stateMgr.Lock(lockInfo)
	if err != nil {
		return nil, fmt.Errorf("Failed to lock state in Google Cloud Storage: %v", err)
	}

	// Local helper function so we can call it multiple places
	lockUnlock := func(parent error) error {
		if err := stateMgr.Unlock(lockId); err != nil {
			return fmt.Errorf(strings.TrimSpace(errStateUnlock), lockId, client.lockFileURL(), err)
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
		if err := stateMgr.WriteState(terraform.NewState()); err != nil {
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

func (b *Backend) stateFileName(stateName string) string {
	if b.stateDir == "" {
		return fmt.Sprintf("%v.tfstate", stateName)
	} else {
		return fmt.Sprintf("%v/%v.tfstate", b.stateDir, stateName)
	}
}

func (b *Backend) lockFileName(stateName string) string {
	if b.stateDir == "" {
		return fmt.Sprintf("%v.tflock", stateName)
	} else {
		return fmt.Sprintf("%v/%v.tflock", b.stateDir, stateName)
	}
}

const errStateUnlock = `
Error unlocking Google Cloud Storage state.

Lock ID: %v
Lock file URL: %v
Error: %v

You may have to force-unlock this state in order to use it again.
The GCloud backend acquires a lock during initialization to ensure
the initial state file is created.
`
