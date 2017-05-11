package triton

import (
	"encoding/json"
	"fmt"
	"net/http"

	"context"
	"github.com/hashicorp/errwrap"
)

type FabricsClient struct {
	*Client
}

// Fabrics returns a client used for accessing functions pertaining to
// Fabric functionality in the Triton API.
func (c *Client) Fabrics() *FabricsClient {
	return &FabricsClient{c}
}

type FabricVLAN struct {
	Name        string `json:"name"`
	ID          int    `json:"vlan_id"`
	Description string `json:"description"`
}

type ListFabricVLANsInput struct{}

func (client *FabricsClient) ListFabricVLANs(ctx context.Context, _ *ListFabricVLANsInput) ([]*FabricVLAN, error) {
	path := fmt.Sprintf("/%s/fabrics/default/vlans", client.accountName)
	respReader, err := client.executeRequest(ctx, http.MethodGet, path, nil)
	if respReader != nil {
		defer respReader.Close()
	}
	if err != nil {
		return nil, errwrap.Wrapf("Error executing ListFabricVLANs request: {{err}}", err)
	}

	var result []*FabricVLAN
	decoder := json.NewDecoder(respReader)
	if err = decoder.Decode(&result); err != nil {
		return nil, errwrap.Wrapf("Error decoding ListFabricVLANs response: {{err}}", err)
	}

	return result, nil
}

type CreateFabricVLANInput struct {
	Name        string `json:"name"`
	ID          int    `json:"vlan_id"`
	Description string `json:"description"`
}

func (client *FabricsClient) CreateFabricVLAN(ctx context.Context, input *CreateFabricVLANInput) (*FabricVLAN, error) {
	path := fmt.Sprintf("/%s/fabrics/default/vlans", client.accountName)
	respReader, err := client.executeRequest(ctx, http.MethodPost, path, input)
	if respReader != nil {
		defer respReader.Close()
	}
	if err != nil {
		return nil, errwrap.Wrapf("Error executing CreateFabricVLAN request: {{err}}", err)
	}

	var result *FabricVLAN
	decoder := json.NewDecoder(respReader)
	if err = decoder.Decode(&result); err != nil {
		return nil, errwrap.Wrapf("Error decoding CreateFabricVLAN response: {{err}}", err)
	}

	return result, nil
}

