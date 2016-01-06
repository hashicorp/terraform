package azurerm

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

// resourceArmPublicIP returns the *schema.Resource
// associated to a public i p resources on ARM.
func resourceArmPublicIP() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmPublicIPCreate,
		Read:   resourceArmPublicIPRead,
		Update: resourceArmPublicIPCreate,
		Delete: resourceArmPublicIPDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"resource_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"location": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			// the default allocation method:
			"dynamic_ip": &schema.Schema{
				Type:          schema.TypeBool,
				Optional:      true,
				Default:       true,
				ConflictsWith: []string{"ip_address"},
			},

			"ip_address": &schema.Schema{
				Type: schema.TypeString,
				// required only when 'dynamic_provate_ip' is NOT set.
				Optional:      true,
				ConflictsWith: []string{"dynamic_ip"},
			},

			"ip_config_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"timeout": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},

			"dns_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"fqdn": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"reverse_fqdn": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

// resourceArmPublicIPCreate goes ahead and creates the specified ARM public i p.
func resourceArmPublicIPCreate(d *schema.ResourceData, meta interface{}) error {
	publicIPClient := meta.(*ArmClient).publicIPClient

	// get the standard params:
	name := d.Get("name").(string)
	resGrp := d.Get("resGrp").(string)
	dnsName := d.Get("dns_name").(string)
	location := d.Get("location").(string)
	ipConfId := d.Get("ip_config_id").(string)

	// get the IP allocation method or the fixed IP address:
	var addr string
	var allocMeth network.IPAllocationMethod
	if dyn, ok := d.GetOk("dynamic_ip"); ok && dyn.(bool) {
		addr = ""
		allocMeth = network.Dynamic
	} else {
		if add, ok := d.GetOk("ip_address"); ok {
			addr = add.(string)
			allocMeth = network.Static
		} else {
			return fmt.Errorf("Error in public IP definition: 'ip_address' must be provided if 'dynamic_ip' is not set.")
		}
	}

	// now get the timeout:
	var timeout int
	if t, ok := d.GetOk("timeout"); ok {
		timeout = t.(int)
	}

	resp, err := publicIPClient.CreateOrUpdate(resGrp, name, network.PublicIPAddress{
		Name:     &name,
		Location: &location,
		Properties: &network.PublicIPAddressPropertiesFormat{
			PublicIPAllocationMethod: allocMeth,
			IPConfiguration:          &network.SubResource{&ipConfId},
			IPAddress:                &addr,
			IdleTimeoutInMinutes:     &timeout,
			DNSSettings: &network.PublicIPAddressDNSSettings{
				DomainNameLabel: &dnsName,
			},
		},
	})

	if err != nil {
		return fmt.Errorf("Error creating ARM public IP address %q: %s", name, err)
	}

	d.SetId(*resp.ID)

	stateConf := &resource.StateChangeConf{
		Pending: []string{"Accepted", "Updating"},
		Target:  "Succeded",
		Refresh: publicIPStateRefreshFunc(meta, name, resGrp),
		Timeout: 10 * time.Minute,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf("Error waiting for ARM public IP %q creation: %s", name, err)
	}

	return resourceArmPublicIPRead(d, meta)
}

// resourceArmPublicIPRead goes ahead and reads the state of the corresponding ARM public i p.
func resourceArmPublicIPRead(d *schema.ResourceData, meta interface{}) error {
	publicIPClient := meta.(*ArmClient).publicIPClient

	// parse the ID for the name of the resource
	// and that of the resource group:
	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return fmt.Errorf("Error parsing the id of ARM public IP: %s")
	}
	name := id.Path["publicIPAddresses"]
	resGrp := id.ResourceGroup

	// make a query to Azure:
	pip, err := publicIPClient.Get(resGrp, name)
	if pip.StatusCode == http.StatusNotFound {
		d.SetId("")
		return nil
	}
	if err != nil {
		return fmt.Errorf("Error reading the state off Azure for public IP %q: %s", name, err)
	}

	props := pip.Properties

	// start reading fields:
	d.Set("ip_address", *props.IPAddress)
	d.Set("dynamic_ip", props.PublicIPAllocationMethod == network.Dynamic)
	d.Set("ip_config_id", *props.IPConfiguration.ID)
	d.Set("timeout", *props.IdleTimeoutInMinutes)
	d.Set("dns_name", *props.DNSSettings.DomainNameLabel)
	d.Set("fqdn", *props.DNSSettings.ReverseFqdn)
	d.Set("reverse_fqdn", *props.DNSSettings.ReverseFqdn)

	return nil
}

// resourceArmPublicIPDelete deletes the specified ARM public i p.
func resourceArmPublicIPDelete(d *schema.ResourceData, meta interface{}) error {
	publicIPClient := meta.(*ArmClient).publicIPClient

	// first; parse the resource ID for the name of the public IP resource as
	// well as that of its containing resurce group:
	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return fmt.Errorf("Error parsing public IP resource ID: %s", err)
	}
	name := id.Path["publicIPAddresses"]
	resGrp := id.ResourceGroup

	// issue the actual deletion:
	resp, err := publicIPClient.Delete(resGrp, name)
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if err != nil {
		return fmt.Errorf("Error issuing deletion of Azure public IP address %q: %s", name, err)
	}

	return nil
}

// publicIPStateRefreshFunc returns the resource.StateRefreshFunc for the
// given public IP resource under the given resource group.
func publicIPStateRefreshFunc(meta interface{}, name, resGrp string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		resp, err := meta.(*ArmClient).publicIPClient.Get(resGrp, name)
		if err != nil {
			return nil, "", err
		}

		return resp, *resp.Properties.ProvisioningState, nil
	}
}
