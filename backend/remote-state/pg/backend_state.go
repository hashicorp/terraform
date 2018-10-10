package pg

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/hashicorp/terraform/terraform"
)

func (b *Backend) States() ([]string, error) {
	query := `SELECT name FROM %s.%s ORDER BY name`
	rows, err := b.db.Query(fmt.Sprintf(query, b.schemaName, statesTableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []string

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		result = append(result, name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (b *Backend) DeleteState(name string) error {
	if name == backend.DefaultStateName || name == "" {
		return fmt.Errorf("can't delete default state")
	}

	query := `DELETE FROM %s.%s WHERE name = $1`
	_, err := b.db.Exec(fmt.Sprintf(query, b.schemaName, statesTableName), name)
	if err != nil {
		return err
	}

	return nil
}

func (b *Backend) State(name string) (state.State, error) {
	// Build the state client
	var stateMgr state.State = &remote.State{
		Client: &RemoteClient{
			Client:     b.db,
			Name:       name,
			SchemaName: b.schemaName,
			lock:       b.lock,
		},
	}

	// If we're not locking, disable it
	if !b.lock {
		stateMgr = &state.LockDisabled{Inner: stateMgr}
	}

	// Check to see if this state already exists.
	// If we're trying to force-unlock a state, we can't take the lock before
	// fetching the state. If the state doesn't exist, we have to assume this
	// is a normal create operation, and take the lock at that point.
	//
	// If we need to force-unlock, but for some reason the state no longer
	// exists, the user will have to use aws tools to manually fix the
	// situation.
	existing, err := b.States()
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

	// Grab a lock, we use this to write an empty state if one doesn't
	// exist already. We have to write an empty state as a sentinel value
	// so States() knows it exists.
	if !exists {
		lockInfo := state.NewLockInfo()
		lockInfo.Operation = "init"
		lockId, err := stateMgr.Lock(lockInfo)
		if err != nil {
			return nil, fmt.Errorf("failed to lock state in Postgres: %s", err)
		}

		// Local helper function so we can call it multiple places
		lockUnlock := func(parent error) error {
			if err := stateMgr.Unlock(lockId); err != nil {
				return fmt.Errorf(strings.TrimSpace(errStateUnlock), lockId, err)
			}

			return parent
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
	}

	return stateMgr, nil
}

const errStateUnlock = `
Error unlocking Postgres state. Lock ID: %s

Error: %s

You may have to force-unlock this state in order to use it again.
The "pg" backend acquires a lock during initialization to ensure
the minimum required key/values are prepared.
`