type UpdateFabricVLANInput struct {
	ID          int    `json:"-"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (client *FabricsClient) UpdateFabricVLAN(ctx context.Context, input *UpdateFabricVLANInput) (*FabricVLAN, error) {
	path := fmt.Sprintf("/%s/fabrics/default/vlans/%d", client.accountName, input.ID)
	respReader, err := client.executeRequest(ctx, http.MethodPut, path, input)
	if respReader != nil {
		defer respReader.Close()
	}
	if err != nil {
		return nil, errwrap.Wrapf("Error executing UpdateFabricVLAN request: {{err}}", err)
	}

	var result *FabricVLAN
	decoder := json.NewDecoder(respReader)
	if err = decoder.Decode(&result); err != nil {
		return nil, errwrap.Wrapf("Error decoding UpdateFabricVLAN response: {{err}}", err)
	}

	return result, nil
}

type GetFabricVLANInput struct {
	ID int `json:"-"`
}

func (client *FabricsClient) GetFabricVLAN(ctx context.Context, input *GetFabricVLANInput) (*FabricVLAN, error) {
	path := fmt.Sprintf("/%s/fabrics/default/vlans/%d", client.accountName, input.ID)
	respReader, err := client.executeRequest(ctx, http.MethodGet, path, nil)
	if respReader != nil {
		defer respReader.Close()
	}
	if err != nil {
		return nil, errwrap.Wrapf("Error executing GetFabricVLAN request: {{err}}", err)
	}

	var result *FabricVLAN
	decoder := json.NewDecoder(respReader)
	if err = decoder.Decode(&result); err != nil {
		return nil, errwrap.Wrapf("Error decoding GetFabricVLAN response: {{err}}", err)
	}

	return result, nil
}

type DeleteFabricVLANInput struct {
	ID int `json:"-"`
}

func (client *FabricsClient) DeleteFabricVLAN(ctx context.Context, input *DeleteFabricVLANInput) error {
	path := fmt.Sprintf("/%s/fabrics/default/vlans/%d", client.accountName, input.ID)
	respReader, err := client.executeRequest(ctx, http.MethodDelete, path, nil)
	if respReader != nil {
		defer respReader.Close()
	}
	if err != nil {
		return errwrap.Wrapf("Error executing DeleteFabricVLAN request: {{err}}", err)
	}

	return nil
}

type ListFabricNetworksInput struct {
	FabricVLANID int `json:"-"`
}

func (client *FabricsClient) ListFabricNetworks(ctx context.Context, input *ListFabricNetworksInput) ([]*Network, error) {
	path := fmt.Sprintf("/%s/fabrics/default/vlans/%d/networks", client.accountName, input.FabricVLANID)
	respReader, err := client.executeRequest(ctx, http.MethodGet, path, nil)
	if respReader != nil {
		defer respReader.Close()
	}
	if err != nil {
		return nil, errwrap.Wrapf("Error executing ListFabricNetworks request: {{err}}", err)
	}

	var result []*Network
	decoder := json.NewDecoder(respReader)
	if err = decoder.Decode(&result); err != nil {
		return nil, errwrap.Wrapf("Error decoding ListFabricNetworks response: {{err}}", err)
	}

	return result, nil
}

type CreateFabricNetworkInput struct {
	FabricVLANID     int               `json:"-"`
	Name             string            `json:"name"`
	Description      string            `json:"description"`
	Subnet           string            `json:"subnet"`
	ProvisionStartIP string            `json:"provision_start_ip"`
	ProvisionEndIP   string            `json:"provision_end_ip"`
	Gateway          string            `json:"gateway"`
	Resolvers        []string          `json:"resolvers"`
	Routes           map[string]string `json:"routes"`
	InternetNAT      bool              `json:"internet_nat"`
}

func (client *FabricsClient) CreateFabricNetwork(ctx context.Context, input *CreateFabricNetworkInput) (*Network, error) {
	path := fmt.Sprintf("/%s/fabrics/default/vlans/%d/networks", client.accountName, input.FabricVLANID)
	respReader, err := client.executeRequest(ctx, http.MethodPost, path, input)
	if respReader != nil {
		defer respReader.Close()
	}
	if err != nil {
		return nil, errwrap.Wrapf("Error executing CreateFabricNetwork request: {{err}}", err)
	}

	var result *Network
	decoder := json.NewDecoder(respReader)
	if err = decoder.Decode(&result); err != nil {
		return nil, errwrap.Wrapf("Error decoding CreateFabricNetwork response: {{err}}", err)
	}

	return result, nil
}

type GetFabricNetworkInput struct {
	FabricVLANID int    `json:"-"`
	NetworkID    string `json:"-"`
}

func (client *FabricsClient) GetFabricNetwork(ctx context.Context, input *GetFabricNetworkInput) (*Network, error) {
	path := fmt.Sprintf("/%s/fabrics/default/vlans/%d/networks/%s", client.accountName, input.FabricVLANID, input.NetworkID)
	respReader, err := client.executeRequest(ctx, http.MethodGet, path, nil)
	if respReader != nil {
		defer respReader.Close()
	}
	if err != nil {
		return nil, errwrap.Wrapf("Error executing GetFabricNetwork request: {{err}}", err)
	}

	var result *Network
	decoder := json.NewDecoder(respReader)
	if err = decoder.Decode(&result); err != nil {
		return nil, errwrap.Wrapf("Error decoding GetFabricNetwork response: {{err}}", err)
	}

	return result, nil
}

type DeleteFabricNetworkInput struct {
	FabricVLANID int    `json:"-"`
	NetworkID    string `json:"-"`
}

func (client *FabricsClient) DeleteFabricNetwork(ctx context.Context, input *DeleteFabricNetworkInput) error {
	path := fmt.Sprintf("/%s/fabrics/default/vlans/%d/networks/%s", client.accountName, input.FabricVLANID, input.NetworkID)
	respReader, err := client.executeRequest(ctx, http.MethodDelete, path, nil)
	if respReader != nil {
		defer respReader.Close()
	}
	if err != nil {
		return errwrap.Wrapf("Error executing DeleteFabricNetwork request: {{err}}", err)
	}

	return nil
}
