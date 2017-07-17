package b2

import (
	"bytes"
	"crypto/md5"
	"io/ioutil"
	"log"

	"github.com/hashicorp/terraform/state/remote"
	"gopkg.in/kothar/go-backblaze.v0"
)

type RemoteClient struct {
	bucket *backblaze.Bucket
	path   string
}

func (c *RemoteClient) Get() (*remote.Payload, error) {
	log.Printf("[DEBUG] Getting file %s from bucket %s", c.path, c.bucket.Name)

	// 404 errors are allowed
	_, reader, err := c.bucket.DownloadFileByName(c.path)
	if err != nil {
		if err.(*backblaze.B2Error).Status == 404 {
			return nil, nil
		}
		return nil, err
	}

	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	hash := md5.Sum(data)
	payload := &remote.Payload{
		Data: data,
		MD5:  hash[:md5.Size],
	}

	return payload, nil
}

func (c *RemoteClient) Put(data []byte) error {
	log.Printf("[DEBUG] Putting file %s in bucket %s", c.path, c.bucket.Name)

	reader := bytes.NewReader(data)
	_, err := c.bucket.UploadFile(c.path, nil, reader)

	return err
}

func (c *RemoteClient) Delete() error {
	log.Printf("[DEBUG] Deleting file %s in bucket %s", c.path, c.bucket.Name)
	_, err := c.bucket.HideFile(c.path)
	return err
}
