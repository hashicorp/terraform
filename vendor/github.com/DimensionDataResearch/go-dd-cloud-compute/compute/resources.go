package compute

import (
	"fmt"
	"strings"
)

// Resources are an abstraction over the various types of entities in the DD compute API

const (
	// ResourceTypeNetworkDomain represents a network domain.
	ResourceTypeNetworkDomain = "NetworkDomain"

	// ResourceTypeVLAN represents a VLAN.
	ResourceTypeVLAN = "VLAN"

	// ResourceTypeServer represents a virtual machine.
	ResourceTypeServer = "Server"

	// ResourceTypeNetworkAdapter represents a network adapter in a virtual machine.
	// Note that when calling methods such as WaitForChange, the Id must be of the form 'serverId/networkAdapterId'.
	ResourceTypeNetworkAdapter = "NetworkAdapter"

	// ResourceTypePublicIPBlock represents a block of public IP addresses.
	ResourceTypePublicIPBlock = "PublicIPBlock"

	// ResourceTypeFirewallRule represents a firewall rule.
	ResourceTypeFirewallRule = "FirewallRule"
)

// Resource represents a compute resource.
type Resource interface {
	// The resource ID.
	GetID() string

	// The resource name.
	GetName() string

	// The resource's current state (e.g. ResourceStatusNormal, etc).
	GetState() string

	// Has the resource been deleted (i.e. the underlying struct is nil)?
	IsDeleted() bool
}

// GetResourceDescription retrieves a textual description of the specified resource type.
func GetResourceDescription(resourceType string) (string, error) {
	switch resourceType {
	case ResourceTypeNetworkDomain:
		return "Network domain", nil

	case ResourceTypeVLAN:
		return "VLAN", nil

	case ResourceTypeServer:
		return "Server", nil

	case ResourceTypeNetworkAdapter:
		return "Network adapter", nil

	case ResourceTypePublicIPBlock:
		return "Public IPv4 address block", nil

	case ResourceTypeFirewallRule:
		return "Firewall rule", nil

	default:
		return "", fmt.Errorf("Unrecognised resource type '%s'.", resourceType)
	}
}

// GetResource retrieves a compute resource of the specified type by Id.
// id is the resource Id.
// resourceType is the resource type (e.g. ResourceTypeNetworkDomain, ResourceTypeVLAN, etc).
func (client *Client) GetResource(id string, resourceType string) (Resource, error) {
	var resourceLoader func(client *Client, id string) (resource Resource, err error)

	switch resourceType {
	case ResourceTypeNetworkDomain:
		resourceLoader = getNetworkDomainByID

	case ResourceTypeVLAN:
		resourceLoader = getVLANByID

	case ResourceTypeServer:
		resourceLoader = getServerByID

	case ResourceTypeNetworkAdapter:
		resourceLoader = getNetworkAdapterByID

	case ResourceTypePublicIPBlock:
		resourceLoader = getPublicIPBlockByID

	case ResourceTypeFirewallRule:
		resourceLoader = getFirewallRuleByID

	default:
		return nil, fmt.Errorf("Unrecognised resource type '%s'.", resourceType)
	}

	return resourceLoader(client, id)
}

func getNetworkDomainByID(client *Client, id string) (networkDomain Resource, err error) {
	return client.GetNetworkDomain(id)
}

func getVLANByID(client *Client, id string) (Resource, error) {
	return client.GetVLAN(id)
}

func getServerByID(client *Client, id string) (Resource, error) {
	return client.GetServer(id)
}

func getNetworkAdapterByID(client *Client, id string) (Resource, error) {
	compositeIDComponents := strings.Split(id, "/")
	if len(compositeIDComponents) != 2 {
		return nil, fmt.Errorf("'%s' is not a valid network adapter Id (when loading as a resource, the Id must be of the form 'serverId/networkAdapterId')", id)
	}

	server, err := client.GetServer(compositeIDComponents[0])
	if err != nil {
		return nil, err
	}
	if server == nil {
		return nil, fmt.Errorf("No server found with Id '%s.'", compositeIDComponents)
	}

	var targetAdapterID = compositeIDComponents[1]
	if *server.Network.PrimaryAdapter.ID == targetAdapterID {
		return &server.Network.PrimaryAdapter, nil
	}

	for _, adapter := range server.Network.AdditionalNetworkAdapters {
		if *adapter.ID == targetAdapterID {
			return &adapter, nil
		}
	}

	return nil, nil
}

func getPublicIPBlockByID(client *Client, id string) (Resource, error) {
	return client.GetPublicIPBlock(id)
}

func getFirewallRuleByID(client *Client, id string) (Resource, error) {
	return client.GetFirewallRule(id)
}
