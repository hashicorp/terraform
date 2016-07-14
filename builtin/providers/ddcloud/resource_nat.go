package ddcloud

import (
	"fmt"
	"github.com/DimensionDataResearch/go-dd-cloud-compute/compute"
	"github.com/hashicorp/terraform/helper/schema"
	"log"
	"strconv"
	"strings"
	"time"
)

const (
	resourceKeyNATNetworkDomainID = "networkdomain"
	resourceKeyNATPrivateAddress  = "private_ipv4"
	resourceKeyNATPublicAddress   = "public_ipv4"
	resourceCreateTimeoutNAT      = 30 * time.Minute
	resourceUpdateTimeoutNAT      = 10 * time.Minute
	resourceDeleteTimeoutNAT      = 15 * time.Minute
)

func resourceNAT() *schema.Resource {
	return &schema.Resource{
		Create: resourceNATCreate,
		Read:   resourceNATRead,
		Update: resourceNATUpdate,
		Delete: resourceNATDelete,

		Schema: map[string]*schema.Schema{
			resourceKeyNATNetworkDomainID: &schema.Schema{
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
				Description: "The network domain that the NAT rule applies to.",
			},
			resourceKeyNATPrivateAddress: &schema.Schema{
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
				Description: "The private (internal) IPv4 address.",
			},
			resourceKeyNATPublicAddress: &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Default:     nil,
				Description: "The public (external) IPv4 address.",
			},
		},
	}
}

// Create a NAT resource.
func resourceNATCreate(data *schema.ResourceData, provider interface{}) error {
	var err error

	propertyHelper := propertyHelper(data)

	networkDomainID := data.Get(resourceKeyNATNetworkDomainID).(string)
	privateIP := data.Get(resourceKeyNATPrivateAddress).(string)
	publicIP := propertyHelper.GetOptionalString(resourceKeyNATPublicAddress, false)

	publicIPDescription := "<computed>"
	if publicIP != nil {
		publicIPDescription = *publicIP
	}
	log.Printf("Create NAT rule (from public IP '%s' to private IP '%s') in network domain '%s'.", publicIPDescription, privateIP, networkDomainID)

	providerState := provider.(*providerState)
	apiClient := providerState.Client()

	domainLock := providerState.GetDomainLock(networkDomainID, "resourceNATCreate(%s -> %s)", privateIP, publicIPDescription)
	domainLock.Lock()
	defer domainLock.Unlock()

	// First, work out if we have any free public IP addresses.
	freeIPs := newStringSet()

	// Public IPs are allocated in blocks.
	publicIPBlocks, err := apiClient.ListPublicIPBlocks(networkDomainID)
	if err != nil {
		return err
	}
	var blockAddresses []string
	for _, block := range publicIPBlocks.Blocks {
		blockAddresses, err = calculateBlockAddresses(block)
		if err != nil {
			return err
		}

		for _, address := range blockAddresses {
			freeIPs.Add(address)
		}
	}

	// Some of those IPs may be reserved for other NAT rules or VIPs.
	reservedIPs, err := apiClient.ListReservedPublicIPAddresses(networkDomainID)
	if err != nil {
		return err
	}
	for _, reservedIP := range reservedIPs.IPs {
		freeIPs.Remove(reservedIP.Address)
	}

	// Anything left is free to use.
	// Note that there is still potentially a race condition here. Improved behaviour would be to handle the relevant error response from CreateNATRule and retry.

	// If there are no free public IP's we'll need to request the allocation of a new block.
	if freeIPs.Len() == 0 {
		log.Printf("There are no free public IPv4 addresses in network domain '%s'; requesting allocation of a new address block...", networkDomainID)

		var blockID string
		blockID, err = apiClient.AddPublicIPBlock(networkDomainID)
		if err != nil {
			return err
		}

		var block *compute.PublicIPBlock
		block, err = apiClient.GetPublicIPBlock(blockID)
		if err != nil {
			return err
		}

		if block == nil {
			return fmt.Errorf("Cannot find newly-added public IPv4 address block '%s'.", blockID)
		}

		log.Printf("Allocated a new public IPv4 address block '%s' (%d addresses, starting at '%s').", block.ID, block.Size, block.BaseIP)
	}

	natRuleID, err := apiClient.AddNATRule(networkDomainID, privateIP, publicIP)
	if err != nil {
		return err
	}

	data.SetId(natRuleID)
	log.Printf("Successfully created NAT rule '%s'.", natRuleID)

	natRule, err := apiClient.GetNATRule(natRuleID)
	if err != nil {
		return err
	}

	if natRule == nil {
		return fmt.Errorf("Cannot find newly-added NAT rule '%s'.", natRuleID)
	}

	data.Set(resourceKeyNATPublicAddress, natRule.ExternalIPAddress)

	return nil
}

