package remote

import (
	"fmt"
	"log"
	"os"

	"encoding/json"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	storageAPI "google.golang.org/api/storage/v1"
	"google.golang.org/cloud"
	storage "google.golang.org/cloud/storage"
	"io/ioutil"
	"net/http"
	"strings"
)

// accountFile represents the structure of the account file JSON file.
// TODO: this is a copy n paste from google provider config.go - use common source
type accountFile struct {
	PrivateKeyId string `json:"private_key_id"`
	PrivateKey   string `json:"private_key"`
	ClientEmail  string `json:"client_email"`
	ClientId     string `json:"client_id"`
}

func parseJSON(result interface{}, contents string) error {
	r := strings.NewReader(contents)
	dec := json.NewDecoder(r)

	return dec.Decode(result)
}

type GCSClient struct {
	bucket string
	path   string

	client  *storageAPI.Service
	context context.Context
}

func gcsFactory(conf map[string]string) (Client, error) {
	bucketName, ok := conf["bucket"]
	if !ok {
		return nil, fmt.Errorf("missing 'bucket' configuration")
	}

	pathName, ok := conf["path"]
	if !ok {
		return nil, fmt.Errorf("missing 'path' configuration")
	}

	projectId, ok := conf["project"]
	if !ok {
		projectId = os.Getenv("GOOGLE_PROJECT")
		if projectId == "" {
			return nil, fmt.Errorf(
				"missing 'project' configuration or GOOGLE_PROJECT environment variable")
		}
	}

	accountFilePath, ok := conf["account_file"]

	var client *http.Client
	var account accountFile

	if accountFilePath != "" {

		contents := accountFilePath

		// Assume account_file is a JSON string
		if err := parseJSON(&account, contents); err != nil {
			// If account_file was not JSON, assume it is a file path instead
			if _, err := os.Stat(accountFilePath); os.IsNotExist(err) {
				return nil, fmt.Errorf(
					"account_file path does not exist: %s",
					accountFilePath)
			}

			b, err := ioutil.ReadFile(accountFilePath)
			if err != nil {
				return nil, fmt.Errorf(
					"Error reading account_file from path '%s': %s",
					accountFilePath,
					err)
			}

			contents = string(b)

			if err := parseJSON(&account, contents); err != nil {
				return nil, fmt.Errorf(
					"Error parsing account file '%s': %s",
					contents,
					err)
			}
		}

		clientScopes := []string{
			storageAPI.DevstorageFullControlScope,
		}

		// Get the token for use in our requests
		log.Printf("[INFO] Requesting Google token...")
		log.Printf("[INFO]   -- Email: %s", account.ClientEmail)
		log.Printf("[INFO]   -- Scopes: %s", clientScopes)
		log.Printf("[INFO]   -- Private Key Length: %d", len(account.PrivateKey))

		conf := jwt.Config{
			Email:      account.ClientEmail,
			PrivateKey: []byte(account.PrivateKey),
			Scopes:     clientScopes,
			TokenURL:   "https://accounts.google.com/o/oauth2/token",
		}

		// Initiate an http.Client.
		client = conf.Client(oauth2.NoContext)

	} else {
		log.Printf("[INFO] Requesting Google token gcloud auth or service role")

		// Authentication is provided by the gcloud tool when running locally, and
		// by the associated service account when running on Compute Engine.
		var err error
		client, err = google.DefaultClient(context.Background(), storageAPI.DevstorageFullControlScope)

		if err != nil {
			log.Fatalf("Unable to get default client: %v", err)
		}
	}

	service, err := storageAPI.New(client)
	if err != nil {
		log.Fatalf("Unable to create storage service: %v", err)
	}

	ctx := cloud.NewContext(projectId, client)

	return &GCSClient{
		client:  service,
		context: ctx,
		bucket:  bucketName,
		path:    pathName,
	}, nil

}

func (c *GCSClient) Get() (*Payload, error) {
	// Read the object from bucket.
	rc, err := storage.NewReader(c.context, c.bucket, c.path)
	if err != nil {
		if err.Error() == "storage: object doesn't exist" {
			return nil, nil
		}
		return nil, fmt.Errorf("Failed to read remote state: %s", err)
	}
	defer rc.Close()
	slurp, err := ioutil.ReadAll(rc)

	if err != nil {
		return nil, fmt.Errorf("Failed to read remote state: %s", err)
	}

	payload := &Payload{
		Data: slurp,
	}

	// If there was no data, then return nil
	if len(payload.Data) == 0 {
		return nil, nil
	}

	return payload, nil
}

func (c *GCSClient) Put(data []byte) error {

	wc := storage.NewWriter(c.context, c.bucket, c.path)
	wc.ContentType = "application/json"
	wc.ACL = []storage.ACLRule{
		{storage.AllAuthenticatedUsers, storage.RoleReader},
		{storage.AllAuthenticatedUsers, storage.RoleOwner},
	}

	if _, err := wc.Write(data); err != nil {
		return fmt.Errorf("Failed to upload state: %v", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("Error while uploading state: %v", err)
	}

	return nil
}

func (c *GCSClient) Delete() error {

	err := storage.DeleteObject(c.context, c.bucket, c.path)
	return err

}
