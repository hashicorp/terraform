package triton

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/errwrap"
	"net/url"
)

type MachinesClient struct {
	*Client
}

// Machines returns a client used for accessing functions pertaining to
// machine functionality in the Triton API.
func (c *Client) Machines() *MachinesClient {
	return &MachinesClient{c}
}

type Machine struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Type            string            `json:"type"`
	Brand           string            `json:"brand"`
	State           string            `json:"state"`
	Image           string            `json:"image"`
	Memory          int               `json:"memory"`
	Disk            int               `json:"disk"`
	Metadata        map[string]string `json:"metadata"`
	Tags            map[string]string `json:"tags"`
	Created         time.Time         `json:"created"`
	Updated         time.Time         `json:"updated"`
	Docker          bool              `json:"docker"`
	IPs             []string          `json:"ips"`
	Networks        []string          `json:"networks"`
	PrimaryIP       string            `json:"primaryIp"`
	FirewallEnabled bool              `json:"firewall_enabled"`
	ComputeNode     string            `json:"compute_node"`
	Package         string            `json:"package"`
	DomainNames     []string          `json:"dns_names"`
}

type NIC struct {
	IP      string `json:"ip"`
	MAC     string `json:"mac"`
	Primary bool   `json:"primary"`
	Netmask string `json:"netmask"`
	Gateway string `json:"gateway"`
	State   string `json:"state"`
	Network string `json:"network"`
}

type GetMachineInput struct {
	ID string
}

func (client *MachinesClient) GetMachine(input *GetMachineInput) (*Machine, error) {
	path := fmt.Sprintf("/%s/machines/%s", client.accountName, input.ID)
	response, err := client.executeRequestRaw(http.MethodGet, path, nil)
	if response != nil {
		defer response.Body.Close()
	}
	if response.StatusCode == http.StatusNotFound {
		return nil, &TritonError{
			Code: "ResourceNotFound",
		}
	}
	if err != nil {
		return nil, errwrap.Wrapf("Error executing GetMachine request: {{err}}",
			client.decodeError(response.StatusCode, response.Body))
	}

	var result *Machine
	decoder := json.NewDecoder(response.Body)
	if err = decoder.Decode(&result); err != nil {
		return nil, errwrap.Wrapf("Error decoding GetMachine response: {{err}}", err)
	}

	return result, nil
}

type CreateMachineInput struct {
	Name            string
	Package         string
	Image           string
	Networks        []string
	LocalityStrict  bool
	LocalityNear    []string
	LocalityFar     []string
	Metadata        map[string]string
	Tags            map[string]string
	FirewallEnabled bool
}

func transformCreateMachineInput(input *CreateMachineInput) map[string]interface{} {
	result := make(map[string]interface{}, 8+len(input.Metadata)+len(input.Tags))
	result["firewall_enabled"] = input.FirewallEnabled
	if input.Name != "" {
		result["name"] = input.Name
	}
	if input.Package != "" {
		result["package"] = input.Package
	}
	if input.Image != "" {
		result["image"] = input.Image
	}
	if len(input.Networks) > 0 {
		result["networks"] = input.Networks
	}
	locality := struct {
		Strict bool     `json:"strict"`
		Near   []string `json:"near,omitempty"`
		Far    []string `json:"far,omitempty"`
	}{
		Strict: input.LocalityStrict,
		Near:   input.LocalityNear,
		Far:    input.LocalityFar,
	}
	result["locality"] = locality
	for key, value := range input.Tags {
		result[fmt.Sprintf("tag.%s", key)] = value
	}
	for key, value := range input.Metadata {
		result[fmt.Sprintf("metadata.%s", key)] = value
	}

	return result
}

func (client *MachinesClient) CreateMachine(input *CreateMachineInput) (*Machine, error) {
	respReader, err := client.executeRequest(http.MethodPost, "/my/machines", transformCreateMachineInput(input))
	if respReader != nil {
		defer respReader.Close()
	}
	if err != nil {
		return nil, errwrap.Wrapf("Error executing CreateMachine request: {{err}}", err)
	}

	var result *Machine
	decoder := json.NewDecoder(respReader)
	if err = decoder.Decode(&result); err != nil {
		return nil, errwrap.Wrapf("Error decoding CreateMachine response: {{err}}", err)
	}

	return result, nil
}

type DeleteMachineInput struct {
	ID string
}

func (client *MachinesClient) DeleteMachine(input *DeleteMachineInput) error {
	path := fmt.Sprintf("/%s/machines/%s", client.accountName, input.ID)
	response, err := client.executeRequestRaw(http.MethodDelete, path, nil)
	if response.Body != nil {
		defer response.Body.Close()
	}
	if response.StatusCode == http.StatusNotFound {
		return nil
	}
	if err != nil {
		return errwrap.Wrapf("Error executing DeleteMachine request: {{err}}",
			client.decodeError(response.StatusCode, response.Body))
	}

	return nil
}