// Read a NAT resource.
func resourceNATRead(data *schema.ResourceData, provider interface{}) error {
	id := data.Id()
	networkDomainID := data.Get(resourceKeyNATNetworkDomainID).(string)
	privateIP := data.Get(resourceKeyNATPrivateAddress).(string)
	publicIP := data.Get(resourceKeyNATPublicAddress).(string)

	log.Printf("Read NAT '%s' (private IP = '%s', public IP = '%s') in network domain '%s'.", id, privateIP, publicIP, networkDomainID)

	apiClient := provider.(*providerState).Client()
	apiClient.Reset() // TODO: Replace call to Reset with appropriate API call(s).

	return nil
}

// Update a NAT resource.
func resourceNATUpdate(data *schema.ResourceData, provider interface{}) error {
	id := data.Id()
	networkDomainID := data.Get(resourceKeyNATNetworkDomainID).(string)
	privateIP := data.Get(resourceKeyNATPrivateAddress).(string)
	publicIP := data.Get(resourceKeyNATPublicAddress).(string)

	log.Printf("Update NAT '%s' (private IP = '%s', public IP = '%s') in network domain '%s'.", id, privateIP, publicIP, networkDomainID)

	providerState := provider.(*providerState)
	apiClient := providerState.Client()
	apiClient.Reset() // TODO: Replace call to Reset with appropriate API call(s).

	domainLock := providerState.GetDomainLock(networkDomainID, "resourceNATUpdate(%s -> %s)", privateIP, publicIP)
	domainLock.Lock()
	defer domainLock.Unlock()

	return nil
}

// Delete a NAT resource.
func resourceNATDelete(data *schema.ResourceData, provider interface{}) error {
	id := data.Id()
	networkDomainID := data.Get(resourceKeyNATNetworkDomainID).(string)
	privateIP := data.Get(resourceKeyNATPrivateAddress).(string)
	publicIP := data.Get(resourceKeyNATPublicAddress).(string)

	log.Printf("Delete NAT '%s' (private IP = '%s', public IP = '%s') in network domain '%s'.", id, privateIP, publicIP, networkDomainID)

	providerState := provider.(*providerState)
	apiClient := providerState.Client()

	domainLock := providerState.GetDomainLock(networkDomainID, "resourceNATDelete(%s -> %s)", privateIP, publicIP)
	domainLock.Lock()
	defer domainLock.Unlock()

	return apiClient.DeleteNATRule(id)
}

func calculateBlockAddresses(block compute.PublicIPBlock) ([]string, error) {
	addresses := make([]string, block.Size)

	baseAddressComponents := strings.Split(block.BaseIP, ".")
	if len(baseAddressComponents) != 4 {
		return addresses, fmt.Errorf("Invalid base IP address '%s'.", block.BaseIP)
	}
	baseOctet, err := strconv.Atoi(baseAddressComponents[3])
	if err != nil {
		return addresses, err
	}

	for index := range addresses {
		// Increment the last octet to determine the next address in the block.
		baseAddressComponents[3] = strconv.Itoa(baseOctet + index)
		addresses[index] = strings.Join(baseAddressComponents, ".")
	}

	return addresses, nil
}
