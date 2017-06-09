package remote

import (
	"crypto/md5"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	joyentclient "github.com/joyent/gocommon/client"
	joyenterrors "github.com/joyent/gocommon/errors"
	"github.com/joyent/gomanta/manta"
	joyentauth "github.com/joyent/gosign/auth"
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

	creds, err := getCredentialsFromEnvironment()

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

func getCredentialsFromEnvironment() (cred *joyentauth.Credentials, err error) {

	user := os.Getenv("MANTA_USER")
	keyId := os.Getenv("MANTA_KEY_ID")
	url := os.Getenv("MANTA_URL")
	keyMaterial := os.Getenv("MANTA_KEY_MATERIAL")

	if _, err := os.Stat(keyMaterial); err == nil {
		// key material is a file path; try to read it
		keyBytes, err := ioutil.ReadFile(keyMaterial)
		if err != nil {
			return nil, fmt.Errorf("Error reading key material from %s: %s",
				keyMaterial, err)
		} else {
			block, _ := pem.Decode(keyBytes)
			if block == nil {
				return nil, fmt.Errorf(
					"Failed to read key material '%s': no key found", keyMaterial)
			}

			if block.Headers["Proc-Type"] == "4,ENCRYPTED" {
				return nil, fmt.Errorf(
					"Failed to read key '%s': password protected keys are\n"+
						"not currently supported. Please decrypt the key prior to use.", keyMaterial)
			}

			keyMaterial = string(keyBytes)
		}
	}

	authentication, err := joyentauth.NewAuth(user, keyMaterial, "rsa-sha256")
	if err != nil {
		return nil, fmt.Errorf("Error constructing authentication for %s: %s", user, err)
	}

	return &joyentauth.Credentials{
		UserAuthentication: authentication,
		SdcKeyId:           "",
		SdcEndpoint:        joyentauth.Endpoint{},
		MantaKeyId:         keyId,
		MantaEndpoint:      joyentauth.Endpoint{URL: url},
	}, nil
}
