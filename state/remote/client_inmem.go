package remote

import (
	"crypto/md5"
)

// InmemClient is a Client implementation that stores data in memory.
type InmemClient struct {
	Data []byte
	MD5  []byte
}

func (c *InmemClient) Get() (*Payload, error) {
	return &Payload{
		Data: c.Data,
		MD5:  c.MD5,
	}, nil
}

func (c *InmemClient) Put(data []byte) error {
	md5 := md5.Sum(data)

	c.Data = data
	c.MD5 = md5[:]
	return nil
}

func (c *InmemClient) Delete() error {
	c.Data = nil
	c.MD5 = nil
	return nil
}
