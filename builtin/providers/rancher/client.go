package rancher

import (
	"net/http"

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