type DeleteMachineTagsInput struct {
	ID string
}

func (client *MachinesClient) DeleteMachineTags(input *DeleteMachineTagsInput) error {
	path := fmt.Sprintf("/%s/machines/%s/tags", client.accountName, input.ID)
	response, err := client.executeRequestRaw(http.MethodDelete, path, nil)
	if response.Body != nil {
		defer response.Body.Close()
	}
	if response.StatusCode == http.StatusNotFound {
		return nil
	}
	if err != nil {
		return errwrap.Wrapf("Error executing DeleteMachineTags request: {{err}}",
			client.decodeError(response.StatusCode, response.Body))
	}

	return nil
}

type DeleteMachineTagInput struct {
	ID  string
	Key string
}

func (client *MachinesClient) DeleteMachineTag(input *DeleteMachineTagInput) error {
	path := fmt.Sprintf("/%s/machines/%s/tags/%s", client.accountName, input.ID, input.Key)
	response, err := client.executeRequestRaw(http.MethodDelete, path, nil)
	if response.Body != nil {
		defer response.Body.Close()
	}
	if response.StatusCode == http.StatusNotFound {
		return nil
	}
	if err != nil {
		return errwrap.Wrapf("Error executing DeleteMachineTag request: {{err}}",
			client.decodeError(response.StatusCode, response.Body))
	}

	return nil
}

type RenameMachineInput struct {
	ID   string
	Name string
}

func (client *MachinesClient) RenameMachine(input *RenameMachineInput) error {
	path := fmt.Sprintf("/%s/machines/%s", client.accountName, input.ID)

	params := &url.Values{}
	params.Set("action", "rename")
	params.Set("name", input.Name)

	respReader, err := client.executeRequestURIParams(http.MethodPost, path, nil, params)
	if respReader != nil {
		defer respReader.Close()
	}
	if err != nil {
		return errwrap.Wrapf("Error executing RenameMachine request: {{err}}", err)
	}

	return nil
}

type ReplaceMachineTagsInput struct {
	ID   string
	Tags map[string]string
}

func (client *MachinesClient) ReplaceMachineTags(input *ReplaceMachineTagsInput) error {
	path := fmt.Sprintf("/%s/machines/%s/tags", client.accountName, input.ID)
	respReader, err := client.executeRequest(http.MethodPut, path, input.Tags)
	if respReader != nil {
		defer respReader.Close()
	}
	if err != nil {
		return errwrap.Wrapf("Error executing ReplaceMachineTags request: {{err}}", err)
	}

	return nil
}

type AddMachineTagsInput struct {
	ID   string
	Tags map[string]string
}

func (client *MachinesClient) AddMachineTags(input *AddMachineTagsInput) error {
	path := fmt.Sprintf("/%s/machines/%s/tags", client.accountName, input.ID)
	respReader, err := client.executeRequest(http.MethodPost, path, input.Tags)
	if respReader != nil {
		defer respReader.Close()
	}
	if err != nil {
		return errwrap.Wrapf("Error executing AddMachineTags request: {{err}}", err)
	}

	return nil
}

type GetMachineTagInput struct {
	ID  string
	Key string
}

func (client *MachinesClient) GetMachineTag(input *GetMachineTagInput) (string, error) {
	path := fmt.Sprintf("/%s/machines/%s/tags/%s", client.accountName, input.ID, input.Key)
	respReader, err := client.executeRequest(http.MethodGet, path, nil)
	if respReader != nil {
		defer respReader.Close()
	}
	if err != nil {
		return "", errwrap.Wrapf("Error executing GetMachineTag request: {{err}}", err)
	}

	var result string
	decoder := json.NewDecoder(respReader)
	if err = decoder.Decode(&result); err != nil {
		return "", errwrap.Wrapf("Error decoding GetMachineTag response: {{err}}", err)
	}

	return result, nil
}

type ListMachineTagsInput struct {
	ID string
}

func (client *MachinesClient) ListMachineTags(input *ListMachineTagsInput) (map[string]string, error) {
	path := fmt.Sprintf("/%s/machines/%s/tags", client.accountName, input.ID)
	respReader, err := client.executeRequest(http.MethodGet, path, nil)
	if respReader != nil {
		defer respReader.Close()
	}
	if err != nil {
		return nil, errwrap.Wrapf("Error executing ListMachineTags request: {{err}}", err)
	}

	var result map[string]string
	decoder := json.NewDecoder(respReader)
	if err = decoder.Decode(&result); err != nil {
		return nil, errwrap.Wrapf("Error decoding ListMachineTags response: {{err}}", err)
	}

	return result, nil
}

