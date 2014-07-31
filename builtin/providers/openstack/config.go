package openstack

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
)

type Config struct {
	ApiUrl          string `mapstructure:"url"`
	User            string `mapstructure:"user"`
	Password        string `mapstructure:"password"`
	TenantName      string `mapstructure:"tenantName"`
	ComputeEndpoint string `mapstructure:"computeEndpoint"`
}

type OpenstackClient struct {
	Config *Config
	Token  string
}

type identityObject struct {
	Access accessObject `json:"access"`
}

type accessObject struct {
	Token tokenObject `json:"token"`
}

type tokenObject struct {
	Id string `json:"id"`
}

type authentication struct {
	Auth auth `json:"auth"`
}

type auth struct {
	TenantName          string              `json:"tenantName"`
	PasswordCredentials passwordCredentials `json:"passwordCredentials"`
}

type passwordCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Client() returns a new client for accessing openstack.
//
func (c *Config) Client() (*OpenstackClient, error) {

	if v := os.Getenv("OPENSTACK_URL"); v != "" {
		c.ApiUrl = v
	}
	if v := os.Getenv("OPENSTACK_USER"); v != "" {
		c.User = v
	}
	if v := os.Getenv("OPENSTACK_PASSWORD"); v != "" {
		c.Password = v
	}
	if v := os.Getenv("OPENSTACK_TENANT_NAME"); v != "" {
		c.TenantName = v
	}

	url := c.ApiUrl + "/tokens"

	// FIXME
	passwordCredentials := passwordCredentials{c.User, c.Password}
	auth := auth{c.TenantName, passwordCredentials}
	authentication := authentication{auth}

	body, err := json.Marshal(authentication)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header = http.Header{
		"Accept":       {"application/json"},
		"Content-Type": {"application/json"},
	}

	httpClient := &http.Client{}
	res, err := httpClient.Do(req)

	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, errors.New("Authentication failed: " + res.Status)
	}

	identity := identityObject{}
	err = json.NewDecoder(res.Body).Decode(&identity)

	if err != nil {
		return nil, err
	}

	// TODO retrieve endpoint

	client := &OpenstackClient{c, identity.Access.Token.Id}

	log.Printf("[INFO] Openstack Client configured for user %s", client.Config.User)

	return client, nil
}
