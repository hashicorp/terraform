package vcd

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	types "github.com/hmrc/vmware-govcd/types/v56"
)

func resourceVcdFirewallRules() *schema.Resource {
	return &schema.Resource{
		Create: resourceVcdFirewallRulesCreate,
		Delete: resourceFirewallRulesDelete,
		Read:   resourceFirewallRulesRead,

		Schema: map[string]*schema.Schema{
			"edge_gateway": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"default_action": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"rule": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},

						"description": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"policy": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"protocol": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"destination_port": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"destination_ip": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"source_port": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"source_ip": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func resourceVcdFirewallRulesCreate(d *schema.ResourceData, meta interface{}) error {
	vcdClient := meta.(*VCDClient)
	vcdClient.Mutex.Lock()
	defer vcdClient.Mutex.Unlock()

	edgeGateway, err := vcdClient.OrgVdc.FindEdgeGateway(d.Get("edge_gateway").(string))
	if err != nil {
		return fmt.Errorf("Unable to find edge gateway: %s", err)
	}

	err = retryCall(vcdClient.MaxRetryTimeout, func() error {
		edgeGateway.Refresh()
		firewallRules, _ := expandFirewallRules(d, edgeGateway.EdgeGateway)
		task, err := edgeGateway.CreateFirewallRules(d.Get("default_action").(string), firewallRules)
		if err != nil {
			log.Printf("[INFO] Error setting firewall rules: %s", err)
			return fmt.Errorf("Error setting firewall rules: %#v", err)
		}

		return task.WaitTaskCompletion()
	})
	if err != nil {
		return fmt.Errorf("Error completing tasks: %#v", err)
	}

	d.SetId(d.Get("edge_gateway").(string))

	return resourceFirewallRulesRead(d, meta)
}

func resourceFirewallRulesDelete(d *schema.ResourceData, meta interface{}) error {
	vcdClient := meta.(*VCDClient)
	vcdClient.Mutex.Lock()
	defer vcdClient.Mutex.Unlock()

	edgeGateway, err := vcdClient.OrgVdc.FindEdgeGateway(d.Get("edge_gateway").(string))

	firewallRules := deleteFirewallRules(d, edgeGateway.EdgeGateway)
	defaultAction := edgeGateway.EdgeGateway.Configuration.EdgeGatewayServiceConfiguration.FirewallService.DefaultAction
	task, err := edgeGateway.CreateFirewallRules(defaultAction, firewallRules)
	if err != nil {
		return fmt.Errorf("Error deleting firewall rules: %#v", err)
	}

	err = task.WaitTaskCompletion()
	if err != nil {
		return fmt.Errorf("Error completing tasks: %#v", err)
	}

	return nil
}

func resourceFirewallRulesRead(d *schema.ResourceData, meta interface{}) error {
	vcdClient := meta.(*VCDClient)

	edgeGateway, err := vcdClient.OrgVdc.FindEdgeGateway(d.Get("edge_gateway").(string))
	if err != nil {
		return fmt.Errorf("Error finding edge gateway: %#v", err)
	}
	ruleList := d.Get("rule").([]interface{})
	firewallRules := *edgeGateway.EdgeGateway.Configuration.EdgeGatewayServiceConfiguration.FirewallService
	rulesCount := d.Get("rule.#").(int)
	for i := 0; i < rulesCount; i++ {
		prefix := fmt.Sprintf("rule.%d", i)
		if d.Get(prefix+".id").(string) == "" {
			log.Printf("[INFO] Rule %d has no id. Searching...", i)
			ruleid, err := matchFirewallRule(d, prefix, firewallRules.FirewallRule)
			if err == nil {
				currentRule := ruleList[i].(map[string]interface{})
				currentRule["id"] = ruleid
				ruleList[i] = currentRule
			}
		}
	}
	d.Set("rule", ruleList)
	d.Set("default_action", firewallRules.DefaultAction)

	return nil
}

func deleteFirewallRules(d *schema.ResourceData, gateway *types.EdgeGateway) []*types.FirewallRule {
	firewallRules := gateway.Configuration.EdgeGatewayServiceConfiguration.FirewallService.FirewallRule
	rulesCount := d.Get("rule.#").(int)
	fwrules := make([]*types.FirewallRule, 0, len(firewallRules)-rulesCount)

	for _, f := range firewallRules {
		keep := true
		for i := 0; i < rulesCount; i++ {
			if d.Get(fmt.Sprintf("rule.%d.id", i)).(string) != f.ID {
				continue
			}
			keep = false
		}
		if keep {
			fwrules = append(fwrules, f)
		}
	}
	return fwrules
}

func matchFirewallRule(d *schema.ResourceData, prefix string, rules []*types.FirewallRule) (string, error) {

	for _, m := range rules {
		if d.Get(prefix+".description").(string) == m.Description &&
			d.Get(prefix+".policy").(string) == m.Policy &&
			strings.ToLower(d.Get(prefix+".protocol").(string)) == getProtocol(*m.Protocols) &&
			strings.ToLower(d.Get(prefix+".destination_port").(string)) == getPortString(m.Port) &&
			strings.ToLower(d.Get(prefix+".destination_ip").(string)) == strings.ToLower(m.DestinationIP) &&
			strings.ToLower(d.Get(prefix+".source_port").(string)) == getPortString(m.SourcePort) &&
			strings.ToLower(d.Get(prefix+".source_ip").(string)) == strings.ToLower(m.SourceIP) {
			return m.ID, nil
		}
	}
	return "", fmt.Errorf("Unable to find rule")
}
