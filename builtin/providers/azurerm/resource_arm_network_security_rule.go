package azurerm

import (
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceArmNetworkSecurityRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmNetworkSecurityRuleCreate,
		Read:   resourceArmNetworkSecurityRuleRead,
		Update: resourceArmNetworkSecurityRuleCreate,
		Delete: resourceArmNetworkSecurityRuleDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"resource_group_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"network_security_group_name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"description": {
				Type:     schema.TypeString,
				Optional: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					if len(value) > 140 {
						errors = append(errors, fmt.Errorf(
							"The network security rule description can be no longer than 140 chars"))
					}
					return
				},
			},

			"protocol": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateNetworkSecurityRuleProtocol,
			},

			"source_port_range": {
				Type:     schema.TypeString,
				Required: true,
			},

			"destination_port_range": {
				Type:     schema.TypeString,
				Required: true,
			},

			"source_address_prefix": {
				Type:     schema.TypeString,
				Required: true,
			},

			"destination_address_prefix": {
				Type:     schema.TypeString,
				Required: true,
			},

			"access": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateNetworkSecurityRuleAccess,
			},

			"priority": {
				Type:     schema.TypeInt,
				Required: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(int)
					if value < 100 || value > 4096 {
						errors = append(errors, fmt.Errorf(
							"The `priority` can only be between 100 and 4096"))
					}
					return
				},
			},

			"direction": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateNetworkSecurityRuleDirection,
			},
		},
	}
}

func resourceArmNetworkSecurityRuleCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	secClient := client.secRuleClient

	name := d.Get("name").(string)
	nsgName := d.Get("network_security_group_name").(string)
	resGroup := d.Get("resource_group_name").(string)

	source_port_range := d.Get("source_port_range").(string)
	destination_port_range := d.Get("destination_port_range").(string)
	source_address_prefix := d.Get("source_address_prefix").(string)
	destination_address_prefix := d.Get("destination_address_prefix").(string)
	priority := int32(d.Get("priority").(int))
	access := d.Get("access").(string)
	direction := d.Get("direction").(string)
	protocol := d.Get("protocol").(string)

	armMutexKV.Lock(nsgName)
	defer armMutexKV.Unlock(nsgName)

	properties := network.SecurityRulePropertiesFormat{
		SourcePortRange:          &source_port_range,
		DestinationPortRange:     &destination_port_range,
		SourceAddressPrefix:      &source_address_prefix,
		DestinationAddressPrefix: &destination_address_prefix,
		Priority:                 &priority,
		Access:                   network.SecurityRuleAccess(access),
		Direction:                network.SecurityRuleDirection(direction),
		Protocol:                 network.SecurityRuleProtocol(protocol),
	}

	if v, ok := d.GetOk("description"); ok {
		description := v.(string)
		properties.Description = &description
	}

	sgr := network.SecurityRule{
		Name: &name,
		SecurityRulePropertiesFormat: &properties,
	}

	_, error := secClient.CreateOrUpdate(resGroup, nsgName, name, sgr, make(chan struct{}))
	err := <-error
	if err != nil {
		return err
	}

	read, err := secClient.Get(resGroup, nsgName, name)
	if err != nil {
		return err
	}
	if read.ID == nil {
		return fmt.Errorf("Cannot read Security Group Rule %s/%s (resource group %s) ID",
			nsgName, name, resGroup)
	}

	d.SetId(*read.ID)

	return resourceArmNetworkSecurityRuleRead(d, meta)
}

func resourceArmNetworkSecurityRuleRead(d *schema.ResourceData, meta interface{}) error {
	secRuleClient := meta.(*ArmClient).secRuleClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	networkSGName := id.Path["networkSecurityGroups"]
	sgRuleName := id.Path["securityRules"]

	resp, err := secRuleClient.Get(resGroup, networkSGName, sgRuleName)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error making Read request on Azure Network Security Rule %s: %s", sgRuleName, err)
	}

	d.Set("resource_group_name", resGroup)
	d.Set("access", resp.SecurityRulePropertiesFormat.Access)
	d.Set("destination_address_prefix", resp.SecurityRulePropertiesFormat.DestinationAddressPrefix)
	d.Set("destination_port_range", resp.SecurityRulePropertiesFormat.DestinationPortRange)
	d.Set("direction", resp.SecurityRulePropertiesFormat.Direction)
	d.Set("description", resp.SecurityRulePropertiesFormat.Description)
	d.Set("name", resp.Name)
	d.Set("priority", resp.SecurityRulePropertiesFormat.Priority)
	d.Set("protocol", resp.SecurityRulePropertiesFormat.Protocol)
	d.Set("source_address_prefix", resp.SecurityRulePropertiesFormat.SourceAddressPrefix)
	d.Set("source_port_range", resp.SecurityRulePropertiesFormat.SourcePortRange)

	return nil
}

func resourceArmNetworkSecurityRuleDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	secRuleClient := client.secRuleClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	nsgName := id.Path["networkSecurityGroups"]
	sgRuleName := id.Path["securityRules"]

	armMutexKV.Lock(nsgName)
	defer armMutexKV.Unlock(nsgName)

	_, error := secRuleClient.Delete(resGroup, nsgName, sgRuleName, make(chan struct{}))
	err = <-error

	return err
}
