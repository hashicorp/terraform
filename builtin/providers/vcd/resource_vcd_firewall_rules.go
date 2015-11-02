package vcd

import (
	"bytes"
	"fmt"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/opencredo/vmware-govcd"
	types "github.com/opencredo/vmware-govcd/types/v56"
	"strings"
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
				Type:     schema.TypeSet,
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
				Set: resourceVcdNetworkFirewallRuleHash,
			},
		},
	}
}

func resourceVcdFirewallRulesCreate(d *schema.ResourceData, meta interface{}) error {
	vcd_client := meta.(*govcd.VCDClient)
	vcd_client.Mutex.Lock()
	defer vcd_client.Mutex.Unlock()

	edgeGateway, err := vcd_client.OrgVdc.FindEdgeGateway(d.Get("edge_gateway").(string))
	if err != nil {
		return fmt.Errorf("Unable to find edge gateway: %s", err)
	}

	err = retryCall(5, func() error {
		edgeGateway.Refresh()
		firewallRules, _ := expandFirewallRules(d.Get("rule").(*schema.Set).List(), edgeGateway.EdgeGateway)
		task, err := edgeGateway.CreateFirewallRules(d.Get("default_action").(string), firewallRules)
		if err != nil {
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
	vcd_client := meta.(*govcd.VCDClient)
	vcd_client.Mutex.Lock()
	defer vcd_client.Mutex.Unlock()

	edgeGateway, err := vcd_client.OrgVdc.FindEdgeGateway(d.Get("edge_gateway").(string))

	firewallRules := deleteFirewallRules(d.Get("rule").(*schema.Set).List(), edgeGateway.EdgeGateway)
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
	vcd_client := meta.(*govcd.VCDClient)

	edgeGateway, err := vcd_client.OrgVdc.FindEdgeGateway(d.Get("edge_gateway").(string))
	if err != nil {
		return fmt.Errorf("Error finding edge gateway: %#v", err)
	}
	firewallRules := *edgeGateway.EdgeGateway.Configuration.EdgeGatewayServiceConfiguration.FirewallService
	d.Set("rule", resourceVcdFirewallRulesGather(firewallRules.FirewallRule, d.Get("rule").(*schema.Set).List()))
	d.Set("default_action", firewallRules.DefaultAction)

	return nil
}

func deleteFirewallRules(configured []interface{}, gateway *types.EdgeGateway) []*types.FirewallRule {
	firewallRules := gateway.Configuration.EdgeGatewayServiceConfiguration.FirewallService.FirewallRule
	fwrules := make([]*types.FirewallRule, 0, len(firewallRules)-len(configured))

	for _, f := range firewallRules {
		keep := true
		for _, r := range configured {
			data := r.(map[string]interface{})
			if data["id"].(string) != f.ID {
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

func resourceVcdFirewallRulesGather(rules []*types.FirewallRule, configured []interface{}) []map[string]interface{} {
	fwrules := make([]map[string]interface{}, 0, len(configured))

	for i := len(configured) - 1; i >= 0; i-- {
		data := configured[i].(map[string]interface{})
		rule, err := matchFirewallRule(data, rules)
		if err != nil {
			continue
		}
		fwrules = append(fwrules, rule)
	}
	return fwrules
}

func matchFirewallRule(data map[string]interface{}, rules []*types.FirewallRule) (map[string]interface{}, error) {
	rule := make(map[string]interface{})
	for _, m := range rules {
		if data["id"].(string) == "" {
			if data["description"].(string) == m.Description &&
				data["policy"].(string) == m.Policy &&
				data["protocol"].(string) == getProtocol(*m.Protocols) &&
				data["destination_port"].(string) == getPortString(m.Port) &&
				strings.ToLower(data["destination_ip"].(string)) == strings.ToLower(m.DestinationIP) &&
				data["source_port"].(string) == getPortString(m.SourcePort) &&
				strings.ToLower(data["source_ip"].(string)) == strings.ToLower(m.SourceIP) {
				rule["id"] = m.ID
				rule["description"] = m.Description
				rule["policy"] = m.Policy
				rule["protocol"] = getProtocol(*m.Protocols)
				rule["destination_port"] = getPortString(m.Port)
				rule["destination_ip"] = strings.ToLower(m.DestinationIP)
				rule["source_port"] = getPortString(m.SourcePort)
				rule["source_ip"] = strings.ToLower(m.SourceIP)
				return rule, nil
			}
		} else {
			if data["id"].(string) == m.ID {
				rule["id"] = m.ID
				rule["description"] = m.Description
				rule["policy"] = m.Policy
				rule["protocol"] = getProtocol(*m.Protocols)
				rule["destination_port"] = getPortString(m.Port)
				rule["destination_ip"] = strings.ToLower(m.DestinationIP)
				rule["source_port"] = getPortString(m.SourcePort)
				rule["source_ip"] = strings.ToLower(m.SourceIP)
				return rule, nil
			}
		}
	}
	return rule, fmt.Errorf("Unable to find rule")
}

func resourceVcdNetworkFirewallRuleHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-",
		strings.ToLower(m["description"].(string))))
	buf.WriteString(fmt.Sprintf("%s-",
		strings.ToLower(m["policy"].(string))))
	buf.WriteString(fmt.Sprintf("%s-",
		strings.ToLower(m["protocol"].(string))))
	buf.WriteString(fmt.Sprintf("%s-",
		strings.ToLower(m["destination_port"].(string))))
	buf.WriteString(fmt.Sprintf("%s-",
		strings.ToLower(m["destination_ip"].(string))))
	buf.WriteString(fmt.Sprintf("%s-",
		strings.ToLower(m["source_port"].(string))))
	buf.WriteString(fmt.Sprintf("%s-",
		strings.ToLower(m["source_ip"].(string))))

	return hashcode.String(buf.String())
}
