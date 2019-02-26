package pg

import (
	"context"
	"crypto/md5"
	"database/sql"
	"fmt"
	"sync"

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

	// In-flight database transaction. Empty unless Locked.
	txn    *sql.Tx
	txnMux sync.Mutex
	info   *state.LockInfo
}

func (c *RemoteClient) Get() (*remote.Payload, error) {
	query := `SELECT data FROM %s.%s WHERE name = $1`
	var row *sql.Row
	// Take exclusive access to the database transaction
	c.txnMux.Lock()
	defer c.txnMux.Unlock()
	// Use the open transaction when present
	if c.txn != nil {
		row = c.txn.QueryRow(fmt.Sprintf(query, c.SchemaName, statesTableName), c.Name)
	} else {
		row = c.Client.QueryRow(fmt.Sprintf(query, c.SchemaName, statesTableName), c.Name)
	}
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
	var err error
	// Take exclusive access to the database transaction
	c.txnMux.Lock()
	defer c.txnMux.Unlock()
	// Use the open transaction when present
	if c.txn != nil {
		_, err = c.txn.Exec(fmt.Sprintf(query, c.SchemaName, statesTableName, statesTableName), c.Name, data)
	} else {
		_, err = c.Client.Exec(fmt.Sprintf(query, c.SchemaName, statesTableName, statesTableName), c.Name, data)
	}
	if err != nil {
		return err
	}
	return nil
}

func (c *RemoteClient) Delete() error {
	query := `DELETE FROM %s.%s WHERE name = $1`
	var err error
	// Take exclusive access to the database transaction
	c.txnMux.Lock()
	defer c.txnMux.Unlock()
	// Use the open transaction when present
	if c.txn != nil {
		_, err = c.txn.Exec(fmt.Sprintf(query, c.SchemaName, statesTableName), c.Name)
	} else {
		_, err = c.Client.Exec(fmt.Sprintf(query, c.SchemaName, statesTableName), c.Name)
	}
	if err != nil {
		return err
	}
	return nil
}

func (c *RemoteClient) Lock(info *state.LockInfo) (string, error) {
	var err error
	var lockID string
	var txn *sql.Tx

	if info.ID == "" {
		lockID, err = uuid.GenerateUUID()
		if err != nil {
			return "", err
		}
		info.Operation = "client"
		info.ID = lockID
	}

	// Take exclusive access to the database transaction
	c.txnMux.Lock()
	defer c.txnMux.Unlock()

	if c.txn == nil {
		// Most strict transaction isolation to prevent cross-talk
		// between incomplete state transactions.
		txn, err = c.Client.BeginTx(context.Background(), &sql.TxOptions{
			Isolation: sql.LevelSerializable,
		})
		if err != nil {
			return "", &state.LockError{Info: info, Err: err}
		}
		c.txn = txn
	} else {
		return "", &state.LockError{Info: info, Err: fmt.Errorf("Client is already in a locking transaction")}
	}

	// Do not wait before giving up on a contended lock.
	_, err = c.Client.Exec(`SET LOCAL lock_timeout = 0`)
	if err != nil {
		c.rollback(info)
		return "", err
	}

	// Try to acquire lock for the existing row.
	query := `SELECT pg_try_advisory_xact_lock(%s.id) FROM %s.%s WHERE %s.name = $1`
	row := c.txn.QueryRow(fmt.Sprintf(query, statesTableName, c.SchemaName, statesTableName, statesTableName), c.Name)
	var didLock []byte
	err = row.Scan(&didLock)
	switch {
	case err == sql.ErrNoRows:
		// When the row does not yet exist in state, take
		// the `-1` lock to create the new row.
		innerRow := c.txn.QueryRow(`SELECT pg_try_advisory_xact_lock(-1)`)
		var innerDidLock []byte
		err := innerRow.Scan(&innerDidLock)
		if err != nil {
			c.rollback(info)
			return "", &state.LockError{Info: info, Err: err}
		}
		if string(innerDidLock) == "false" {
			c.rollback(info)
			return "", &state.LockError{Info: info, Err: fmt.Errorf("Workspace is already locked: %s", c.Name)}
		}
	case err != nil:
		c.rollback(info)
		return "", &state.LockError{Info: info, Err: err}
	case string(didLock) == "false":
		c.rollback(info)
		return "", &state.LockError{Info: info, Err: fmt.Errorf("Workspace is already locked: %s", c.Name)}
	default:
	}

	return info.ID, nil
}

func (c *RemoteClient) getLockInfo() (*state.LockInfo, error) {
	return c.info, nil
}

func (c *RemoteClient) Unlock(id string) error {
	// Take exclusive access to the database transaction
	c.txnMux.Lock()
	defer c.txnMux.Unlock()
	if c.txn != nil {
		err := c.txn.Commit()
		if err != nil {
			return err
		}
		c.txn = nil
	}
	c.info = nil
	return nil
}

// This must be called from any code path where the
// transaction would not be committed (unlocked),
// otherwise the transactions will leak and prevent
// the process from exiting cleanly.
//
// Does not use mutex because this will implicitly be
// called from within an already mutex'd scope.
func (c *RemoteClient) rollback(info *state.LockInfo) error {
	if c.txn != nil {
		err := c.txn.Rollback()
		if err != nil {
			return err
		}
		c.txn = nil
	}
	c.info = nil
	return nil
}
