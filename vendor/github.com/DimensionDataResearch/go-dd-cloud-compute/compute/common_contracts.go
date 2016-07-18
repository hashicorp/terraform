package compute

import "fmt"

// EntitySummary is used to group an entity Id and name together for serialisation / deserialisation purposes.
type EntitySummary struct {
	// The entity Id.
	ID string `json:"id"`
	// The entity name.
	Name string `json:"name,omitempty"`
}

// IPRange represents an IPvX range.
type IPRange interface {
	// Convert the IPvX range to a display string.
	ToDisplayString() string
}

// IPv4Range represents an IPv4 network (base address and prefix size)
type IPv4Range struct {
	// The network base address.
	BaseAddress string `json:"address"`
	// The network prefix size.
	PrefixSize int `json:"prefixSize"`
}

// ToDisplayString converts the IPv4 range to a display string.
func (network IPv4Range) ToDisplayString() string {
	return fmt.Sprintf("%s/%d", network.BaseAddress, network.PrefixSize)
}

// IPv6Range represents an IPv6 network (base address and prefix size)
type IPv6Range struct {
	// The network base address.
	BaseAddress string `json:"address"`
	// The network prefix size.
	PrefixSize int `json:"prefixSize"`
}

// ToDisplayString converts the IPv6 range to a display string.
func (network IPv6Range) ToDisplayString() string {
	return fmt.Sprintf("%s/%d", network.BaseAddress, network.PrefixSize)
}

// OperatingSystem represents a well-known operating system for virtual machines.
type OperatingSystem struct {
	// The operating system Id.
	ID string `json:"id"`

	// The operating system type.
	Family string `json:"family"`

	// The operating system display-name.
	DisplayName string `json:"displayName"`
}

// VirtualMachineCPU represents the CPU configuration for a virtual machine.
type VirtualMachineCPU struct {
	Count          int    `json:"count,omitempty"`
	Speed          string `json:"speed,omitempty"`
	CoresPerSocket int    `json:"coresPerSocket,omitempty"`
}

// VirtualMachineDisk represents the disk configuration for a virtual machine.
type VirtualMachineDisk struct {
	ID         *string `json:"id,omitempty"`
	SCSIUnitID int     `json:"scsiId"`
	SizeGB     int     `json:"sizeGb"`
	Speed      string  `json:"speed"`
}

// VirtualMachineNetwork represents the networking configuration for a virtual machine.
type VirtualMachineNetwork struct {
	NetworkDomainID           string                         `json:"networkDomainId,omitempty"`
	PrimaryAdapter            VirtualMachineNetworkAdapter   `json:"primaryNic"`
	AdditionalNetworkAdapters []VirtualMachineNetworkAdapter `json:"additionalNic"`
}

// VirtualMachineNetworkAdapter represents the configuration for a virtual machine's network adapter.
// If deploying a new VM, exactly one of VLANID / PrivateIPv4Address must be specified.
type VirtualMachineNetworkAdapter struct {
	ID                 *string `json:"id,omitempty"`
	VLANID             *string `json:"vlanId,omitempty"`
	VLANName           *string `json:"vlanName,omitempty"`
	PrivateIPv4Address *string `json:"privateIpv4,omitempty"`
	PrivateIPv6Address *string `json:"ipv6,omitempty"`
	State              *string `json:"state,omitempty"`
}

// GetID returns the network adapter's Id.
func (server *VirtualMachineNetworkAdapter) GetID() string {
	if server.ID == nil {
		return ""
	}

	return *server.ID
}

// GetName returns the network adapter's name (actually Id, since adapters don't have names).
func (server *VirtualMachineNetworkAdapter) GetName() string {
	return server.GetID()
}

// GetState returns the network adapter's current state.
func (server *VirtualMachineNetworkAdapter) GetState() string {
	if server.State == nil {
		return ""
	}

	return *server.State
}

// IsDeleted determines whether the network adapter has been deleted (is nil).
func (server *VirtualMachineNetworkAdapter) IsDeleted() bool {
	return server == nil
}

var _ Resource = &VirtualMachineNetworkAdapter{}
