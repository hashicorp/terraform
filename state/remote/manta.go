package remote

import (
	"crypto/md5"
	"fmt"
	"log"
	"os"

	joyentclient "github.com/joyent/gocommon/client"
	joyenterrors "github.com/joyent/gocommon/errors"
	"github.com/joyent/gocommon/jpc"
	"github.com/joyent/gomanta/manta"
)

const DEFAULT_OBJECT_NAME = "terraform.tfstate"

func mantaFactory(conf map[string]string) (Client, error) {
	path, ok := conf["path"]
	if !ok {
		return nil, fmt.Errorf("missing 'path' configuration")
	}

	objectName, ok := conf["objectName"]
	if !ok {
		objectName = DEFAULT_OBJECT_NAME
	}

	keyName, ok := conf["keyName"]
	if !ok {
		keyName = ""
	}

	creds, err := jpc.CompleteCredentialsFromEnv(keyName)
	if err != nil {
		return nil, fmt.Errorf("Error getting Manta credentials: %s", err.Error())
	}

	client := manta.New(joyentclient.NewClient(
		creds.MantaEndpoint.URL,
		"",
		creds,
		log.New(os.Stderr, "", log.LstdFlags),
	))

	return &MantaClient{
		Client:     client,
		Path:       path,
		ObjectName: objectName,
	}, nil
}

type MantaClient struct {
	Client     *manta.Client
	Path       string
	ObjectName string
}

func (c *MantaClient) Get() (*Payload, error) {
	bytes, err := c.Client.GetObject(c.Path, c.ObjectName)
	if err != nil {
		if joyenterrors.IsResourceNotFound(err.(joyenterrors.Error).Cause()) {
			return nil, nil
		}

		return nil, err
	}

	md5 := md5.Sum(bytes)

	return &Payload{
		Data: bytes,
		MD5:  md5[:],
	}, nil
}

func (c *MantaClient) Put(data []byte) error {
	return c.Client.PutObject(c.Path, c.ObjectName, data)
}

func (c *MantaClient) Delete() error {
	return c.Client.DeleteObject(c.Path, c.ObjectName)
}
