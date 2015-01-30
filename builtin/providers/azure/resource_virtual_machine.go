package azure

import (
	"bytes"
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/MSOpenTech/azure-sdk-for-go/clients/vmClient"
)

func resourceVirtualMachine() *schema.Resource {
	return &schema.Resource{
		Create: resourceVirtualMachineCreate,
		Read:   resourceVirtualMachineRead,
		Delete: resourceVirtualMachineDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"location": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"image": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"size": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"username": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"password": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
				ForceNew: true,
			},

			"ssh_public_key_file": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
				ForceNew: true,
			},

			"ssh_port": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  22,
				ForceNew: true,
			},

			"endpoint": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				ForceNew: true, // This can be updatable once we support updates on the resource
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"protocol": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"port": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},

						"local_port": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
					},
				},
				Set: resourceVirtualMachineEndpointHash,
			},

			"url": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"ip_address": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"vip_address": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceVirtualMachineCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Creating Azure Virtual Machine Configuration...")
	vmConfig, err := vmClient.CreateAzureVMConfiguration(
		d.Get("name").(string),
		d.Get("size").(string),
		d.Get("image").(string),
		d.Get("location").(string))
	if err != nil {
		return fmt.Errorf("Error creating Azure virtual machine configuration: %s", err)
	}

	// Only Linux VMs are supported. If we want to support other VM types, we need to
	// grab the image details and based on the OS add the corresponding configuration.
	log.Printf("[DEBUG] Adding Azure Linux Provisioning Configuration...")
	vmConfig, err = vmClient.AddAzureLinuxProvisioningConfig(
		vmConfig,
		d.Get("username").(string),
		d.Get("password").(string),
		d.Get("ssh_public_key_file").(string),
		d.Get("ssh_port").(int))
	if err != nil {
		return fmt.Errorf("Error adding Azure linux provisioning configuration: %s", err)
	}

	if v := d.Get("endpoint").(*schema.Set); v.Len() > 0 {
		log.Printf("[DEBUG] Adding Endpoints to the Azure Virtual Machine...")
		endpoints := make([]vmClient.InputEndpoint, v.Len())
		for i, v := range v.List() {
			m := v.(map[string]interface{})
			endpoint := vmClient.InputEndpoint{}
			endpoint.Name = m["name"].(string)
			endpoint.Protocol = m["protocol"].(string)
			endpoint.Port = m["port"].(int)
			endpoint.LocalPort = m["local_port"].(int)
			endpoints[i] = endpoint
		}

		configSets := vmConfig.ConfigurationSets.ConfigurationSet
		if len(configSets) == 0 {
			return fmt.Errorf("Azure virtual machine does not have configuration sets")
		}
		for i := 0; i < len(configSets); i++ {
			if configSets[i].ConfigurationSetType != "NetworkConfiguration" {
				continue
			}
			configSets[i].InputEndpoints.InputEndpoint =
				append(configSets[i].InputEndpoints.InputEndpoint, endpoints...)
		}
	}

	log.Printf("[DEBUG] Creating Azure Virtual Machine...")
	err = vmClient.CreateAzureVM(
		vmConfig,
		d.Get("name").(string),
		d.Get("location").(string))
	if err != nil {
		return fmt.Errorf("Error creating Azure virtual machine: %s", err)
	}

	d.SetId(d.Get("name").(string))

	return resourceVirtualMachineRead(d, meta)
}

func resourceVirtualMachineRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Getting Azure Virtual Machine Deployment: %s", d.Id())
	VMDeployment, err := vmClient.GetVMDeployment(d.Id(), d.Id())
	if err != nil {
		return fmt.Errorf("Error getting Azure virtual machine deployment: %s", err)
	}

	d.Set("url", VMDeployment.Url)

	roleInstances := VMDeployment.RoleInstanceList.RoleInstance
	if len(roleInstances) == 0 {
		return fmt.Errorf("Virtual Machine does not have IP addresses")
	}
	ipAddress := roleInstances[0].IpAddress
	d.Set("ip_address", ipAddress)

	vips := VMDeployment.VirtualIPs.VirtualIP
	if len(vips) == 0 {
		return fmt.Errorf("Virtual Machine does not have VIP addresses")
	}
	vip := vips[0].Address
	d.Set("vip_address", vip)

	d.SetConnInfo(map[string]string{
		"type": "ssh",
		"host": vip,
		"user": d.Get("username").(string),
	})

	return nil
}

func resourceVirtualMachineDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Deleting Azure Virtual Machine Deployment: %s", d.Id())
	if err := vmClient.DeleteVMDeployment(d.Id(), d.Id()); err != nil {
		return fmt.Errorf("Error deleting Azure virtual machine deployment: %s", err)
	}

	log.Printf("[DEBUG] Deleting Azure Hosted Service: %s", d.Id())
	if err := vmClient.DeleteHostedService(d.Id()); err != nil {
		return fmt.Errorf("Error deleting Azure hosted service: %s", err)
	}

	d.SetId("")

	return nil
}

func resourceVirtualMachineEndpointHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["name"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["protocol"].(string)))
	buf.WriteString(fmt.Sprintf("%d-", m["port"].(int)))
	buf.WriteString(fmt.Sprintf("%d-", m["local_port"].(int)))

	return hashcode.String(buf.String())
}
