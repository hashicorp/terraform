package pg

import (
	"crypto/md5"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	multierror "github.com/hashicorp/go-multierror"
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

	// Uses locks to synchronize state access
	lock bool
	info *state.LockInfo
}

func (c *RemoteClient) Get() (*remote.Payload, error) {
	query := `SELECT data FROM %s.%s WHERE name = $1`
	row := c.Client.QueryRow(fmt.Sprintf(query, c.SchemaName, statesTableName), c.Name)
	var data []byte
	err := row.Scan(&data)
	if err != nil {
		return nil, nil
	}
	md5 := md5.Sum(data)
	return &remote.Payload{
		Data: data,
		MD5:  md5[:],
	}, nil
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
	// No-op when locking is disabled
	if !c.lock {
		return "", nil
	}

	c.info = info

	if info.ID == "" {
		lockID, err := uuid.GenerateUUID()
		if err != nil {
			return "", err
		}

		info.ID = lockID
	}

	lockInfo, _ := c.getLockInfo()
	if lockInfo != nil {
		lockErr := &state.LockError{
			Info: lockInfo,
		}
		lockErr.Err = errors.New("state locked")
		return "", lockErr
	}

	query := `INSERT INTO %s.%s (name, info) VALUES ($1, $2)`
	data, err := json.Marshal(info)
	if err != nil {
		return "", err
	}
	_, err = c.Client.Exec(fmt.Sprintf(query, c.SchemaName, locksTableName), c.Name, data)
	if err != nil {
		return "", err
	}

	if err != nil {
		lockInfo, infoErr := c.getLockInfo()
		if infoErr != nil {
			err = multierror.Append(err, infoErr)
		}

		lockErr := &state.LockError{
			Err:  err,
			Info: lockInfo,
		}
		return "", lockErr
	}

	return info.ID, nil
}

func (c *RemoteClient) getLockInfo() (*state.LockInfo, error) {
	query := `SELECT info FROM %s.%s WHERE name = $1`
	row := c.Client.QueryRow(fmt.Sprintf(query, c.SchemaName, locksTableName), c.Name)
	var data []byte
	err := row.Scan(&data)
	if err != nil {
		return nil, err
	}

	lockInfo := &state.LockInfo{}
	err = json.Unmarshal(data, lockInfo)
	if err != nil {
		return nil, err
	}

	return lockInfo, nil
}

func (c *RemoteClient) Unlock(id string) error {
	lockErr := &state.LockError{}

	lockInfo, err := c.getLockInfo()
	if err != nil {
		lockErr.Err = fmt.Errorf("failed to retrieve lock info: %s", err)
		return lockErr
	}
	lockErr.Info = lockInfo

	if lockInfo.ID != id {
		lockErr.Err = fmt.Errorf("lock id %q does not match existing lock", id)
		return lockErr
	}

	query := `DELETE FROM %s.%s WHERE name = $1`
	_, err = c.Client.Exec(fmt.Sprintf(query, c.SchemaName, locksTableName), c.Name)
	if err != nil {
		lockErr.Err = err
		return lockErr
	}

	c.info = nil

	return nil
}
