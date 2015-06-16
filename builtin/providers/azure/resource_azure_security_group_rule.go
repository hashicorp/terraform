package azure

import (
	"fmt"
	"log"

	"github.com/Azure/azure-sdk-for-go/management"
	netsecgroup "github.com/Azure/azure-sdk-for-go/management/networksecuritygroup"
	"github.com/hashicorp/terraform/helper/schema"
)

// resourceAzureSecurityGroupRule returns the *schema.Resource for
// a network security group rule on Azure.
func resourceAzureSecurityGroupRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceAzureSecurityGroupRuleCreate,
		Read:   resourceAzureSecurityGroupRuleRead,
		Update: resourceAzureSecurityGroupRuleUpdate,
		Exists: resourceAzureSecurityGroupRuleExists,
		Delete: resourceAzureSecurityGroupRuleDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: parameterDescriptions["name"],
			},
			"security_group_name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: parameterDescriptions["netsecgroup_secgroup_name"],
			},
			"type": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: parameterDescriptions["netsecgroup_type"],
			},
			"priority": &schema.Schema{
				Type:        schema.TypeInt,
				Required:    true,
				Description: parameterDescriptions["netsecgroup_priority"],
			},
			"action": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: parameterDescriptions["netsecgroup_action"],
			},
			"source_address_prefix": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: parameterDescriptions["netsecgroup_src_addr_prefix"],
			},
			"source_port_range": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: parameterDescriptions["netsecgroup_src_port_range"],
			},
			"destination_address_prefix": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: parameterDescriptions["netsecgroup_dest_addr_prefix"],
			},
			"destination_port_range": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: parameterDescriptions["netsecgroup_dest_port_range"],
			},
			"protocol": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: parameterDescriptions["netsecgroup_protocol"],
			},
		},
	}
}

// resourceAzureSecurityGroupRuleCreate does all the necessary API calls to
// create a new network security group rule on Azure.
func resourceAzureSecurityGroupRuleCreate(d *schema.ResourceData, meta interface{}) error {
	azureClient := meta.(*Client)
	mgmtClient := azureClient.mgmtClient
	secGroupClient := azureClient.secGroupClient

	// create and configure the RuleResponse:
	name := d.Get("name").(string)
	rule := netsecgroup.RuleRequest{
		Name:                     name,
		Type:                     netsecgroup.RuleType(d.Get("type").(string)),
		Priority:                 d.Get("priority").(int),
		Action:                   netsecgroup.RuleAction(d.Get("action").(string)),
		SourceAddressPrefix:      d.Get("source_address_prefix").(string),
		SourcePortRange:          d.Get("source_port_range").(string),
		DestinationAddressPrefix: d.Get("destination_address_prefix").(string),
		DestinationPortRange:     d.Get("destination_port_range").(string),
		Protocol:                 netsecgroup.RuleProtocol(d.Get("protocol").(string)),
	}

	// send the create request to Azure:
	log.Println("[INFO] Sending network security group rule creation request to Azure.")
	reqID, err := secGroupClient.SetNetworkSecurityGroupRule(
		d.Get("security_group_name").(string),
		rule,
	)
	if err != nil {
		return fmt.Errorf("Error sending network security group rule creation request to Azure: %s", err)
	}
	err = mgmtClient.WaitForOperation(reqID, nil)
	if err != nil {
		return fmt.Errorf("Error creating network security group rule on Azure: %s", err)
	}

	d.SetId(name)
	return nil
}

// resourceAzureSecurityGroupRuleRead does all the necessary API calls to
// read the state of a network security group ruke off Azure.
func resourceAzureSecurityGroupRuleRead(d *schema.ResourceData, meta interface{}) error {
	azureClient := meta.(*Client)
	secGroupClient := azureClient.secGroupClient

	secGroupName := d.Get("security_group_name").(string)

	// get info on the network security group and check its rules for this one:
	log.Println("[INFO] Sending network security group rule query to Azure.")
	secgroup, err := secGroupClient.GetNetworkSecurityGroup(secGroupName)
	if err != nil {
		if !management.IsResourceNotFoundError(err) {
			return fmt.Errorf("Error issuing network security group rules query: %s", err)
		} else {
			// it meants that the network security group this rule belonged to has
			// been deleted; so we must remove this resource from the schema:
			d.SetId("")
			return nil
		}
	}

	// find our security rule:
	var found bool
	name := d.Get("name").(string)
	for _, rule := range secgroup.Rules {
		if rule.Name == name {
			found = true
			log.Println("[DEBUG] Reading state of Azure network security group rule.")

			d.Set("type", rule.Type)
			d.Set("priority", rule.Priority)
			d.Set("action", rule.Action)
			d.Set("source_address_prefix", rule.SourceAddressPrefix)
			d.Set("source_port_range", rule.SourcePortRange)
			d.Set("destination_address_prefix", rule.DestinationAddressPrefix)
			d.Set("destination_port_range", rule.DestinationPortRange)
			d.Set("protocol", rule.Protocol)

			break
		}
	}

	// check if the rule still exists, and is not, remove the resource:
	if !found {
		d.SetId("")
	}
	return nil
}

