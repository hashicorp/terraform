package azurerm

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceArmNetworkInterface() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmNetworkInterfaceCreate,
		Read:   resourceArmNetworkInterfaceRead,
		Update: resourceArmNetworkInterfaceCreate,
		Delete: resourceArmNetworkInterfaceDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"location": &schema.Schema{
				Type:      schema.TypeString,
				Required:  true,
				ForceNew:  true,
				StateFunc: azureRMNormalizeLocation,
			},

			"resource_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"network_security_group_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"mac_address": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"private_ip_address": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"virtual_machine_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"ip_configuration": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"subnet_id": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"private_ip_address": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},

						"private_ip_address_allocation": &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateNetworkInterfacePrivateIpAddressAllocation,
						},

						"public_ip_address_id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},

						"load_balancer_backend_address_pools_ids": &schema.Schema{
							Type:     schema.TypeSet,
							Optional: true,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},

						"load_balancer_inbound_nat_rules_ids": &schema.Schema{
							Type:     schema.TypeSet,
							Optional: true,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
					},
				},
				Set: resourceArmNetworkInterfaceIpConfigurationHash,
			},

			"dns_servers": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"internal_dns_name_label": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"applied_dns_servers": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"internal_fqdn": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceArmNetworkInterfaceCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	ifaceClient := client.ifaceClient

	log.Printf("[INFO] preparing arguments for Azure ARM Network Interface creation.")

	name := d.Get("name").(string)
	location := d.Get("location").(string)
	resGroup := d.Get("resource_group_name").(string)
	tags := d.Get("tags").(map[string]interface{})

	properties := network.InterfacePropertiesFormat{}

	if v, ok := d.GetOk("network_security_group_id"); ok {
		nsgId := v.(string)
		properties.NetworkSecurityGroup = &network.SecurityGroup{
			ID: &nsgId,
		}
	}

	dns, hasDns := d.GetOk("dns_servers")
	nameLabel, hasNameLabel := d.GetOk("internal_dns_name_label")
	if hasDns || hasNameLabel {
		ifaceDnsSettings := network.InterfaceDNSSettings{}

		if hasDns {
			var dnsServers []string
			dns := dns.(*schema.Set).List()
			for _, v := range dns {
				str := v.(string)
				dnsServers = append(dnsServers, str)
			}
			ifaceDnsSettings.DNSServers = &dnsServers
		}

		if hasNameLabel {
			name_label := nameLabel.(string)
			ifaceDnsSettings.InternalDNSNameLabel = &name_label

		}

		properties.DNSSettings = &ifaceDnsSettings
	}

	ipConfigs, sgErr := expandAzureRmNetworkInterfaceIpConfigurations(d)
	if sgErr != nil {
		return fmt.Errorf("Error Building list of Network Interface IP Configurations: %s", sgErr)
	}
	if len(ipConfigs) > 0 {
		properties.IPConfigurations = &ipConfigs
	}

	iface := network.Interface{
		Name:       &name,
		Location:   &location,
		Properties: &properties,
		Tags:       expandTags(tags),
	}

	resp, err := ifaceClient.CreateOrUpdate(resGroup, name, iface)
	if err != nil {
		return err
	}

	d.SetId(*resp.ID)

	log.Printf("[DEBUG] Waiting for Network Interface (%s) to become available", name)
	stateConf := &resource.StateChangeConf{
		Pending: []string{"Accepted", "Updating"},
		Target:  []string{"Succeeded"},
		Refresh: networkInterfaceStateRefreshFunc(client, resGroup, name),
		Timeout: 10 * time.Minute,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf("Error waiting for Network Interface (%s) to become available: %s", name, err)
	}

	return resourceArmNetworkInterfaceRead(d, meta)
}

