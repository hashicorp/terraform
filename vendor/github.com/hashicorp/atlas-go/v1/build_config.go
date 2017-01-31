package atlas

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
)

// bcWrapper is the API wrapper since the server wraps the resulting object.
type bcWrapper struct {
	BuildConfig *BuildConfig `json:"build_configuration"`
}

// Atlas expects a list of key/value vars
type BuildVar struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
type BuildVars []BuildVar

// BuildConfig represents a Packer build configuration.
type BuildConfig struct {
	// User is the namespace under which the build config lives
	User string `json:"username"`

	// Name is the actual name of the build config, unique in the scope
	// of the username.
	Name string `json:"name"`
}

// Slug returns the slug format for this BuildConfig (User/Name)
func (b *BuildConfig) Slug() string {
	return fmt.Sprintf("%s/%s", b.User, b.Name)
}

// BuildConfigVersion represents a single uploaded (or uploadable) version
// of a build configuration.
type BuildConfigVersion struct {
	// The fields below are the username/name combo to uniquely identify
	// a build config.
	User string `json:"username"`
	Name string `json:"name"`

	// Builds is the list of builds that this version supports.
	Builds []BuildConfigBuild
}

// Slug returns the slug format for this BuildConfigVersion (User/Name)
func (bv *BuildConfigVersion) Slug() string {
	return fmt.Sprintf("%s/%s", bv.User, bv.Name)
}

// BuildConfigBuild is a single build that is present in an uploaded
// build configuration.
type BuildConfigBuild struct {
	// Name is a unique name for this build
	Name string `json:"name"`

	// Type is the type of builder that this build needs to run on,
	// such as "amazon-ebs" or "qemu".
	Type string `json:"type"`

	// Artifact is true if this build results in one or more artifacts
	// being sent to Atlas
	Artifact bool `json:"artifact"`
}

// BuildConfig gets a single build configuration by user and name.
func (c *Client) BuildConfig(user, name string) (*BuildConfig, error) {
	log.Printf("[INFO] getting build configuration %s/%s", user, name)

	endpoint := fmt.Sprintf("/api/v1/packer/build-configurations/%s/%s", user, name)
	request, err := c.Request("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	response, err := checkResp(c.HTTPClient.Do(request))
	if err != nil {
		return nil, err
	}

	var bc BuildConfig
	if err := decodeJSON(response, &bc); err != nil {
		return nil, err
	}

	return &bc, nil
}

// CreateBuildConfig creates a new build configuration.
func (c *Client) CreateBuildConfig(user, name string) (*BuildConfig, error) {
	log.Printf("[INFO] creating build configuration %s/%s", user, name)

	endpoint := "/api/v1/packer/build-configurations"
	body, err := json.Marshal(&bcWrapper{
		BuildConfig: &BuildConfig{
			User: user,
			Name: name,
		},
	})
	if err != nil {
		return nil, err
	}

	request, err := c.Request("POST", endpoint, &RequestOptions{
		Body: bytes.NewReader(body),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	})
	if err != nil {
		return nil, err
	}

	response, err := checkResp(c.HTTPClient.Do(request))
	if err != nil {
		return nil, err
	}

	var bc BuildConfig
	if err := decodeJSON(response, &bc); err != nil {
		return nil, err
	}

	return &bc, nil
}

// UploadBuildConfigVersion creates a single build configuration version
// and uploads the template associated with it.
//
// Actual API: "Create Build Config Version"
func (c *Client) UploadBuildConfigVersion(v *BuildConfigVersion, metadata map[string]interface{},
	vars BuildVars, data io.Reader, size int64) error {

	log.Printf("[INFO] uploading build configuration version %s (%d bytes), with metadata %q",
		v.Slug(), size, metadata)

	endpoint := fmt.Sprintf("/api/v1/packer/build-configurations/%s/%s/versions",
		v.User, v.Name)

	var bodyData bcCreateWrapper
	bodyData.Version.Builds = v.Builds
	bodyData.Version.Metadata = metadata
	bodyData.Version.Vars = vars
	body, err := json.Marshal(bodyData)
	if err != nil {
		return err
	}

	request, err := c.Request("POST", endpoint, &RequestOptions{
		Body: bytes.NewReader(body),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	})
	if err != nil {
		return err
	}

	response, err := checkResp(c.HTTPClient.Do(request))
	if err != nil {
		return err
	}

	var bv bcCreate
	if err := decodeJSON(response, &bv); err != nil {
		return err
	}

	if err := c.putFile(bv.UploadPath, data, size); err != nil {
		return err
	}

	return nil
}

// bcCreate is the struct returned when creating a build configuration.
type bcCreate struct {
	UploadPath string `json:"upload_path"`
}

// bcCreateWrapper is the wrapper for creating a build config.
type bcCreateWrapper struct {
	Version struct {
		Metadata map[string]interface{} `json:"metadata,omitempty"`
		Builds   []BuildConfigBuild     `json:"builds"`
		Vars     BuildVars              `json:"packer_vars,omitempty"`
	} `json:"version"`
}