// resourceAzureSecurityGroupRuleUpdate does all the necessary API calls to
// update the state of a network security group ruke off Azure.
func resourceAzureSecurityGroupRuleUpdate(d *schema.ResourceData, meta interface{}) error {
	azureClient := meta.(*Client)
	mgmtClient := azureClient.mgmtClient
	secGroupClient := azureClient.secGroupClient

	secGroupName := d.Get("security_group_name").(string)

	// get info on the network security group and check its rules for this one:
	log.Println("[INFO] Sending network security group rule query for update to Azure.")
	secgroup, err := secGroupClient.GetNetworkSecurityGroup(secGroupName)
	if err != nil {
		if !management.IsResourceNotFoundError(err) {
			return fmt.Errorf("Error issuing network security group rules query: %s", err)
		} else {
			// it meants that the network security group this rule belonged to has
			// been deleted; so we must remove this resource from the schema:
			d.SetId("")
			return nil
		}
	}

	// try and find our security group rule:
	var found bool
	name := d.Get("name").(string)
	for _, rule := range secgroup.Rules {
		if rule.Name == name {
			found = true
		}
	}
	// check is the resource has not been deleted in the meantime:
	if !found {
		// if not; remove the resource:
		d.SetId("")
		return nil
	}

	// else, start building up the rule request struct:
	newRule := netsecgroup.RuleRequest{
		Name:                     d.Get("name").(string),
		Type:                     netsecgroup.RuleType(d.Get("type").(string)),
		Priority:                 d.Get("priority").(int),
		Action:                   netsecgroup.RuleAction(d.Get("action").(string)),
		SourceAddressPrefix:      d.Get("source_address_prefix").(string),
		SourcePortRange:          d.Get("source_port_range").(string),
		DestinationAddressPrefix: d.Get("destination_address_prefix").(string),
		DestinationPortRange:     d.Get("destination_port_range").(string),
		Protocol:                 netsecgroup.RuleProtocol(d.Get("protocol").(string)),
	}

	// send the create request to Azure:
	log.Println("[INFO] Sending network security group rule update request to Azure.")
	reqID, err := secGroupClient.SetNetworkSecurityGroupRule(
		secGroupName,
		newRule,
	)
	if err != nil {
		return fmt.Errorf("Error sending network security group rule update request to Azure: %s", err)
	}
	err = mgmtClient.WaitForOperation(reqID, nil)
	if err != nil {
		return fmt.Errorf("Error updating network security group rule on Azure: %s", err)
	}

	return nil
}

// resourceAzureSecurityGroupRuleExists does all the necessary API calls to
// check for the existence of the network security group rule on Azure.
func resourceAzureSecurityGroupRuleExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	secGroupClient := meta.(*Client).secGroupClient

	secGroupName := d.Get("security_group_name").(string)

	// get info on the network security group and search for our rule:
	log.Println("[INFO] Sending network security group rule query for existence check to Azure.")
	secgroup, err := secGroupClient.GetNetworkSecurityGroup(secGroupName)
	if err != nil {
		if !management.IsResourceNotFoundError(err) {
			return false, fmt.Errorf("Error issuing network security group rules query: %s", err)
		} else {
			// it meants that the network security group this rule belonged to has
			// been deleted; so we must remove this resource from the schema:
			d.SetId("")
			return false, nil
		}
	}

	// try and find our security group rule:
	name := d.Get("name").(string)
	for _, rule := range secgroup.Rules {
		if rule.Name == name {
			return true, nil
		}
	}

	// if here; it means the resource has been deleted in the
	// meantime and must be removed from the schema:
	d.SetId("")

	return false, nil
}

// resourceAzureSecurityGroupRuleDelete does all the necessary API calls to
// delete a network security group rule off Azure.
func resourceAzureSecurityGroupRuleDelete(d *schema.ResourceData, meta interface{}) error {
	azureClient := meta.(*Client)
	mgmtClient := azureClient.mgmtClient
	secGroupClient := azureClient.secGroupClient

	secGroupName := d.Get("security_group_name").(string)

	// get info on the network security group and search for our rule:
	log.Println("[INFO] Sending network security group rule query for deletion to Azure.")
	secgroup, err := secGroupClient.GetNetworkSecurityGroup(secGroupName)
	if err != nil {
		if management.IsResourceNotFoundError(err) {
			// it meants that the network security group this rule belonged to has
			// been deleted; so we need do nothing more but stop tracking the resource:
			d.SetId("")
			return nil
		} else {
			return fmt.Errorf("Error issuing network security group rules query: %s", err)
		}
	}

	// check is the resource has not been deleted in the meantime:
	name := d.Get("name").(string)
	for _, rule := range secgroup.Rules {
		if rule.Name == name {
			// if not; we shall issue the delete:
			reqID, err := secGroupClient.DeleteNetworkSecurityGroupRule(secGroupName, name)
			if err != nil {
				return fmt.Errorf("Error sending network security group rule delete request to Azure: %s", err)
			}
			err = mgmtClient.WaitForOperation(reqID, nil)
			if err != nil {
				return fmt.Errorf("Error deleting network security group rule off Azure: %s", err)
			}
		}
	}

	return nil
}
