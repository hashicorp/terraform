package azurerm

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceArmNetworkSecurityGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmNetworkSecurityGroupCreate,
		Read:   resourceArmNetworkSecurityGroupRead,
		Update: resourceArmNetworkSecurityGroupCreate,
		Delete: resourceArmNetworkSecurityGroupDelete,
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

			"security_rule": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
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
							StateFunc:    ignoreCaseStateFunc,
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
				},
				Set: resourceArmNetworkSecurityGroupRuleHash,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceArmNetworkSecurityGroupCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	secClient := client.secGroupClient

	name := d.Get("name").(string)
	location := d.Get("location").(string)
	resGroup := d.Get("resource_group_name").(string)
	tags := d.Get("tags").(map[string]interface{})

	sgRules, sgErr := expandAzureRmSecurityRules(d)
	if sgErr != nil {
		return fmt.Errorf("Error Building list of Network Security Group Rules: %s", sgErr)
	}

	sg := network.SecurityGroup{
		Name:     &name,
		Location: &location,
		SecurityGroupPropertiesFormat: &network.SecurityGroupPropertiesFormat{
			SecurityRules: &sgRules,
		},
		Tags: expandTags(tags),
	}

	_, error := secClient.CreateOrUpdate(resGroup, name, sg, make(chan struct{}))
	err := <-error
	if err != nil {
		return err
	}

	read, err := secClient.Get(resGroup, name, "")
	if err != nil {
		return err
	}
	if read.ID == nil {
		return fmt.Errorf("Cannot read Virtual Network %s (resource group %s) ID", name, resGroup)
	}

	log.Printf("[DEBUG] Waiting for NSG (%s) to become available", d.Get("name"))
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"Updating", "Creating"},
		Target:     []string{"Succeeded"},
		Refresh:    networkSecurityGroupStateRefreshFunc(client, resGroup, name),
		Timeout:    30 * time.Minute,
		MinTimeout: 15 * time.Second,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf("Error waiting for NSG (%s) to become available: %s", d.Get("name"), err)
	}

	d.SetId(*read.ID)

	return resourceArmNetworkSecurityGroupRead(d, meta)
}

func resourceArmNetworkSecurityGroupRead(d *schema.ResourceData, meta interface{}) error {
	secGroupClient := meta.(*ArmClient).secGroupClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["networkSecurityGroups"]

	resp, err := secGroupClient.Get(resGroup, name, "")
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error making Read request on Azure Network Security Group %s: %s", name, err)
	}

	if resp.SecurityGroupPropertiesFormat.SecurityRules != nil {
		d.Set("security_rule", flattenNetworkSecurityRules(resp.SecurityGroupPropertiesFormat.SecurityRules))
	}

	d.Set("resource_group_name", resGroup)
	d.Set("name", resp.Name)
	d.Set("location", resp.Location)
	flattenAndSetTags(d, resp.Tags)

	return nil
}

func resourceArmNetworkSecurityGroupDelete(d *schema.ResourceData, meta interface{}) error {
	secGroupClient := meta.(*ArmClient).secGroupClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["networkSecurityGroups"]

	_, error := secGroupClient.Delete(resGroup, name, make(chan struct{}))
	err = <-error

	return err
}

func resourceArmNetworkSecurityGroupRuleHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["protocol"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["source_port_range"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["destination_port_range"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["source_address_prefix"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["destination_address_prefix"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["access"].(string)))
	buf.WriteString(fmt.Sprintf("%d-", m["priority"].(int)))
	buf.WriteString(fmt.Sprintf("%s-", m["direction"].(string)))

	return hashcode.String(buf.String())
}

func flattenNetworkSecurityRules(rules *[]network.SecurityRule) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(*rules))
	for _, rule := range *rules {
		sgRule := make(map[string]interface{})
		sgRule["name"] = *rule.Name
		sgRule["destination_address_prefix"] = *rule.SecurityRulePropertiesFormat.DestinationAddressPrefix
		sgRule["destination_port_range"] = *rule.SecurityRulePropertiesFormat.DestinationPortRange
		sgRule["source_address_prefix"] = *rule.SecurityRulePropertiesFormat.SourceAddressPrefix
		sgRule["source_port_range"] = *rule.SecurityRulePropertiesFormat.SourcePortRange
		sgRule["priority"] = int(*rule.SecurityRulePropertiesFormat.Priority)
		sgRule["access"] = rule.SecurityRulePropertiesFormat.Access
		sgRule["direction"] = rule.SecurityRulePropertiesFormat.Direction
		sgRule["protocol"] = rule.SecurityRulePropertiesFormat.Protocol

		if rule.SecurityRulePropertiesFormat.Description != nil {
			sgRule["description"] = *rule.SecurityRulePropertiesFormat.Description
		}

		result = append(result, sgRule)
	}
	return result
}

func expandAzureRmSecurityRules(d *schema.ResourceData) ([]network.SecurityRule, error) {
	sgRules := d.Get("security_rule").(*schema.Set).List()
	rules := make([]network.SecurityRule, 0, len(sgRules))

	for _, sgRaw := range sgRules {
		data := sgRaw.(map[string]interface{})

		source_port_range := data["source_port_range"].(string)
		destination_port_range := data["destination_port_range"].(string)
		source_address_prefix := data["source_address_prefix"].(string)
		destination_address_prefix := data["destination_address_prefix"].(string)
		priority := int32(data["priority"].(int))

		properties := network.SecurityRulePropertiesFormat{
			SourcePortRange:          &source_port_range,
			DestinationPortRange:     &destination_port_range,
			SourceAddressPrefix:      &source_address_prefix,
			DestinationAddressPrefix: &destination_address_prefix,
			Priority:                 &priority,
			Access:                   network.SecurityRuleAccess(data["access"].(string)),
			Direction:                network.SecurityRuleDirection(data["direction"].(string)),
			Protocol:                 network.SecurityRuleProtocol(data["protocol"].(string)),
		}

		if v := data["description"].(string); v != "" {
			properties.Description = &v
		}

		name := data["name"].(string)
		rule := network.SecurityRule{
			Name: &name,
			SecurityRulePropertiesFormat: &properties,
		}

		rules = append(rules, rule)
	}

	return rules, nil
}

func networkSecurityGroupStateRefreshFunc(client *ArmClient, resourceGroupName string, sgName string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		res, err := client.secGroupClient.Get(resourceGroupName, sgName, "")
		if err != nil {
			return nil, "", fmt.Errorf("Error issuing read request in networkSecurityGroupStateRefreshFunc to Azure ARM for NSG '%s' (RG: '%s'): %s", sgName, resourceGroupName, err)
		}

		return res, *res.SecurityGroupPropertiesFormat.ProvisioningState, nil
	}
}
