package inmem

import (
	"github.com/hashicorp/terraform/states/statemgr"
)

// RemoteClient is a remote client that stores data in memory for testing.
type RemoteClient struct {
	Data []byte
	Name string
}

func (c *RemoteClient) Get() ([]byte, error) {
	if c.Data == nil {
		return nil, nil
	}

	return c.Data, nil
}

func (c *RemoteClient) Put(data []byte) error {
	c.Data = data
	return nil
}

func (c *RemoteClient) Delete() error {
	c.Data = nil
	return nil
}

func (c *RemoteClient) Lock(info *statemgr.LockInfo) (string, error) {
	return locks.lock(c.Name, info)
}
func (c *RemoteClient) Unlock(id string) error {
	return locks.unlock(c.Name, id)
}
