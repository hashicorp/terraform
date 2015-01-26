package openstack

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack"
	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/secgroups"
)

func resourceComputeSecGroupRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeSecGroupRuleCreate,
		Read:   resourceComputeSecGroupRuleRead,
		Delete: resourceComputeSecGroupRuleDelete,

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: envDefaultFunc("OS_REGION_NAME"),
			},
			"group_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"from_port": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
			"to_port": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
			"ip_protocol": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"cidr": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"from_group_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceComputeSecGroupRuleCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := openstack.NewComputeV2(config.osClient, gophercloud.EndpointOpts{
		Region: d.Get("region").(string),
	})
	if err != nil {
		return fmt.Errorf("Error creating OpenStack compute client: %s", err)
	}

	createOpts := secgroups.CreateRuleOpts{
		ParentGroupID: d.Get("group_id").(string),
		FromPort:      d.Get("from_port").(int),
		ToPort:        d.Get("to_port").(int),
		IPProtocol:    d.Get("ip_protocol").(string),
		CIDR:          d.Get("cidr").(string),
		FromGroupID:   d.Get("from_group_id").(string),
	}

	sgr, err := secgroups.CreateRule(computeClient, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating OpenStack security group rule: %s", err)
	}

	d.SetId(sgr.ID)
	d.Set("group_id", sgr.ParentGroupID)
	d.Set("from_port", sgr.FromPort)
	d.Set("to_port", sgr.ToPort)
	d.Set("ip_protocol", sgr.IPProtocol)
	d.Set("cidr", sgr.IPRange.CIDR)
	d.Set("from_group_id", d.Get("from_group_id").(string))

	return resourceComputeSecGroupRuleRead(d, meta)
}

func resourceComputeSecGroupRuleRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceComputeSecGroupRuleDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := openstack.NewComputeV2(config.osClient, gophercloud.EndpointOpts{
		Region: d.Get("region").(string),
	})
	if err != nil {
		return fmt.Errorf("Error creating OpenStack compute client: %s", err)
	}

	err = secgroups.DeleteRule(computeClient, d.Id()).ExtractErr()
	if err != nil {
		return fmt.Errorf("Error deleting OpenStack security group rule: %s", err)
	}
	d.SetId("")
	return nil
}
