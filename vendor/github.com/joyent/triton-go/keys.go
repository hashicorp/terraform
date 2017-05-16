package triton

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/errwrap"
)

type KeysClient struct {
	*Client
}

// Keys returns a c used for accessing functions pertaining to
// SSH key functionality in the Triton API.
func (c *Client) Keys() *KeysClient {
	return &KeysClient{c}
}

// Key represents a public key
type Key struct {
	// Name of the key
	Name string `json:"name"`

	// Key fingerprint
	Fingerprint string `json:"fingerprint"`

	// OpenSSH-formatted public key
	Key string `json:"key"`
}

type ListKeysInput struct{}

// ListKeys lists all public keys we have on record for the specified
// account.
func (client *KeysClient) ListKeys(ctx context.Context, _ *ListKeysInput) ([]*Key, error) {
	path := fmt.Sprintf("/%s/keys", client.accountName)
	respReader, err := client.executeRequest(ctx, http.MethodGet, path, nil)
	if respReader != nil {
		defer respReader.Close()
	}
	if err != nil {
		return nil, errwrap.Wrapf("Error executing ListKeys request: {{err}}", err)
	}

	var result []*Key
	decoder := json.NewDecoder(respReader)
	if err = decoder.Decode(&result); err != nil {
		return nil, errwrap.Wrapf("Error decoding ListKeys response: {{err}}", err)
	}

	return result, nil
}

type GetKeyInput struct {
	KeyName string
}

func (client *KeysClient) GetKey(ctx context.Context, input *GetKeyInput) (*Key, error) {
	path := fmt.Sprintf("/%s/keys/%s", client.accountName, input.KeyName)
	respReader, err := client.executeRequest(ctx, http.MethodGet, path, nil)
	if respReader != nil {
		defer respReader.Close()
	}
	if err != nil {
		return nil, errwrap.Wrapf("Error executing GetKey request: {{err}}", err)
	}

	var result *Key
	decoder := json.NewDecoder(respReader)
	if err = decoder.Decode(&result); err != nil {
		return nil, errwrap.Wrapf("Error decoding GetKey response: {{err}}", err)
	}

	return result, nil
}

type DeleteKeyInput struct {
	KeyName string
}

func (client *KeysClient) DeleteKey(ctx context.Context, input *DeleteKeyInput) error {
	path := fmt.Sprintf("/%s/keys/%s", client.accountName, input.KeyName)
	respReader, err := client.executeRequest(ctx, http.MethodDelete, path, nil)
	if respReader != nil {
		defer respReader.Close()
	}
	if err != nil {
		return errwrap.Wrapf("Error executing DeleteKey request: {{err}}", err)
	}

	return nil
}

// CreateKeyInput represents the option that can be specified
// when creating a new key.
type CreateKeyInput struct {
	// Name of the key. Optional.
	Name string `json:"name,omitempty"`

	// OpenSSH-formatted public key.
	Key string `json:"key"`
}

// CreateKey uploads a new OpenSSH key to Triton for use in HTTP signing and SSH.
func (client *KeysClient) CreateKey(ctx context.Context, input *CreateKeyInput) (*Key, error) {
	path := fmt.Sprintf("/%s/keys", client.accountName)
	respReader, err := client.executeRequest(ctx, http.MethodPost, path, input)
	if respReader != nil {
		defer respReader.Close()
	}
	if err != nil {
		return nil, errwrap.Wrapf("Error executing CreateKey request: {{err}}", err)
	}

	var result *Key
	decoder := json.NewDecoder(respReader)
	if err = decoder.Decode(&result); err != nil {
		return nil, errwrap.Wrapf("Error decoding CreateKey response: {{err}}", err)
	}

	return result, nil
}
