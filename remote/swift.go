package remote

import (
    "bytes"
    "bufio"
	"crypto/md5"
	"fmt"
    "io/ioutil"
    "os"
    "strings"

    "github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack"
	"github.com/rackspace/gophercloud/openstack/objectstorage/v1/containers"
	"github.com/rackspace/gophercloud/openstack/objectstorage/v1/objects"
)

const TFSTATE_NAME = "tfstate.tf"

// SwiftRemoteClient implements the RemoteClient interface
// for a Swift compatible server.
type SwiftRemoteClient struct {
    client *gophercloud.ServiceClient
    path string
}

func NewSwiftRemoteClient(conf map[string]string) (*SwiftRemoteClient, error) {
	client := &SwiftRemoteClient{}

	if err := client.validateConfig(conf); err != nil {
		return nil, err
	}

    return client, nil
}

func (c *SwiftRemoteClient) validateConfig(conf map[string]string) (err error) {
    if val := os.Getenv("OS_AUTH_URL"); val == "" {
        return fmt.Errorf("missing OS_AUTH_URL environment variable")
    }
    if val := os.Getenv("OS_USERNAME"); val == "" {
        return fmt.Errorf("missing OS_USERNAME environment variable")
    }
    if val := os.Getenv("OS_TENANT_NAME"); val == "" {
        return fmt.Errorf("missing OS_TENANT_NAME environment variable")
    }
    if val := os.Getenv("OS_PASSWORD"); val == "" {
        return fmt.Errorf("missing OS_PASSWORD environment variable")
    }
    path, ok := conf["path"]
    if !ok || path == "" {
        return fmt.Errorf("missing 'path' configuration")
    }

    provider, err := openstack.AuthenticatedClient(gophercloud.AuthOptions{
        IdentityEndpoint: os.Getenv("OS_AUTH_URL"),
        Username:         os.Getenv("OS_USERNAME"),
        TenantName:       os.Getenv("OS_TENANT_NAME"),
        Password:         os.Getenv("OS_PASSWORD"),
	})

    c.path = path
    c.client, err = openstack.NewObjectStorageV1(provider, gophercloud.EndpointOpts{
		Region: os.Getenv("OS_REGION_NAME"),
	})

    return err
}

func (c *SwiftRemoteClient) GetState() (*RemoteStatePayload, error) {
    fmt.Println("Downloading object...")
    result := objects.Download(c.client, c.path, TFSTATE_NAME, nil)

    if result.Err != nil {
        // GopherCloud doesn't give us an elegant way to distinguish
        // between an actual error and "not found".
        reader := bufio.NewReader(result.Body)

        if buffer, err := ioutil.ReadAll(reader); err == nil {
            body := string(buffer[:])
            fmt.Println("BODY: " + body)
            // If the object doesn't exist, then return empty state.
            if strings.Contains(body, "404") {
                fmt.Println("Body contains 404")
                return nil, nil
            }
        } else {
            return nil, err
        }

        // Return the non-404 HTTP error.
        return nil, result.Err
    }

    reader := bufio.NewReader(result.Body)
    scanner := bufio.NewScanner(reader)

	// Create the payload
	payload := &RemoteStatePayload{
		State: scanner.Bytes(),
	}

	// Generate the MD5
	hash := md5.Sum(payload.State)
	payload.MD5 = hash[:md5.Size]
	return payload, nil
}

func (c *SwiftRemoteClient) PutState(state []byte, force bool) error {
    // Ensure the Swift Container exists.
    if err := c.ensureContainer(); err != nil {
        return err
    }

    readbuffer := bytes.NewBuffer(state)
    reader := bufio.NewReader(readbuffer)
    result := objects.Create(c.client, c.path, TFSTATE_NAME, reader, nil)

    return result.Err
}

func (c *SwiftRemoteClient) DeleteState() error {
    result := objects.Delete(c.client, c.path, TFSTATE_NAME, nil)
	return result.Err
}

func (c *SwiftRemoteClient) ensureContainer() error {
    result := containers.Create(c.client, c.path, nil)
    if result.Err != nil {
        return result.Err
    }

    return nil
}
