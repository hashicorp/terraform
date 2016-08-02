package remote

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/hashicorp/terraform/helper/pathorcontents"
	"github.com/hashicorp/terraform/terraform"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/storage/v1"
)

// accountFile represents the structure of the credentials JSON
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
	bucket        string
	path          string
	clientStorage *storage.Service
	context       context.Context
}

func gcsFactory(conf map[string]string) (Client, error) {
	var account accountFile
	var client *http.Client
	clientScopes := []string{
		"https://www.googleapis.com/auth/devstorage.full_control",
	}

	bucketName, ok := conf["bucket"]
	if !ok {
		return nil, fmt.Errorf("missing 'bucket' configuration")
	}

	pathName, ok := conf["path"]
	if !ok {
		return nil, fmt.Errorf("missing 'path' configuration")
	}

	credentials, ok := conf["credentials"]
	if !ok {
		credentials = os.Getenv("GOOGLE_CREDENTIALS")
	}

	if credentials != "" {
		contents, _, err := pathorcontents.Read(credentials)
		if err != nil {
			return nil, fmt.Errorf("Error loading credentials: %s", err)
		}

		// Assume account_file is a JSON string
		if err := parseJSON(&account, contents); err != nil {
			return nil, fmt.Errorf("Error parsing credentials '%s': %s", contents, err)
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

		client = conf.Client(oauth2.NoContext)

	} else {
		log.Printf("[INFO] Authenticating using DefaultClient")
		err := error(nil)
		client, err = google.DefaultClient(oauth2.NoContext, clientScopes...)
		if err != nil {
			return nil, err
		}
	}
	versionString := terraform.Version
	userAgent := fmt.Sprintf(
		"(%s %s) Terraform/%s", runtime.GOOS, runtime.GOARCH, versionString)

	log.Printf("[INFO] Instantiating Google Storage Client...")
	clientStorage, err := storage.New(client)
	if err != nil {
		return nil, err
	}
	clientStorage.UserAgent = userAgent

	return &GCSClient{
		clientStorage: clientStorage,
		bucket:        bucketName,
		path:          pathName,
	}, nil

}

func (c *GCSClient) Get() (*Payload, error) {
	// Read the object from bucket.
	log.Printf("[INFO] Reading %s/%s", c.bucket, c.path)

	resp, err := c.clientStorage.Objects.Get(c.bucket, c.path).Download()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			log.Printf("[INFO] %s/%s not found", c.bucket, c.path)

			return nil, nil
		}

		return nil, fmt.Errorf("[WARN] Error retrieving object %s/%s: %s", c.bucket, c.path, err)
	}
	defer resp.Body.Close()

	var buf []byte
	w := bytes.NewBuffer(buf)
	n, err := io.Copy(w, resp.Body)
	if err != nil {
		log.Fatalf("[WARN] error buffering %q: %v", c.path, err)
	}
	log.Printf("[INFO] Downloaded %d bytes", n)

	payload := &Payload{
		Data: w.Bytes(),
	}

	// If there was no data, then return nil
	if len(payload.Data) == 0 {
		return nil, nil
	}

	return payload, nil
}

func (c *GCSClient) Put(data []byte) error {
	log.Printf("[INFO] Writing %s/%s", c.bucket, c.path)

	r := bytes.NewReader(data)
	_, err := c.clientStorage.Objects.Insert(c.bucket, &storage.Object{Name: c.path}).Media(r).Do()
	if err != nil {
		return err
	}

	return nil
}

func (c *GCSClient) Delete() error {
	log.Printf("[INFO] Deleting %s/%s", c.bucket, c.path)

	err := c.clientStorage.Objects.Delete(c.bucket, c.path).Do()
	return err

}
