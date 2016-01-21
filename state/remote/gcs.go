package remote

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"google.golang.org/api/googleapi"
	"google.golang.org/api/storage/v1"
)

type GCSClient struct {
	nativeClient *storage.Service
	httpClient   *http.Client
	bucketName   string
	keyName      string
}

func gcsFactory(conf map[string]string) (Client, error) {
	bucketName, ok := conf["bucket"]
	if !ok {
		return nil, fmt.Errorf("missing 'bucket' configuration")
	}

	keyName, ok := conf["key"]
	if !ok {
		return nil, fmt.Errorf("missing 'key' configuration")
	}

	projectKey, ok := conf["project_key"]
	if !ok {
		return nil, fmt.Errorf("missing 'project_key' configuration")
	}

	data, err := ioutil.ReadFile(projectKey)
	if err != nil {
		return nil, fmt.Errorf("can't read 'project_key' file")
	}
	gconf, err := google.JWTConfigFromJSON(data, storage.DevstorageFullControlScope)
	if err != nil {
		log.Fatal(err)
	}
	client := gconf.Client(oauth2.NoContext)

	nativeClient, err := storage.New(client)
	if err != nil {
		log.Fatalf("Unable to create storage service: %v", err)
	}

	return &GCSClient{
		nativeClient: nativeClient,
		httpClient:   client,
		bucketName:   bucketName,
		keyName:      keyName,
	}, nil
}

func (c *GCSClient) Get() (*Payload, error) {
	res, err := c.nativeClient.Objects.Get(c.bucketName, c.keyName).Do()
	if err != nil {
		switch t := err.(type) {
		case *googleapi.Error:
			if t.Code == http.StatusNotFound {
				return nil, nil
			}
			return nil, err
		default:
			return nil, err
		}
	}
	resp, err := c.httpClient.Get(res.MediaLink)
	if err != nil {
		return nil, fmt.Errorf("Failed to load %s: %s", res.MediaLink, err)
	}
	defer resp.Body.Close()
	pload, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Failed to load %s: %s", res.MediaLink, err)
	}
	return &Payload{Data: pload}, nil
}

func (c *GCSClient) Put(data []byte) error {
	// Insert an object into a bucket.
	file := bytes.NewReader(data)
	_, err := c.nativeClient.Objects.Insert(c.bucketName, &storage.Object{Name: c.keyName}).Media(file).Do()
	if err != nil {
		return fmt.Errorf("Failed to insert object: %v", err)
	}
	return nil
}

func (c *GCSClient) Delete() error {
	return c.nativeClient.Objects.Delete(c.bucketName, c.keyName).Do()
}
