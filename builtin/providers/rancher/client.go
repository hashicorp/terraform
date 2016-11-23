package rancher

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/hashicorp/go-cleanhttp"
)

// Client struct holding connection string
type Client struct {
	ServerUrl  string
	AccessKey  string
	SecretKey  string
	ApiVersion int
	Http       *http.Client
}

// NewClient returns a new Rancher client
func NewClient(serverUrl string, accessKey string, secretKey string) (*Client, error) {
	client := Client{
		ServerUrl: serverUrl,
		AccessKey: accessKey,
		SecretKey: secretKey,
		Http:      cleanhttp.DefaultClient(),
	}
	var err error
	client.ApiVersion, err = client.detectApiVersion()
	if err != nil {
		return nil, err
	}
	return &client, nil
}

// Detects the API version in use on the server
// TODO: implement that
func (client *Client) detectApiVersion() (int, error) {
	return 1, nil
}

// Creates a new request with necessary headers
func (c *Client) newRequest(method string, endpoint string, body []byte) (*http.Request, error) {

	urlStr := c.ServerUrl + "/v" + strconv.Itoa(c.ApiVersion) + endpoint
	url, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("Error during parsing request URL: %s", err)
	}

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, url.String(), bodyReader)
	if err != nil {
		return nil, fmt.Errorf("Error during creation of request: %s", err)
	}

	req.SetBasicAuth(c.AccessKey, c.SecretKey)
	req.Header.Add("Accept", "application/json")

	if method != "GET" {
		req.Header.Add("Content-Type", "application/json")
	}

	return req, nil
}

type Environments struct {
	Environments []Environment `json:"data"`
}

type Environment struct {
	Id                string              `json:"id"`
	Description       string              `json:"description"`
	Kubernetes        bool                `json:kubernetes"`
	Members           []EnvironmentMember `json:"members"`
	Mesos             bool                `json:"mesos"`
	Name              string              `json:"name"`
	PublicDNS         bool                `json:"publicDns"`
	ServicesPortRange PortRange           `json:"servicesPortRange"`
	Swarm             bool                `json:"swarm"`
	VirtualMachine    bool                `json:"virtualMachine"`
}

type EnvironmentMember struct {
	ExternalId     string `json:"externalId"`
	ExternalIdType string `json:"externalIdType"`
	Role           string `json:"role"`
}

type PortRange struct {
	StartPort int `json:"startPort"`
	EndPort   int `json:"endPort"`
}

type RegistrationTokens struct {
	Tokens []RegistrationToken `json:"data"`
}

type RegistrationToken struct {
	Id              string `json:"id"`
	State           string `json:"state"`
	RegistrationUrl string `json:"registrationUrl"`
	Token           string `json:"token"`
}

func (client *Client) CreateEnvironment(env Environment) (string, error) {
	reqBody, _ := json.Marshal(env)

	req, err := client.newRequest("POST", "/projects", reqBody)
	if err != nil {
		return "", err
	}

	resp, err := client.Http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return "", fmt.Errorf("Error creating environment: %s", env.Name)
	}

	newEnv := new(Environment)
	if err = json.NewDecoder(resp.Body).Decode(newEnv); err != nil {
		return "", fmt.Errorf("Failed to get new environment id for %s", env.Name)
	}

	return newEnv.Id, nil
}

func (client *Client) GetEnvironmentById(id string) (e *Environment, err error) {
	req, err := client.newRequest("GET", fmt.Sprintf("/projects/%s", id), nil)
	if err != nil {
		return
	}

	resp, err := client.Http.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	env := new(Environment)
	err = json.NewDecoder(resp.Body).Decode(env)
	if err != nil {
		return
	}

	return env, nil
}

func (client *Client) DeleteEnvironmentById(id string) (err error) {
	req, err := client.newRequest("DELETE", fmt.Sprintf("/projects/%s", id), nil)
	if err != nil {
		return
	}

	_, err = client.Http.Do(req)
	return
}

func (client *Client) EnvironmentExists(name string) (bool, error) {
	req, err := client.newRequest("GET", "/projects", nil)
	if err != nil {
		return false, err
	}

	resp, err := client.Http.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return false, fmt.Errorf("Error checking environment: %s", name)
	}

	envs := new(Environments)
	if err = json.NewDecoder(resp.Body).Decode(envs); err != nil {
		return false, fmt.Errorf("Failed to list environments looking for %s", name)
	}

	for _, e := range envs.Environments {
		if e.Name == name {
			return true, nil
		}
	}

	return false, nil
}

func (client *Client) GetRegistrationToken(id string) (token RegistrationToken, err error) {
	req, err := client.newRequest("GET", fmt.Sprintf("/projects/%s/registrationtokens", id), nil)
	if err != nil {
		return
	}

	resp, err := client.Http.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return token, fmt.Errorf("Error getting registation token for environment: %s", id)
	}

	tokens := new(RegistrationTokens)
	if err = json.NewDecoder(resp.Body).Decode(tokens); err != nil {
		return token, fmt.Errorf("Failed to list registration tokens for environment %s", id)
	}

	for _, t := range tokens.Tokens {
		if t.State == "active" {
			return t, nil
		}
	}

	return token, nil
}
