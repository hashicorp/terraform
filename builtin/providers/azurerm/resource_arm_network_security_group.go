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

			"security_rule": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"description": &schema.Schema{
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

						"protocol": &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateNetworkSecurityRuleProtocol,
						},

						"source_port_range": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"destination_port_range": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"source_address_prefix": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"destination_address_prefix": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"access": &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateNetworkSecurityRuleAccess,
						},

						"priority": &schema.Schema{
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

						"direction": &schema.Schema{
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
		Properties: &network.SecurityGroupPropertiesFormat{
			SecurityRules: &sgRules,
		},
		Tags: expandTags(tags),
	}

	resp, err := secClient.CreateOrUpdate(resGroup, name, sg)
	if err != nil {
		return err
	}

	d.SetId(*resp.ID)

	log.Printf("[DEBUG] Waiting for Network Security Group (%s) to become available", name)
	stateConf := &resource.StateChangeConf{
		Pending: []string{"Accepted", "Updating"},
		Target:  []string{"Succeeded"},
		Refresh: securityGroupStateRefreshFunc(client, resGroup, name),
		Timeout: 10 * time.Minute,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf("Error waiting for Network Securty Group (%s) to become available: %s", name, err)
	}

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
	if resp.StatusCode == http.StatusNotFound {
		d.SetId("")
		return nil
	}
	if err != nil {
		return fmt.Errorf("Error making Read request on Azure Network Security Group %s: %s", name, err)
	}

	if resp.Properties.SecurityRules != nil {
		d.Set("security_rule", flattenNetworkSecurityRules(resp.Properties.SecurityRules))
	}

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

	_, err = secGroupClient.Delete(resGroup, name)

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

func securityGroupStateRefreshFunc(client *ArmClient, resourceGroupName string, securityGroupName string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		res, err := client.secGroupClient.Get(resourceGroupName, securityGroupName, "")
		if err != nil {
			return nil, "", fmt.Errorf("Error issuing read request in securityGroupStateRefreshFunc to Azure ARM for network security group '%s' (RG: '%s'): %s", securityGroupName, resourceGroupName, err)
		}

		return res, *res.Properties.ProvisioningState, nil
	}
}

func flattenNetworkSecurityRules(rules *[]network.SecurityRule) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(*rules))
	for _, rule := range *rules {
		sgRule := make(map[string]interface{})
		sgRule["name"] = *rule.Name
		sgRule["destination_address_prefix"] = *rule.Properties.DestinationAddressPrefix
		sgRule["destination_port_range"] = *rule.Properties.DestinationPortRange
		sgRule["source_address_prefix"] = *rule.Properties.SourceAddressPrefix
		sgRule["source_port_range"] = *rule.Properties.SourcePortRange
		sgRule["priority"] = int(*rule.Properties.Priority)
		sgRule["access"] = rule.Properties.Access
		sgRule["direction"] = rule.Properties.Direction
		sgRule["protocol"] = rule.Properties.Protocol

		if rule.Properties.Description != nil {
			sgRule["description"] = *rule.Properties.Description
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
		priority := data["priority"].(int)

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
			Name:       &name,
			Properties: &properties,
		}

		rules = append(rules, rule)
	}

	return rules, nil
}