type UpdateMachineMetadataInput struct {
	ID       string
	Metadata map[string]string
}

func (client *MachinesClient) UpdateMachineMetadata(input *UpdateMachineMetadataInput) (map[string]string, error) {
	path := fmt.Sprintf("/%s/machines/%s/tags", client.accountName, input.ID)
	respReader, err := client.executeRequest(http.MethodPost, path, input.Metadata)
	if respReader != nil {
		defer respReader.Close()
	}
	if err != nil {
		return nil, errwrap.Wrapf("Error executing UpdateMachineMetadata request: {{err}}", err)
	}

	var result map[string]string
	decoder := json.NewDecoder(respReader)
	if err = decoder.Decode(&result); err != nil {
		return nil, errwrap.Wrapf("Error decoding UpdateMachineMetadata response: {{err}}", err)
	}

	return result, nil
}

type ResizeMachineInput struct {
	ID      string
	Package string
}

func (client *MachinesClient) ResizeMachine(input *ResizeMachineInput) error {
	path := fmt.Sprintf("/%s/machines/%s", client.accountName, input.ID)

	params := &url.Values{}
	params.Set("action", "resize")
	params.Set("package", input.Package)

	respReader, err := client.executeRequestURIParams(http.MethodPost, path, nil, params)
	if respReader != nil {
		defer respReader.Close()
	}
	if err != nil {
		return errwrap.Wrapf("Error executing ResizeMachine request: {{err}}", err)
	}

	return nil
}

type EnableMachineFirewallInput struct {
	ID string
}

func (client *MachinesClient) EnableMachineFirewall(input *EnableMachineFirewallInput) error {
	path := fmt.Sprintf("/%s/machines/%s", client.accountName, input.ID)

	params := &url.Values{}
	params.Set("action", "enable_firewall")

	respReader, err := client.executeRequestURIParams(http.MethodPost, path, nil, params)
	if respReader != nil {
		defer respReader.Close()
	}
	if err != nil {
		return errwrap.Wrapf("Error executing EnableMachineFirewall request: {{err}}", err)
	}

	return nil
}

type DisableMachineFirewallInput struct {
	ID string
}

func (client *MachinesClient) DisableMachineFirewall(input *DisableMachineFirewallInput) error {
	path := fmt.Sprintf("/%s/machines/%s", client.accountName, input.ID)

	params := &url.Values{}
	params.Set("action", "disable_firewall")

	respReader, err := client.executeRequestURIParams(http.MethodPost, path, nil, params)
	if respReader != nil {
		defer respReader.Close()
	}
	if err != nil {
		return errwrap.Wrapf("Error executing DisableMachineFirewall request: {{err}}", err)
	}

	return nil
}

type ListNICsInput struct {
	MachineID string
}

func (client *MachinesClient) ListNICs(input *ListNICsInput) ([]*NIC, error) {
	respReader, err := client.executeRequest(http.MethodGet, fmt.Sprintf("/my/machines/%s/nics", input.MachineID), nil)
	if respReader != nil {
		defer respReader.Close()
	}
	if err != nil {
		return nil, errwrap.Wrapf("Error executing ListNICs request: {{err}}", err)
	}

	var result []*NIC
	decoder := json.NewDecoder(respReader)
	if err = decoder.Decode(&result); err != nil {
		return nil, errwrap.Wrapf("Error decoding ListNICs response: {{err}}", err)
	}

	return result, nil
}

type AddNICInput struct {
	MachineID string `json:"-"`
	Network   string `json:"network"`
}

func (client *MachinesClient) AddNIC(input *AddNICInput) (*NIC, error) {
	path := fmt.Sprintf("/%s/machines/%s/nics", client.accountName, input.MachineID)
	respReader, err := client.executeRequest(http.MethodPost, path, input)
	if respReader != nil {
		defer respReader.Close()
	}
	if err != nil {
		return nil, errwrap.Wrapf("Error executing AddNIC request: {{err}}", err)
	}

	var result *NIC
	decoder := json.NewDecoder(respReader)
	if err = decoder.Decode(&result); err != nil {
		return nil, errwrap.Wrapf("Error decoding AddNIC response: {{err}}", err)
	}

	return result, nil
}

type RemoveNICInput struct {
	MachineID string
	MAC       string
}

func (client *MachinesClient) RemoveNIC(input *RemoveNICInput) error {
	path := fmt.Sprintf("/%s/machines/%s/nics/%s", client.accountName, input.MachineID, input.MAC)
	respReader, err := client.executeRequest(http.MethodDelete, path, nil)
	if respReader != nil {
		defer respReader.Close()
	}
	if err != nil {
		return errwrap.Wrapf("Error executing RemoveNIC request: {{err}}", err)
	}

	return nil
}
