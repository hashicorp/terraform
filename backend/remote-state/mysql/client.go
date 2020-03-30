package mysql

import (
	"crypto/md5"
	"database/sql"
	"fmt"

	// mysql import
	_ "github.com/go-sql-driver/mysql"
	uuid "github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
)

// RemoteClient is a remote client that stores data in a MySQL/MariaDB database
type RemoteClient struct {
	Client     *sql.DB
	Name       string
	SchemaName string

	info *state.LockInfo
}

// Get func
func (c *RemoteClient) Get() (*remote.Payload, error) {
	query := `SELECT data FROM %s.%s WHERE name = ?`
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

// Put func
func (c *RemoteClient) Put(data []byte) error {
	query := `INSERT INTO %s.%s (name, data) VALUES (?, ?) ON DUPLICATE KEY UPDATE data = ?`
	_, err := c.Client.Exec(fmt.Sprintf(query, c.SchemaName, statesTableName), c.Name, data, data)

	if err != nil {
		return err
	}
	return nil
}

//Delete func
func (c *RemoteClient) Delete() error {
	query := `DELETE FROM %s.%s WHERE name = ?`
	_, err := c.Client.Exec(fmt.Sprintf(query, c.SchemaName, statesTableName), c.Name)
	if err != nil {
		return err
	}
	return nil
}

//Lock func
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

	// Local helper function so we can call it multiple places
	//
	lockUnlock := func(advisoryLockId string) error {
		query := `SELECT RELEASE_LOCK('%s')`
		row := c.Client.QueryRow(fmt.Sprintf(query, advisoryLockId))
		var didUnlock []byte
		err := row.Scan(&didUnlock)
		if err != nil {
			return &state.LockError{Info: info, Err: err}
		}
		return nil
	}

	// Try to acquire locks for the existing row `id` and the creation lock `terraform_workspace_creation_lock` (non-blocking).
	//query := `SELECT %s.id, GET_LOCK(%s.id, %d), GET_LOCK('%s', %d) FROM %s.%s WHERE %s.name = ?`
	timeout := 5
	workspaceCreationLockName := "terraform_workspace_creation_lock"
	query := `SELECT %s.id, GET_LOCK(%s.id, %d), GET_LOCK('%s', %d) FROM %s.%s WHERE %s.name = ?`
	row := c.Client.QueryRow(fmt.Sprintf(query, statesTableName, statesTableName, timeout, workspaceCreationLockName, timeout, c.SchemaName, statesTableName, statesTableName), c.Name)
	var mysqlLockID, didLock, didLockForCreate []byte
	err = row.Scan(&mysqlLockID, &didLock, &didLockForCreate)
	switch {
	case err == sql.ErrNoRows:
		// No rows means we're creating the workspace. Take the creation lock.
		innerRow := c.Client.QueryRow(fmt.Sprintf(`SELECT GET_LOCK('%s', %d)`, workspaceCreationLockName, timeout))
		var innerDidLock []byte
		err := innerRow.Scan(&innerDidLock)
		if err != nil {
			return "", &state.LockError{Info: info, Err: err}
		}
		if string(innerDidLock) == "0" {
			return "", &state.LockError{Info: info, Err: fmt.Errorf("Already locked for workspace creation: %s", c.Name)}
		}
		info.Path = workspaceCreationLockName
	case err != nil:
		return "", &state.LockError{Info: info, Err: err}
	case string(didLock) == "0":
		// Existing workspace is already locked. Release the attempted creation lock.
		lockUnlock(workspaceCreationLockName)
		return "", &state.LockError{Info: info, Err: fmt.Errorf("Workspace is already locked: %s", c.Name)}
	case string(didLockForCreate) == "0":
		// Someone has the creation lock already. Release the existing workspace because it might not be safe to touch.
		lockUnlock(string(mysqlLockID))
		return "", &state.LockError{Info: info, Err: fmt.Errorf("Cannot lock workspace; already locked for workspace creation: %s", c.Name)}
	default:
		// Existing workspace is now locked. Release the attempted creation lock.
		lockUnlock(workspaceCreationLockName)
		info.Path = string(mysqlLockID)
	}
	c.info = info

	return info.ID, nil
}

func (c *RemoteClient) getLockInfo() (*state.LockInfo, error) {
	return c.info, nil
}

//Unlock func
func (c *RemoteClient) Unlock(id string) error {
	if c.info != nil && c.info.Path != "" {
		query := `SELECT RELEASE_LOCK('%s')`
		row := c.Client.QueryRow(fmt.Sprintf(query, c.info.Path))
		var didUnlock []byte
		err := row.Scan(&didUnlock)
		if err != nil {
			return &state.LockError{Info: c.info, Err: err}
		}
		c.info = nil
	}
	return nil
}
