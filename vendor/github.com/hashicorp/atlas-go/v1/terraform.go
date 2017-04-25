package atlas

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
)

// TerraformConfigVersion represents a single uploaded version of a
// Terraform configuration.
type TerraformConfigVersion struct {
	Version   int
	Remotes   []string          `json:"remotes"`
	Metadata  map[string]string `json:"metadata"`
	Variables map[string]string `json:"variables,omitempty"`
	TFVars    []TFVar           `json:"tf_vars"`
}

// TFVar is used to serialize a single Terraform variable sent by the
// manager as a collection of Variables in a Job payload.
type TFVar struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	IsHCL bool   `json:"hcl"`
}

// TerraformConfigLatest returns the latest Terraform configuration version.
func (c *Client) TerraformConfigLatest(user, name string) (*TerraformConfigVersion, error) {
	log.Printf("[INFO] getting terraform configuration %s/%s", user, name)

	endpoint := fmt.Sprintf("/api/v1/terraform/configurations/%s/%s/versions/latest", user, name)
	request, err := c.Request("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	response, err := checkResp(c.HTTPClient.Do(request))
	if err == ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var wrapper tfConfigVersionWrapper
	if err := decodeJSON(response, &wrapper); err != nil {
		return nil, err
	}

	return wrapper.Version, nil
}

// CreateTerraformConfigVersion creatse a new Terraform configuration
// versions and uploads a slug with it.
func (c *Client) CreateTerraformConfigVersion(
	user string, name string,
	version *TerraformConfigVersion,
	data io.Reader, size int64) (int, error) {
	log.Printf("[INFO] creating terraform configuration %s/%s", user, name)

	endpoint := fmt.Sprintf(
		"/api/v1/terraform/configurations/%s/%s/versions", user, name)
	body, err := json.Marshal(&tfConfigVersionWrapper{
		Version: version,
	})
	if err != nil {
		return 0, err
	}

	request, err := c.Request("POST", endpoint, &RequestOptions{
		Body: bytes.NewReader(body),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	})
	if err != nil {
		return 0, err
	}

	response, err := checkResp(c.HTTPClient.Do(request))
	if err != nil {
		return 0, err
	}

	var result tfConfigVersionCreate
	if err := decodeJSON(response, &result); err != nil {
		return 0, err
	}

	if err := c.putFile(result.UploadPath, data, size); err != nil {
		return 0, err
	}

	return result.Version, nil
}

type tfConfigVersionCreate struct {
	UploadPath string `json:"upload_path"`
	Version    int
}

type tfConfigVersionWrapper struct {
	Version *TerraformConfigVersion `json:"version"`
}
