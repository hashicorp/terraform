package azurerm

import (
	"fmt"
	"log"
	"time"

	"github.com/Azure/azure-sdk-for-go/arm/compute"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/jen20/riviera/azure"
)

func resourceArmVirtualMachineExtensions() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmVirtualMachineExtensionsCreate,
		Read:   resourceArmVirtualMachineExtensionsRead,
		Update: resourceArmVirtualMachineExtensionsCreate,
		Delete: resourceArmVirtualMachineExtensionsDelete,

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

			"virtual_machine_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"publisher": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"type_handler_version": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"settings": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
			},

			//"command_to_execute": &schema.Schema{
			//	Type:     schema.TypeString,
			//	Optional: true,
			//},
			//
			//"file_uris": &schema.Schema{
			//	Type:     schema.TypeSet,
			//	Optional: true,
			//	Computed: true,
			//	Elem:     &schema.Schema{Type: schema.TypeString},
			//	Set:      schema.HashString,
			//},
		},
	}
}

func resourceArmVirtualMachineExtensionsCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	vmClient := client.vmExtensionClient

	name := d.Get("name").(string)
	vmName := d.Get("virtual_machine_name").(string)
	resGroup := d.Get("resource_group_name").(string)
	settings := d.Get("settings").(map[string]interface{})

	extensionParams := compute.VirtualMachineExtension{
		Location: azure.String(d.Get("location").(string)),
		Properties: &compute.VirtualMachineExtensionProperties{
			Type:               azure.String(d.Get("type").(string)),
			Publisher:          azure.String(d.Get("publisher").(string)),
			TypeHandlerVersion: azure.String(d.Get("type_handler_version").(string)),
			Settings:           expandAzureRMVirtualMachineExtensionSettings(settings),
		},
	}

	resp, err := vmClient.CreateOrUpdate(resGroup, vmName, name, extensionParams)
	if err != nil {
		return err
	}

	d.SetId(*resp.ID)

	log.Printf("[DEBUG] Waiting for Virtual Machine Extension (%s) to become available", name)
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"Creating", "Updating"},
		Target:     []string{"Succeeded"},
		Refresh:    virtualMachineExtensionStateRefreshFunc(client, resGroup, vmName, name),
		Timeout:    10 * time.Minute,
		MinTimeout: 5 * time.Second,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf("Error waiting for Virtual Machine Extension (%s) to become available: %s", name, err)
	}

	return resourceArmVirtualMachineExtensionsRead(d, meta)
}

func resourceArmVirtualMachineExtensionsRead(d *schema.ResourceData, meta interface{}) error {
	//client := meta.(*ArmClient)
	//vmClient := client.vmExtensionClient

	return nil
}

func resourceArmVirtualMachineExtensionsDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	vmClient := client.vmExtensionClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["extensions"]
	vmName := id.Path["virtualMachines"]

	_, err = vmClient.Delete(resGroup, vmName, name)

	return nil
}

func virtualMachineExtensionStateRefreshFunc(client *ArmClient, resourceGroupName string, vmName string, extensionName string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		res, err := client.vmExtensionClient.Get(resourceGroupName, vmName, extensionName, "")
		if err != nil {
			return nil, "", fmt.Errorf("Error issuing read request in virtualMachineExtensionStateRefreshFunc to Azure ARM for Virtual Machine Extension '%s' (RG: '%s'): %s", extensionName, resourceGroupName, err)
		}

		return res, *res.Properties.ProvisioningState, nil
	}
}

func expandAzureRMVirtualMachineExtensionSettings(settings map[string]interface{}) *map[string]interface{} {
	output := make(map[string]interface{}, len(settings))

	for i, v := range settings {
		value, _ := tagValueToString(v)
		output[i] = value
	}

	return &output
}
