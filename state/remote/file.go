package remote

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"os"
)

func fileFactory(conf map[string]string) (Client, error) {
	path, ok := conf["path"]
	if !ok {
		return nil, fmt.Errorf("missing 'path' configuration")
	}

	return &FileClient{
		Path: path,
	}, nil
}

// FileClient is a remote client that stores data locally on disk.
// This is only used for development reasons to test remote state... locally.
type FileClient struct {
	Path string
}

func (c *FileClient) Get() (payload *Payload, err error) {
	var buf bytes.Buffer
	f, err := os.Open(c.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, err
	}
	defer func() {
		err2 := f.Close()
		if err == nil {
			err = err2
		}
	}()

	if _, err := io.Copy(&buf, f); err != nil {
		return nil, err
	}

	md5 := md5.Sum(buf.Bytes())
	return &Payload{
		Data: buf.Bytes(),
		MD5:  md5[:],
	}, nil
}

func (c *FileClient) Put(data []byte) (err error) {
	f, err := os.Create(c.Path)
	if err != nil {
		return err
	}
	defer func() {
		err2 := f.Close()
		if err == nil {
			err = err2
		}
	}()

	_, err = f.Write(data)
	return err
}

func (c *FileClient) Delete() error {
	return os.Remove(c.Path)
}
