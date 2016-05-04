package docker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"

	dc "github.com/fsouza/go-dockerclient"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDockerRegistryCreate(d *schema.ResourceData, meta interface{}) error {
	d.SetId(fmt.Sprintf("%d", rand.Int()))

	return resourceDockerRegistryRead(d, meta)
}

func resourceDockerRegistryUpdate(d *schema.ResourceData, meta interface{}) error {
	return resourceDockerRegistryRead(d, meta)
}

func resourceDockerRegistryRead(d *schema.ResourceData, meta interface{}) error {
	authConfigurations, err := loadAuthconfigurations(d)

	if err != nil {
		return err
	}

	serializedConfigurations, err := serializeDockerRegistryConfigurations(authConfigurations)
	if err != nil {
		return fmt.Errorf("Unable to serialize registry configurations %s", err)
	}

	d.Set("configurations", serializedConfigurations)

	return nil
}

func resourceDockerRegistryDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")

	return nil
}

func loadAuthconfigurations(d *schema.ResourceData) (*dc.AuthConfigurations, error) {
	var authConfigurations *dc.AuthConfigurations
	var err error

	if v, ok := d.GetOk("settings_file"); ok {
		authConfigurations, err = loadConfigurationFromFile(v.(string))

		if err != nil {
			return nil, err
		}
	} else if auth, ok := d.GetOk("auth"); ok {
		authConfigurations = loadConfigurationFromResource(auth.([]interface{}))
	} else {
		return nil, fmt.Errorf("Invalid registry configuration missing 'auth' or 'settings_file' field")
	}

	if len(authConfigurations.Configs) == 0 {
		return nil, fmt.Errorf("No valid configuration found")
	}

	return authConfigurations, nil
}

func loadConfigurationFromFile(filename string) (*dc.AuthConfigurations, error) {
	authFile, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("Registry configuration file error: %s", err)
	}

	authConfigurations, err := dc.NewAuthConfigurations(authFile)
	if err != nil {
		return nil, fmt.Errorf("File configuration invalid syntax: %s", err)
	}

	return authConfigurations, nil
}

func loadConfigurationFromResource(in []interface{}) *dc.AuthConfigurations {
	// Only one element in this array
	auth := in[0].(map[string]interface{})

	authConfigurations := &dc.AuthConfigurations{
		Configs: make(map[string]dc.AuthConfiguration),
	}

	authConfiguration := dc.AuthConfiguration{}
	authConfiguration.Username = auth["username"].(string)
	authConfiguration.Password = auth["password"].(string)
	authConfiguration.ServerAddress = auth["server_address"].(string)
	authConfiguration.Email = auth["email"].(string)

	authConfigurations.Configs[authConfiguration.ServerAddress] = authConfiguration

	return authConfigurations
}

func serializeDockerRegistryConfigurations(in *dc.AuthConfigurations) (string, error) {
	out := new(bytes.Buffer)
	encoder := json.NewEncoder(out)

	// Encoding the map
	err := encoder.Encode(*in)

	return out.String(), err
}

func deserializeDockerRegistryConfigurations(in string) (*dc.AuthConfigurations, error) {
	out := &dc.AuthConfigurations{
		Configs: make(map[string]dc.AuthConfiguration),
	}

	decoder := json.NewDecoder(bytes.NewBufferString(in))

	// Decoding the serialized data
	err := decoder.Decode(out)

	return out, err
}
