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
		Delete: resourceAzureSecurityGroupRuleDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: parameterDescriptions["name"],
			},
			"security_group_names": &schema.Schema{
				Type:        schema.TypeSet,
				Required:    true,
				ForceNew:    true,
				Description: parameterDescriptions["netsecgroup_secgroup_names"],
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Set: schema.HashString,
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

	azureClient.secGroupMutex.Lock()
	defer azureClient.secGroupMutex.Unlock()

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

	// apply the rule to all the necessary network security groups:
	secGroups := d.Get("security_group_names").(*schema.Set).List()
	for _, sg := range secGroups {
		secGroup := sg.(string)

		// send the create request to Azure:
		log.Printf("[INFO] Sending Azure security group rule addition request for security group %q.", secGroup)
		reqID, err := secGroupClient.SetNetworkSecurityGroupRule(
			secGroup,
			rule,
		)
		if err != nil {
			return fmt.Errorf("Error sending Azure network security group rule creation request for security group %q: %s", secGroup, err)
		}
		err = mgmtClient.WaitForOperation(reqID, nil)
		if err != nil {
			return fmt.Errorf("Error creating Azure network security group rule for security group %q: %s", secGroup, err)
		}
	}

	d.SetId(name)
	return nil
}

// resourceAzureSecurityGroupRuleRead does all the necessary API calls to
// read the state of a network security group ruke off Azure.
func resourceAzureSecurityGroupRuleRead(d *schema.ResourceData, meta interface{}) error {
	azureClient := meta.(*Client)
	secGroupClient := azureClient.secGroupClient

	var found bool
	name := d.Get("name").(string)

	secGroups := d.Get("security_group_names").(*schema.Set).List()
	remaining := schema.NewSet(schema.HashString, nil)

	// for each of our security groups; check for our rule:
	for _, sg := range secGroups {
		secGroupName := sg.(string)

		// get info on the network security group and check its rules for this one:
		log.Printf("[INFO] Sending Azure network security group rule query for security group %s.", secGroupName)
		secgroup, err := secGroupClient.GetNetworkSecurityGroup(secGroupName)
		if err != nil {
			if !management.IsResourceNotFoundError(err) {
				return fmt.Errorf("Error issuing network security group rules query for security group %q: %s", secGroupName, err)
			} else {
				// it meants that the network security group this rule belonged to has
				// been deleted; so we skip this iteration:
				continue
			}
		}

		// find our security rule:
		for _, rule := range secgroup.Rules {
			if rule.Name == name {
				// note the fact that this rule still apllies to this security group:
				found = true
				remaining.Add(secGroupName)

				break
			}
		}
	}

	// check to see if there is any security group still having this rule:
	if !found {
		d.SetId("")
		return nil
	}

	// now; we must update the set of security groups still having this rule:
	d.Set("security_group_names", remaining)
	return nil
}

// resourceAzureSecurityGroupRuleUpdate does all the necessary API calls to
// update the state of a network security group rule off Azure.
func resourceAzureSecurityGroupRuleUpdate(d *schema.ResourceData, meta interface{}) error {
	azureClient := meta.(*Client)
	mgmtClient := azureClient.mgmtClient
	secGroupClient := azureClient.secGroupClient

	azureClient.secGroupMutex.Lock()
	defer azureClient.secGroupMutex.Unlock()

	var found bool
	name := d.Get("name").(string)
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

	// iterate over all the security groups that should have this rule and
	// update it per security group:
	remaining := schema.NewSet(schema.HashString, nil)
	secGroupNames := d.Get("security_group_names").(*schema.Set).List()
	for _, sg := range secGroupNames {
		secGroupName := sg.(string)

		// get info on the network security group and check its rules for this one:
		log.Printf("[INFO] Sending Azure network security group rule query for security group %q.", secGroupName)
		secgroup, err := secGroupClient.GetNetworkSecurityGroup(secGroupName)
		if err != nil {
			if !management.IsResourceNotFoundError(err) {
				return fmt.Errorf("Error issuing network security group rules query: %s", err)
			} else {
				// it meants that the network security group this rule belonged to has
				// been deleted; so we skip this iteration:
				continue
			}
		}

		// try and find our security group rule:
		for _, rule := range secgroup.Rules {
			if rule.Name == name {
				// note the fact that this rule still applies to this security group:
				found = true
				remaining.Add(secGroupName)

				// and go ahead and update it:
				log.Printf("[INFO] Sending Azure network security group rule update request for security group %q.", secGroupName)
				reqID, err := secGroupClient.SetNetworkSecurityGroupRule(
					secGroupName,
					newRule,
				)
				if err != nil {
					return fmt.Errorf("Error sending Azure network security group rule update request for security group %q: %s", secGroupName, err)
				}
				err = mgmtClient.WaitForOperation(reqID, nil)
				if err != nil {
					return fmt.Errorf("Error updating Azure network security group rule for security group %q: %s", secGroupName, err)
				}

				break
			}
		}
	}

	// check to see if there is any security group still having this rule:
	if !found {
		d.SetId("")
		return nil
	}

	// here; we must update the set of security groups still having this rule:
	d.Set("security_group_names", remaining)

	return nil
}

// resourceAzureSecurityGroupRuleDelete does all the necessary API calls to
// delete a network security group rule off Azure.
func resourceAzureSecurityGroupRuleDelete(d *schema.ResourceData, meta interface{}) error {
	azureClient := meta.(*Client)
	mgmtClient := azureClient.mgmtClient
	secGroupClient := azureClient.secGroupClient

	azureClient.secGroupMutex.Lock()
	defer azureClient.secGroupMutex.Unlock()

	name := d.Get("name").(string)
	secGroupNames := d.Get("security_group_names").(*schema.Set).List()
	for _, sg := range secGroupNames {
		secGroupName := sg.(string)

		// get info on the network security group and search for our rule:
		log.Printf("[INFO] Sending network security group rule query for security group %q.", secGroupName)
		secgroup, err := secGroupClient.GetNetworkSecurityGroup(secGroupName)
		if err != nil {
			if management.IsResourceNotFoundError(err) {
				// it means that this network security group this rule belonged to has
				// been deleted; so we need not do anything more here:
				continue
			} else {
				return fmt.Errorf("Error issuing Azure network security group rules query for security group %q: %s", secGroupName, err)
			}
		}

		// check if the rule has been deleted in the meantime:
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
			break
		}
	}

	return nil
}
