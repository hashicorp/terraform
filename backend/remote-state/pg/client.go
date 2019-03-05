package pg

import (
	"crypto/md5"
	"database/sql"
	"fmt"

	uuid "github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
	_ "github.com/lib/pq"
)

// RemoteClient is a remote client that stores data in a Postgres database
type RemoteClient struct {
	Client     *sql.DB
	Name       string
	SchemaName string

	info *state.LockInfo
}

func (c *RemoteClient) Get() (*remote.Payload, error) {
	query := `SELECT data FROM %s.%s WHERE name = $1`
	row := c.Client.QueryRow(fmt.Sprintf(query, c.SchemaName, statesTableName), c.Name)
	var data []byte
	err := row.Scan(&data)
	switch {
	case err == sql.ErrNoRows:
		// No existing state returns empty.
		return nil, nil
	case err != nil:
		return nil, err
	default:
		md5 := md5.Sum(data)
		return &remote.Payload{
			Data: data,
			MD5:  md5[:],
		}, nil
	}
}

func (c *RemoteClient) Put(data []byte) error {
	query := `INSERT INTO %s.%s (name, data) VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE
		SET data = $2 WHERE %s.name = $1`
	_, err := c.Client.Exec(fmt.Sprintf(query, c.SchemaName, statesTableName, statesTableName), c.Name, data)
	if err != nil {
		return err
	}
	return nil
}

func (c *RemoteClient) Delete() error {
	query := `DELETE FROM %s.%s WHERE name = $1`
	_, err := c.Client.Exec(fmt.Sprintf(query, c.SchemaName, statesTableName), c.Name)
	if err != nil {
		return err
	}
	return nil
}

func (c *RemoteClient) Lock(info *state.LockInfo) (string, error) {
	var err error
	var lockID string

	if info.ID == "" {
		lockID, err = uuid.GenerateUUID()
		if err != nil {
			return "", err
		}
		info.ID = lockID
	}

	// Try to acquire lock for the existing row.
	query := `SELECT %s.id, pg_try_advisory_lock(%s.id) FROM %s.%s WHERE %s.name = $1`
	row := c.Client.QueryRow(fmt.Sprintf(query, statesTableName, statesTableName, c.SchemaName, statesTableName, statesTableName), c.Name)
	var pgLockId, didLock []byte
	err = row.Scan(&pgLockId, &didLock)
	switch {
	case err == sql.ErrNoRows:
		// When the row does not yet exist in state, take
		// the `-1` lock to create the new row.
		innerRow := c.Client.QueryRow(`SELECT pg_try_advisory_lock(-1)`)
		var innerDidLock []byte
		err := innerRow.Scan(&innerDidLock)
		if err != nil {
			return "", &state.LockError{Info: info, Err: err}
		}
		if string(innerDidLock) == "false" {
			return "", &state.LockError{Info: info, Err: fmt.Errorf("Already locked for workspace creation: %s", c.Name)}
		}
		info.Path = "-1"
	case err != nil:
		return "", &state.LockError{Info: info, Err: err}
	case string(didLock) == "false":
		return "", &state.LockError{Info: info, Err: fmt.Errorf("Workspace is already locked: %s", c.Name)}
	default:
		info.Path = string(pgLockId)
	}
	c.info = info

	return info.ID, nil
}

func (c *RemoteClient) getLockInfo() (*state.LockInfo, error) {
	return c.info, nil
}

func (c *RemoteClient) Unlock(id string) error {
	if c.info != nil && c.info.Path != "" {
		query := `SELECT pg_advisory_unlock(%s)`
		row := c.Client.QueryRow(fmt.Sprintf(query, c.info.Path))
		var didUnlock []byte
		err := row.Scan(&didUnlock)
		if err != nil {
			return &state.LockError{Info: c.info, Err: err}
		}
		if string(didUnlock) == "false" {
			return &state.LockError{Info: c.info, Err: fmt.Errorf("Workspace is already unlocked: %s", c.Name)}
		}
		c.info = nil
	}
	return nil
}
