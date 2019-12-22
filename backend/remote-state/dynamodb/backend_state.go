package dynamodb

import (
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/hashicorp/terraform/states"

	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
)

func unique(stringSlice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range stringSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func (b *Backend) Workspaces() ([]string, error) {
	prefix := ""

	if b.workspaceKeyPrefix != "" {
		prefix = b.workspaceKeyPrefix + "="
	} else {
		prefix += "/"
	}

	// Build Query
	filt := expression.Name("StateID").Contains(prefix)
	proj := expression.NamesList(expression.Name("StateID"))
	expr, err := expression.NewBuilder().WithFilter(filt).WithProjection(proj).Build()
	if err != nil {
		return nil, fmt.Errorf("During query build. %s", err)
	}
	dyparams := &dynamodb.ScanInput{
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		FilterExpression:          expr.Filter(),
		ProjectionExpression:      expr.Projection(),
		TableName:                 aws.String(b.tableName),
	}
	// Execute Query
	result, err := b.dynClient.Scan(dyparams)
	if err != nil {
		return nil, fmt.Errorf("During scan operation on table %s %s.", b.tableName, err)
	}
	items := result.Items
	for {
		if result.LastEvaluatedKey == nil {
			break
		}
		dyparams.ExclusiveStartKey = result.LastEvaluatedKey
		result, err = b.dynClient.Scan(dyparams)
		if err != nil {
			return nil, fmt.Errorf("During scan operation on table %s %s.", b.tableName, err)
		}
		for _, i := range result.Items {
			items = append(items, i)
		}

		time.Sleep(consistencyRetryPollInterval)
	}

	// Extract Workspaces
	wss := []string{backend.DefaultStateName}
	for _, i := range items {
		state := State{}

		err = dynamodbattribute.UnmarshalMap(i, &state)
		if err != nil {
			return nil, fmt.Errorf("Error while parsing state : %s", err)
		}

		ws := b.keyEnv(state.StateID)
		if ws != "" {
			wss = append(wss, ws)
		}
	}
	wss = unique(wss)
	sort.Strings(wss[1:])
	return wss, nil
}

func (b *Backend) keyEnv(key string) string {
	prefix := b.workspaceKeyPrefix
	if prefix == "" {
		parts := strings.SplitN(key, "/", 2)
		if len(parts) > 1 && parts[1] == b.hashName {
			return parts[0]
		} else {
			return ""
		}
	}

	// add a = (equal) to to follow convention workspace=<name>
	prefix += "="

	parts := strings.SplitAfterN(key, prefix, 2)
	if len(parts) < 2 {
		return ""
	}

	// shouldn't happen since we listed by prefix
	if parts[0] != prefix {
		return ""
	}

	parts = strings.SplitN(parts[1], "/", 2)

	if len(parts) < 2 {
		return ""
	}

	// not our key, so don't include it in our listing
	if parts[1] != b.hashName {
		return ""
	}
	return parts[0]
}

func (b *Backend) DeleteWorkspace(name string) error {
	if name == backend.DefaultStateName || name == "" {
		return fmt.Errorf("You can't delete default state.")
	}

	client, err := b.remoteClient(name)
	if err != nil {
		return err
	}

	return client.Delete()
}

// get a remote client configured for this state
func (b *Backend) remoteClient(name string) (*RemoteClient, error) {
	if name == "" {
		return nil, errors.New("missing state name")
	}

	client := &RemoteClient{
		dynClient:        b.dynClient,
		dynGlobalClients: b.dynGlobalClients,
		tableName:        b.tableName,
		path:             b.path(name),
		lockTable:        b.lockTable,
		state_days_ttl:   b.state_days_ttl,
	}

	return client, nil
}

func (b *Backend) StateMgr(name string) (state.State, error) {
	client, err := b.remoteClient(name)
	if err != nil {
		return nil, err
	}

	stateMgr := &remote.State{Client: client}
	// Check to see if this state already exists.
	// If we're trying to force-unlock a state, we can't take the lock before
	// fetching the state. If the state doesn't exist, we have to assume this
	// is a normal create operation, and take the lock at that point.
	//
	// If we need to force-unlock, but for some reason the state no longer
	// exists, the user will have to use aws tools to manually fix the
	// situation.
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
		// take a lock on this state while we write it
		lockInfo := state.NewLockInfo()
		lockInfo.Operation = "init"
		lockId, err := client.Lock(lockInfo)
		if err != nil {
			return nil, fmt.Errorf("Failed to lock state: %s", err)
		}

		// Local helper function so we can call it multiple places
		lockUnlock := func(parent error) error {
			if err := stateMgr.Unlock(lockId); err != nil {
				return fmt.Errorf(strings.TrimSpace(errStateUnlock), lockId, err)
			}
			return parent
		}

		// Grab the value
		// This is to ensure that no one beat us to writing a state between
		// the `exists` check and taking the lock.
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

func (b *Backend) client() *RemoteClient {
	return &RemoteClient{}
}

func (b *Backend) path(name string) string {
	if name == backend.DefaultStateName {
		return b.hashName
	}

	if b.workspaceKeyPrefix == "" {
		return path.Join(name, b.hashName)
	} else {
		return path.Join(b.workspaceKeyPrefix+"="+name, b.hashName)
	}

}

const errStateUnlock = `
Error unlocking state. Lock ID: %s

Error: %s

You may have to force-unlock this state in order to use it again.
`
