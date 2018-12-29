package triton

import (
	"encoding/json"
	"net/http"

	"fmt"
	"github.com/hashicorp/errwrap"
)

type NetworksClient struct {
	*Client
}

// Networks returns a c used for accessing functions pertaining to
// Network functionality in the Triton API.
func (c *Client) Networks() *NetworksClient {
	return &NetworksClient{c}
}

type Network struct {
	Id                  string            `json:"id"`
	Name                string            `json:"name"`
	Public              bool              `json:"public"`
	Fabric              bool              `json:"fabric"`
	Description         string            `json:"description"`
	Subnet              string            `json:"subnet"`
	ProvisioningStartIP string            `json:"provision_start_ip"`
	ProvisioningEndIP   string            `json:"provision_end_ip"`
	Gateway             string            `json:"gateway"`
	Resolvers           []string          `json:"resolvers"`
	Routes              map[string]string `json:"routes"`
	InternetNAT         bool              `json:"internet_nat"`
}

type ListNetworksInput struct{}

func (client *NetworksClient) ListNetworks(*ListNetworksInput) ([]*Network, error) {
	respReader, err := client.executeRequest(http.MethodGet, "/my/networks", nil)
	if respReader != nil {
		defer respReader.Close()
	}
	if err != nil {
		return nil, errwrap.Wrapf("Error executing ListNetworks request: {{err}}", err)
	}

	var result []*Network
	decoder := json.NewDecoder(respReader)
	if err = decoder.Decode(&result); err != nil {
		return nil, errwrap.Wrapf("Error decoding ListNetworks response: {{err}}", err)
	}

	return result, nil
}

type GetNetworkInput struct {
	ID string
}

func (client *NetworksClient) GetNetwork(input *GetNetworkInput) (*Network, error) {
	path := fmt.Sprintf("/%s/networks/%s", client.accountName, input.ID)
	respReader, err := client.executeRequest(http.MethodGet, path, nil)
	if respReader != nil {
		defer respReader.Close()
	}
	if err != nil {
		return nil, errwrap.Wrapf("Error executing GetNetwork request: {{err}}", err)
	}

	var result *Network
	decoder := json.NewDecoder(respReader)
	if err = decoder.Decode(&result); err != nil {
		return nil, errwrap.Wrapf("Error decoding GetNetwork response: {{err}}", err)
	}

	return result, nil
}
