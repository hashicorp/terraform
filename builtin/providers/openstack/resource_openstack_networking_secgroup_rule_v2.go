package openstack

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/networking/v2/extensions/security/rules"
)

func resourceNetworkingSecGroupRuleV2() *schema.Resource {
	return &schema.Resource{
		Create: resourceNetworkingSecGroupRuleV2Create,
		Read:   resourceNetworkingSecGroupRuleV2Read,
		Delete: resourceNetworkingSecGroupRuleV2Delete,

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: schema.EnvDefaultFunc("OS_REGION_NAME", ""),
			},
			"direction": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"ethertype": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"port_range_min": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
			"port_range_max": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
			"protocol": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
			"remote_group_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
			"remote_ip_prefix": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
			"security_group_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"tenant_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
		},
	}
}

func resourceNetworkingSecGroupRuleV2Create(d *schema.ResourceData, meta interface{}) error {

	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	portRangeMin := d.Get("port_range_min").(int)
	portRangeMax := d.Get("port_range_max").(int)
	protocol := d.Get("protocol").(string)

	if protocol == "" {
		if portRangeMin != 0 || portRangeMax != 0 {
			return fmt.Errorf("A protocol must be specified when using port_range_min and port_range_max")
		}
	}

	opts := rules.CreateOpts{
		Direction:      d.Get("direction").(string),
		EtherType:      d.Get("ethertype").(string),
		SecGroupID:     d.Get("security_group_id").(string),
		PortRangeMin:   d.Get("port_range_min").(int),
		PortRangeMax:   d.Get("port_range_max").(int),
		Protocol:       d.Get("protocol").(string),
		RemoteGroupID:  d.Get("remote_group_id").(string),
		RemoteIPPrefix: d.Get("remote_ip_prefix").(string),
		TenantID:       d.Get("tenant_id").(string),
	}

	log.Printf("[DEBUG] Create OpenStack Neutron security group: %#v", opts)

	security_group_rule, err := rules.Create(networkingClient, opts).Extract()
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] OpenStack Neutron Security Group Rule created: %#v", security_group_rule)

	d.SetId(security_group_rule.ID)

	return resourceNetworkingSecGroupRuleV2Read(d, meta)
}

func resourceNetworkingSecGroupRuleV2Read(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Retrieve information about security group rule: %s", d.Id())

	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	security_group_rule, err := rules.Get(networkingClient, d.Id()).Extract()

	if err != nil {
		return CheckDeleted(d, err, "OpenStack Security Group Rule")
	}

	d.Set("protocol", security_group_rule.Protocol)
	d.Set("port_range_min", security_group_rule.PortRangeMin)
	d.Set("port_range_max", security_group_rule.PortRangeMax)
	d.Set("remote_group_id", security_group_rule.RemoteGroupID)
	d.Set("remote_ip_prefix", security_group_rule.RemoteIPPrefix)
	d.Set("tenant_id", security_group_rule.TenantID)
	return nil
}

func resourceNetworkingSecGroupRuleV2Delete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Destroy security group rule: %s", d.Id())

	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ACTIVE"},
		Target:     []string{"DELETED"},
		Refresh:    waitForSecGroupRuleDelete(networkingClient, d.Id()),
		Timeout:    2 * time.Minute,
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error deleting OpenStack Neutron Security Group Rule: %s", err)
	}

	d.SetId("")
	return err
}

func waitForSecGroupRuleDelete(networkingClient *gophercloud.ServiceClient, secGroupRuleId string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		log.Printf("[DEBUG] Attempting to delete OpenStack Security Group Rule %s.\n", secGroupRuleId)

		r, err := rules.Get(networkingClient, secGroupRuleId).Extract()
		if err != nil {
			errCode, ok := err.(*gophercloud.UnexpectedResponseCodeError)
			if !ok {
				return r, "ACTIVE", err
			}
			if errCode.Actual == 404 {
				log.Printf("[DEBUG] Successfully deleted OpenStack Neutron Security Group Rule %s", secGroupRuleId)
				return r, "DELETED", nil
			}
		}

		err = rules.Delete(networkingClient, secGroupRuleId).ExtractErr()
		if err != nil {
			errCode, ok := err.(*gophercloud.UnexpectedResponseCodeError)
			if !ok {
				return r, "ACTIVE", err
			}
			if errCode.Actual == 404 {
				log.Printf("[DEBUG] Successfully deleted OpenStack Neutron Security Group Rule %s", secGroupRuleId)
				return r, "DELETED", nil
			}
		}

		log.Printf("[DEBUG] OpenStack Neutron Security Group Rule %s still active.\n", secGroupRuleId)
		return r, "ACTIVE", nil
	}
}
