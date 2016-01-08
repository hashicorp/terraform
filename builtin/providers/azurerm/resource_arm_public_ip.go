package azurerm

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceArmPublicIp() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmPublicIpCreate,
		Read:   resourceArmPublicIpRead,
		Update: resourceArmPublicIpCreate,
		Delete: resourceArmPublicIpDelete,

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

			"public_ip_address_allocation": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validatePublicIpAllocation,
			},

			"idle_timeout_in_minutes": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(int)
					if value < 4 || value > 30 {
						errors = append(errors, fmt.Errorf(
							"The idle timeout must be between 4 and 30 minutes"))
					}
					return
				},
			},

			"domain_name_label": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validatePublicIpDomainNameLabel,
			},

			"reverse_fqdn": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"fqdn": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"ip_address": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceArmPublicIpCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	publicIPClient := client.publicIPClient

	log.Printf("[INFO] preparing arguments for Azure ARM Public IP creation.")

	name := d.Get("name").(string)
	location := d.Get("location").(string)
	resGroup := d.Get("resource_group_name").(string)

	properties := network.PublicIPAddressPropertiesFormat{
		PublicIPAllocationMethod: network.IPAllocationMethod(d.Get("public_ip_address_allocation").(string)),
	}

	dnl, hasDnl := d.GetOk("domain_name_label")
	rfqdn, hasRfqdn := d.GetOk("reverse_fqdn")

	if hasDnl || hasRfqdn {
		dnsSettings := network.PublicIPAddressDNSSettings{}

		if hasRfqdn {
			reverse_fqdn := rfqdn.(string)
			dnsSettings.ReverseFqdn = &reverse_fqdn
		}

		if hasDnl {
			domain_name_label := dnl.(string)
			dnsSettings.DomainNameLabel = &domain_name_label

		}

		properties.DNSSettings = &dnsSettings
	}

	if v, ok := d.GetOk("idle_timeout_in_minutes"); ok {
		idle_timeout := v.(int)
		properties.IdleTimeoutInMinutes = &idle_timeout
	}

	publicIp := network.PublicIPAddress{
		Name:       &name,
		Location:   &location,
		Properties: &properties,
	}

	resp, err := publicIPClient.CreateOrUpdate(resGroup, name, publicIp)
	if err != nil {
		return err
	}

	d.SetId(*resp.ID)

	log.Printf("[DEBUG] Waiting for Public IP (%s) to become available", name)
	stateConf := &resource.StateChangeConf{
		Pending: []string{"Accepted", "Updating"},
		Target:  "Succeeded",
		Refresh: publicIPStateRefreshFunc(client, resGroup, name),
		Timeout: 10 * time.Minute,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf("Error waiting for Public IP (%s) to become available: %s", name, err)
	}

	return resourceArmPublicIpRead(d, meta)
}

func resourceArmPublicIpRead(d *schema.ResourceData, meta interface{}) error {
	publicIPClient := meta.(*ArmClient).publicIPClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["publicIPAddresses"]

	resp, err := publicIPClient.Get(resGroup, name)
	if resp.StatusCode == http.StatusNotFound {
		d.SetId("")
		return nil
	}
	if err != nil {
		return fmt.Errorf("Error making Read request on Azure public ip %s: %s", name, err)
	}

	if resp.Properties.DNSSettings != nil && resp.Properties.DNSSettings.Fqdn != nil && *resp.Properties.DNSSettings.Fqdn != "" {
		d.Set("fqdn", resp.Properties.DNSSettings.Fqdn)
	}

	if resp.Properties.IPAddress != nil && *resp.Properties.IPAddress != "" {
		d.Set("ip_address", resp.Properties.IPAddress)
	}

	return nil
}

func resourceArmPublicIpDelete(d *schema.ResourceData, meta interface{}) error {
	publicIPClient := meta.(*ArmClient).publicIPClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["publicIPAddresses"]

	_, err = publicIPClient.Delete(resGroup, name)

	return err
}

func publicIPStateRefreshFunc(client *ArmClient, resourceGroupName string, publicIpName string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		res, err := client.publicIPClient.Get(resourceGroupName, publicIpName)
		if err != nil {
			return nil, "", fmt.Errorf("Error issuing read request in publicIPStateRefreshFunc to Azure ARM for public ip '%s' (RG: '%s'): %s", publicIpName, resourceGroupName, err)
		}

		return res, *res.Properties.ProvisioningState, nil
	}
}

func validatePublicIpAllocation(v interface{}, k string) (ws []string, errors []error) {
	value := strings.ToLower(v.(string))
	allocations := map[string]bool{
		"static":  true,
		"dynamic": true,
	}

	if !allocations[value] {
		errors = append(errors, fmt.Errorf("Public IP Allocation can only be Static of Dynamic"))
	}
	return
}

func validatePublicIpDomainNameLabel(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if !regexp.MustCompile(`^[a-z0-9-]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"only alphanumeric characters and hyphens allowed in %q: %q",
			k, value))
	}

	if len(value) > 61 {
		errors = append(errors, fmt.Errorf(
			"%q cannot be longer than 61 characters: %q", k, value))
	}

	if len(value) == 0 {
		errors = append(errors, fmt.Errorf(
			"%q cannot be an empty string: %q", k, value))
	}
	if regexp.MustCompile(`-$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q cannot end with a hyphen: %q", k, value))
	}

	return

}
