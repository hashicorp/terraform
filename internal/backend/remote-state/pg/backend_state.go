package pg

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/remote"
	"github.com/hashicorp/terraform/internal/states/statemgr"
)

func (b *Backend) Workspaces() ([]string, error) {
	query := `SELECT name FROM %s.%s WHERE name != 'default' ORDER BY name`
	rows, err := b.db.Query(fmt.Sprintf(query, b.schemaName, statesTableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []string{
		backend.DefaultStateName,
	}

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

func (b *Backend) DeleteWorkspace(name string) error {
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

func (b *Backend) StateMgr(name string) (statemgr.Full, error) {
	// Build the state client
	var stateMgr statemgr.Full = &remote.State{
		Client: &RemoteClient{
			Client:     b.db,
			Name:       name,
			SchemaName: b.schemaName,
		},
	}

	// Check to see if this state already exists.
	// If the state doesn't exist, we have to assume this
	// is a normal create operation, and take the lock at that point.
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

	// Grab a lock, we use this to write an empty state if one doesn't
	// exist already. We have to write an empty state as a sentinel value
	// so Workspaces() knows it exists.
	if !exists {
		lockInfo := statemgr.NewLockInfo()
		lockInfo.Operation = "init"
		lockId, err := stateMgr.Lock(lockInfo)
		if err != nil {
			return nil, fmt.Errorf("failed to lock state in Postgres: %s", err)
		}

		// Local helper function so we can call it multiple places
		lockUnlock := func(parent error) error {
			if err := stateMgr.Unlock(lockId); err != nil {
				return fmt.Errorf(`error unlocking Postgres state: %s`, err)
			}
			return parent
		}

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
