package azurerm

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceArmPublicIp() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmPublicIpCreate,
		Read:   resourceArmPublicIpRead,
		Update: resourceArmPublicIpCreate,
		Delete: resourceArmPublicIpDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"location": locationSchema(),

			"resource_group_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"public_ip_address_allocation": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateFunc:     validatePublicIpAllocation,
				StateFunc:        ignoreCaseStateFunc,
				DiffSuppressFunc: ignoreCaseDiffSuppressFunc,
			},

			"idle_timeout_in_minutes": {
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

			"domain_name_label": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validatePublicIpDomainNameLabel,
			},

			"reverse_fqdn": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"fqdn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"ip_address": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"tags": tagsSchema(),
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
	tags := d.Get("tags").(map[string]interface{})

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
		idle_timeout := int32(v.(int))
		properties.IdleTimeoutInMinutes = &idle_timeout
	}

	publicIp := network.PublicIPAddress{
		Name:                            &name,
		Location:                        &location,
		PublicIPAddressPropertiesFormat: &properties,
		Tags: expandTags(tags),
	}

	_, error := publicIPClient.CreateOrUpdate(resGroup, name, publicIp, make(chan struct{}))
	err := <-error
	if err != nil {
		return err
	}

	read, err := publicIPClient.Get(resGroup, name, "")
	if err != nil {
		return err
	}
	if read.ID == nil {
		return fmt.Errorf("Cannot read Public IP %s (resource group %s) ID", name, resGroup)
	}

	d.SetId(*read.ID)

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

	resp, err := publicIPClient.Get(resGroup, name, "")
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error making Read request on Azure public ip %s: %s", name, err)
	}

	d.Set("resource_group_name", resGroup)
	d.Set("location", resp.Location)
	d.Set("name", resp.Name)
	d.Set("public_ip_address_allocation", strings.ToLower(string(resp.PublicIPAddressPropertiesFormat.PublicIPAllocationMethod)))

	if resp.PublicIPAddressPropertiesFormat.DNSSettings != nil && resp.PublicIPAddressPropertiesFormat.DNSSettings.Fqdn != nil && *resp.PublicIPAddressPropertiesFormat.DNSSettings.Fqdn != "" {
		d.Set("fqdn", resp.PublicIPAddressPropertiesFormat.DNSSettings.Fqdn)
	}

	if resp.PublicIPAddressPropertiesFormat.IPAddress != nil && *resp.PublicIPAddressPropertiesFormat.IPAddress != "" {
		d.Set("ip_address", resp.PublicIPAddressPropertiesFormat.IPAddress)
	}

	flattenAndSetTags(d, resp.Tags)

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

	_, error := publicIPClient.Delete(resGroup, name, make(chan struct{}))
	err = <-error

	return err
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
