package triton

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hashicorp/errwrap"
)

type MachinesClient struct {
	*Client
}

// Machines returns a client used for accessing functions pertaining to
// machine functionality in the Triton API.
func (c *Client) Machines() *MachinesClient {
	return &MachinesClient{c}
}

const (
	machineCNSTagDisable    = "triton.cns.disable"
	machineCNSTagReversePTR = "triton.cns.reverse_ptr"
	machineCNSTagServices   = "triton.cns.services"
)

// MachineCNS is a container for the CNS-specific attributes.  In the API these
// values are embedded within a Machine's Tags attribute, however they are
// exposed to the caller as their native types.
type MachineCNS struct {
	Disable    *bool
	ReversePTR *string
	Services   []string
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
	CNS             MachineCNS
}

// _Machine is a private facade over Machine that handles the necessary API
// overrides from VMAPI's machine endpoint(s).
type _Machine struct {
	Machine
	Tags map[string]interface{} `json:"tags"`
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

func (gmi *GetMachineInput) Validate() error {
	if gmi.ID == "" {
		return fmt.Errorf("machine ID can not be empty")
	}

	return nil
}

func (client *MachinesClient) GetMachine(ctx context.Context, input *GetMachineInput) (*Machine, error) {
	if err := input.Validate(); err != nil {
		return nil, errwrap.Wrapf("unable to get machine: {{err}}", err)
	}

	path := fmt.Sprintf("/%s/machines/%s", client.accountName, input.ID)
	response, err := client.executeRequestRaw(ctx, http.MethodGet, path, nil)
	if response != nil {
		defer response.Body.Close()
	}
	if response.StatusCode == http.StatusNotFound || response.StatusCode == http.StatusGone {
		return nil, &TritonError{
			StatusCode: response.StatusCode,
			Code:       "ResourceNotFound",
		}
	}
	if err != nil {
		return nil, errwrap.Wrapf("Error executing GetMachine request: {{err}}",
			client.decodeError(response.StatusCode, response.Body))
	}

	var result *_Machine
	decoder := json.NewDecoder(response.Body)
	if err = decoder.Decode(&result); err != nil {
		return nil, errwrap.Wrapf("Error decoding GetMachine response: {{err}}", err)
	}

	native, err := result.toNative()
	if err != nil {
		return nil, errwrap.Wrapf("unable to convert API response for machines to native type: {{err}}", err)
	}

	return native, nil
}

type ListMachinesInput struct{}

func (client *MachinesClient) ListMachines(ctx context.Context, _ *ListMachinesInput) ([]*Machine, error) {
	path := fmt.Sprintf("/%s/machines", client.accountName)
	response, err := client.executeRequestRaw(ctx, http.MethodGet, path, nil)
	if response != nil {
		defer response.Body.Close()
	}
	if response.StatusCode == http.StatusNotFound {
		return nil, &TritonError{
			StatusCode: response.StatusCode,
			Code:       "ResourceNotFound",
		}
	}
	if err != nil {
		return nil, errwrap.Wrapf("Error executing ListMachines request: {{err}}",
			client.decodeError(response.StatusCode, response.Body))
	}

	var results []*_Machine
	decoder := json.NewDecoder(response.Body)
	if err = decoder.Decode(&results); err != nil {
		return nil, errwrap.Wrapf("Error decoding ListMachines response: {{err}}", err)
	}

	machines := make([]*Machine, 0, len(results))
	for _, machineAPI := range results {
		native, err := machineAPI.toNative()
		if err != nil {
			return nil, errwrap.Wrapf("unable to convert API response for machines to native type: {{err}}", err)
		}
		machines = append(machines, native)
	}
	return machines, nil
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
	CNS             MachineCNS
}

func (input *CreateMachineInput) toAPI() map[string]interface{} {
	const numExtraParams = 8
	result := make(map[string]interface{}, numExtraParams+len(input.Metadata)+len(input.Tags))

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

	// Deliberately clobber any user-specified Tags with the attributes from the
	// CNS struct.
	input.CNS.toTags(result)

	for key, value := range input.Metadata {
		result[fmt.Sprintf("metadata.%s", key)] = value
	}

	return result
}

func (client *MachinesClient) CreateMachine(ctx context.Context, input *CreateMachineInput) (*Machine, error) {
	path := fmt.Sprintf("/%s/machines", client.accountName)
	respReader, err := client.executeRequest(ctx, http.MethodPost, path, input.toAPI())
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

func (client *MachinesClient) DeleteMachine(ctx context.Context, input *DeleteMachineInput) error {
	path := fmt.Sprintf("/%s/machines/%s", client.accountName, input.ID)
	response, err := client.executeRequestRaw(ctx, http.MethodDelete, path, nil)
	if response.Body != nil {
		defer response.Body.Close()
	}
	if response.StatusCode == http.StatusNotFound || response.StatusCode == http.StatusGone {
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

func (client *MachinesClient) DeleteMachineTags(ctx context.Context, input *DeleteMachineTagsInput) error {
	path := fmt.Sprintf("/%s/machines/%s/tags", client.accountName, input.ID)
	response, err := client.executeRequestRaw(ctx, http.MethodDelete, path, nil)
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

func (client *MachinesClient) DeleteMachineTag(ctx context.Context, input *DeleteMachineTagInput) error {
	path := fmt.Sprintf("/%s/machines/%s/tags/%s", client.accountName, input.ID, input.Key)
	response, err := client.executeRequestRaw(ctx, http.MethodDelete, path, nil)
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

func (client *MachinesClient) RenameMachine(ctx context.Context, input *RenameMachineInput) error {
	path := fmt.Sprintf("/%s/machines/%s", client.accountName, input.ID)

	params := &url.Values{}
	params.Set("action", "rename")
	params.Set("name", input.Name)

	respReader, err := client.executeRequestURIParams(ctx, http.MethodPost, path, nil, params)
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

func (client *MachinesClient) ReplaceMachineTags(ctx context.Context, input *ReplaceMachineTagsInput) error {
	path := fmt.Sprintf("/%s/machines/%s/tags", client.accountName, input.ID)
	respReader, err := client.executeRequest(ctx, http.MethodPut, path, input.Tags)
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

func (client *MachinesClient) AddMachineTags(ctx context.Context, input *AddMachineTagsInput) error {
	path := fmt.Sprintf("/%s/machines/%s/tags", client.accountName, input.ID)
	respReader, err := client.executeRequest(ctx, http.MethodPost, path, input.Tags)
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

func (client *MachinesClient) GetMachineTag(ctx context.Context, input *GetMachineTagInput) (string, error) {
	path := fmt.Sprintf("/%s/machines/%s/tags/%s", client.accountName, input.ID, input.Key)
	respReader, err := client.executeRequest(ctx, http.MethodGet, path, nil)
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

func (client *MachinesClient) ListMachineTags(ctx context.Context, input *ListMachineTagsInput) (map[string]string, error) {
	path := fmt.Sprintf("/%s/machines/%s/tags", client.accountName, input.ID)
	respReader, err := client.executeRequest(ctx, http.MethodGet, path, nil)
	if respReader != nil {
		defer respReader.Close()
	}
	if err != nil {
		return nil, errwrap.Wrapf("Error executing ListMachineTags request: {{err}}", err)
	}

	var result map[string]interface{}
	decoder := json.NewDecoder(respReader)
	if err = decoder.Decode(&result); err != nil {
		return nil, errwrap.Wrapf("Error decoding ListMachineTags response: {{err}}", err)
	}

	_, tags := machineTagsExtractMeta(result)
	return tags, nil
}

type UpdateMachineMetadataInput struct {
	ID       string
	Metadata map[string]string
}

func (client *MachinesClient) UpdateMachineMetadata(ctx context.Context, input *UpdateMachineMetadataInput) (map[string]string, error) {
	path := fmt.Sprintf("/%s/machines/%s/tags", client.accountName, input.ID)
	respReader, err := client.executeRequest(ctx, http.MethodPost, path, input.Metadata)
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

func (client *MachinesClient) ResizeMachine(ctx context.Context, input *ResizeMachineInput) error {
	path := fmt.Sprintf("/%s/machines/%s", client.accountName, input.ID)

	params := &url.Values{}
	params.Set("action", "resize")
	params.Set("package", input.Package)

	respReader, err := client.executeRequestURIParams(ctx, http.MethodPost, path, nil, params)
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

func (client *MachinesClient) EnableMachineFirewall(ctx context.Context, input *EnableMachineFirewallInput) error {
	path := fmt.Sprintf("/%s/machines/%s", client.accountName, input.ID)

	params := &url.Values{}
	params.Set("action", "enable_firewall")

	respReader, err := client.executeRequestURIParams(ctx, http.MethodPost, path, nil, params)
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

func (client *MachinesClient) DisableMachineFirewall(ctx context.Context, input *DisableMachineFirewallInput) error {
	path := fmt.Sprintf("/%s/machines/%s", client.accountName, input.ID)

	params := &url.Values{}
	params.Set("action", "disable_firewall")

	respReader, err := client.executeRequestURIParams(ctx, http.MethodPost, path, nil, params)
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

func (client *MachinesClient) ListNICs(ctx context.Context, input *ListNICsInput) ([]*NIC, error) {
	path := fmt.Sprintf("/%s/machines/%s/nics", client.accountName, input.MachineID)
	respReader, err := client.executeRequest(ctx, http.MethodGet, path, nil)
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

func (client *MachinesClient) AddNIC(ctx context.Context, input *AddNICInput) (*NIC, error) {
	path := fmt.Sprintf("/%s/machines/%s/nics", client.accountName, input.MachineID)
	respReader, err := client.executeRequest(ctx, http.MethodPost, path, input)
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

func (client *MachinesClient) RemoveNIC(ctx context.Context, input *RemoveNICInput) error {
	path := fmt.Sprintf("/%s/machines/%s/nics/%s", client.accountName, input.MachineID, input.MAC)
	respReader, err := client.executeRequest(ctx, http.MethodDelete, path, nil)
	if respReader != nil {
		defer respReader.Close()
	}
	if err != nil {
		return errwrap.Wrapf("Error executing RemoveNIC request: {{err}}", err)
	}

	return nil
}

type StopMachineInput struct {
	MachineID string
}

func (client *MachinesClient) StopMachine(ctx context.Context, input *StopMachineInput) error {
	path := fmt.Sprintf("/%s/machines/%s", client.accountName, input.MachineID)

	params := &url.Values{}
	params.Set("action", "stop")

	respReader, err := client.executeRequestURIParams(ctx, http.MethodPost, path, nil, params)
	if respReader != nil {
		defer respReader.Close()
	}
	if err != nil {
		return errwrap.Wrapf("Error executing StopMachine request: {{err}}", err)
	}

	return nil
}

type StartMachineInput struct {
	MachineID string
}

func (client *MachinesClient) StartMachine(ctx context.Context, input *StartMachineInput) error {
	path := fmt.Sprintf("/%s/machines/%s", client.accountName, input.MachineID)

	params := &url.Values{}
	params.Set("action", "start")

	respReader, err := client.executeRequestURIParams(ctx, http.MethodPost, path, nil, params)
	if respReader != nil {
		defer respReader.Close()
	}
	if err != nil {
		return errwrap.Wrapf("Error executing StartMachine request: {{err}}", err)
	}

	return nil
}

var reservedMachineCNSTags = map[string]struct{}{
	machineCNSTagDisable:    {},
	machineCNSTagReversePTR: {},
	machineCNSTagServices:   {},
}

// machineTagsExtractMeta() extracts all of the misc parameters from Tags and
// returns a clean CNS and Tags struct.
func machineTagsExtractMeta(tags map[string]interface{}) (MachineCNS, map[string]string) {
	nativeCNS := MachineCNS{}
	nativeTags := make(map[string]string, len(tags))
	for k, raw := range tags {
		if _, found := reservedMachineCNSTags[k]; found {
			switch k {
			case machineCNSTagDisable:
				b := raw.(bool)
				nativeCNS.Disable = &b
			case machineCNSTagReversePTR:
				s := raw.(string)
				nativeCNS.ReversePTR = &s
			case machineCNSTagServices:
				nativeCNS.Services = strings.Split(raw.(string), ",")
			default:
				// TODO(seanc@): should assert, logic fail
			}
		} else {
			nativeTags[k] = raw.(string)
		}
	}

	return nativeCNS, nativeTags
}

// toNative() exports a given _Machine (API representation) to its native object
// format.
func (api *_Machine) toNative() (*Machine, error) {
	m := Machine(api.Machine)
	m.CNS, m.Tags = machineTagsExtractMeta(api.Tags)
	return &m, nil
}

// toTags() injects its state information into a Tags map suitable for use to
// submit an API call to the vmapi machine endpoint
func (mcns *MachineCNS) toTags(m map[string]interface{}) {
	if mcns.Disable != nil {
		s := fmt.Sprintf("%t", mcns.Disable)
		m[machineCNSTagDisable] = &s
	}

	if mcns.ReversePTR != nil {
		m[machineCNSTagReversePTR] = &mcns.ReversePTR
	}

	if len(mcns.Services) > 0 {
		m[machineCNSTagServices] = strings.Join(mcns.Services, ",")
	}
}