func resourceArmNetworkInterfaceRead(d *schema.ResourceData, meta interface{}) error {
	ifaceClient := meta.(*ArmClient).ifaceClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["networkInterfaces"]

	resp, err := ifaceClient.Get(resGroup, name, "")
	if resp.StatusCode == http.StatusNotFound {
		d.SetId("")
		return nil
	}
	if err != nil {
		return fmt.Errorf("Error making Read request on Azure Network Interface %s: %s", name, err)
	}

	iface := *resp.Properties

	if iface.MacAddress != nil {
		if *iface.MacAddress != "" {
			d.Set("mac_address", iface.MacAddress)
		}
	}

	if iface.IPConfigurations != nil && len(*iface.IPConfigurations) > 0 {
		var privateIPAddress *string
		///TODO: Change this to a loop when https://github.com/Azure/azure-sdk-for-go/issues/259 is fixed
		if (*iface.IPConfigurations)[0].Properties != nil {
			privateIPAddress = (*iface.IPConfigurations)[0].Properties.PrivateIPAddress
		}

		if *privateIPAddress != "" {
			d.Set("private_ip_address", *privateIPAddress)
		}
	}

	if iface.VirtualMachine != nil {
		if *iface.VirtualMachine.ID != "" {
			d.Set("virtual_machine_id", *iface.VirtualMachine.ID)
		}
	}

	if iface.DNSSettings != nil {
		if iface.DNSSettings.AppliedDNSServers != nil && len(*iface.DNSSettings.AppliedDNSServers) > 0 {
			dnsServers := make([]string, 0, len(*iface.DNSSettings.AppliedDNSServers))
			for _, dns := range *iface.DNSSettings.AppliedDNSServers {
				dnsServers = append(dnsServers, dns)
			}

			if err := d.Set("applied_dns_servers", dnsServers); err != nil {
				return err
			}
		}

		if iface.DNSSettings.InternalFqdn != nil && *iface.DNSSettings.InternalFqdn != "" {
			d.Set("internal_fqdn", iface.DNSSettings.InternalFqdn)
		}
	}

	flattenAndSetTags(d, resp.Tags)

	return nil
}

func resourceArmNetworkInterfaceDelete(d *schema.ResourceData, meta interface{}) error {
	ifaceClient := meta.(*ArmClient).ifaceClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["networkInterfaces"]

	_, err = ifaceClient.Delete(resGroup, name)

	return err
}

func networkInterfaceStateRefreshFunc(client *ArmClient, resourceGroupName string, ifaceName string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		res, err := client.ifaceClient.Get(resourceGroupName, ifaceName, "")
		if err != nil {
			return nil, "", fmt.Errorf("Error issuing read request in networkInterfaceStateRefreshFunc to Azure ARM for network interace '%s' (RG: '%s'): %s", ifaceName, resourceGroupName, err)
		}

		return res, *res.Properties.ProvisioningState, nil
	}
}

func resourceArmNetworkInterfaceIpConfigurationHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["name"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["subnet_id"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["private_ip_address_allocation"].(string)))

	return hashcode.String(buf.String())
}

func validateNetworkInterfacePrivateIpAddressAllocation(v interface{}, k string) (ws []string, errors []error) {
	value := strings.ToLower(v.(string))
	allocations := map[string]bool{
		"static":  true,
		"dynamic": true,
	}

	if !allocations[value] {
		errors = append(errors, fmt.Errorf("Network Interface Allocations can only be Static or Dynamic"))
	}
	return
}

func expandAzureRmNetworkInterfaceIpConfigurations(d *schema.ResourceData) ([]network.InterfaceIPConfiguration, error) {
	configs := d.Get("ip_configuration").(*schema.Set).List()
	ipConfigs := make([]network.InterfaceIPConfiguration, 0, len(configs))

	for _, configRaw := range configs {
		data := configRaw.(map[string]interface{})

		subnet_id := data["subnet_id"].(string)
		private_ip_allocation_method := data["private_ip_address_allocation"].(string)

		properties := network.InterfaceIPConfigurationPropertiesFormat{
			Subnet: &network.Subnet{
				ID: &subnet_id,
			},
			PrivateIPAllocationMethod: &private_ip_allocation_method,
		}

		if v := data["private_ip_address"].(string); v != "" {
			properties.PrivateIPAddress = &v
		}

		if v := data["public_ip_address_id"].(string); v != "" {
			properties.PublicIPAddress = &network.PublicIPAddress{
				ID: &v,
			}
		}

		if v, ok := data["load_balancer_backend_address_pools_ids"]; ok {
			var ids []network.BackendAddressPool
			pools := v.(*schema.Set).List()
			for _, p := range pools {
				pool_id := p.(string)
				id := network.BackendAddressPool{
					ID: &pool_id,
				}

				ids = append(ids, id)
			}

			properties.LoadBalancerBackendAddressPools = &ids
		}

		if v, ok := data["load_balancer_inbound_nat_rules_ids"]; ok {
			var natRules []network.InboundNatRule
			rules := v.(*schema.Set).List()
			for _, r := range rules {
				rule_id := r.(string)
				rule := network.InboundNatRule{
					ID: &rule_id,
				}

				natRules = append(natRules, rule)
			}

			properties.LoadBalancerInboundNatRules = &natRules
		}

		name := data["name"].(string)
		ipConfig := network.InterfaceIPConfiguration{
			Name:       &name,
			Properties: &properties,
		}

		ipConfigs = append(ipConfigs, ipConfig)
	}

	return ipConfigs, nil
}
