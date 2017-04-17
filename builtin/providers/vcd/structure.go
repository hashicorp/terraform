package vcd

import (
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	types "github.com/ukcloud/govcloudair/types/v56"
)

func expandIPRange(configured []interface{}) types.IPRanges {
	ipRange := make([]*types.IPRange, 0, len(configured))

	for _, ipRaw := range configured {
		data := ipRaw.(map[string]interface{})

		ip := types.IPRange{
			StartAddress: data["start_address"].(string),
			EndAddress:   data["end_address"].(string),
		}

		ipRange = append(ipRange, &ip)
	}

	ipRanges := types.IPRanges{
		IPRange: ipRange,
	}

	return ipRanges
}

func expandFirewallRules(d *schema.ResourceData, gateway *types.EdgeGateway) ([]*types.FirewallRule, error) {
	//firewallRules := make([]*types.FirewallRule, 0, len(configured))
	firewallRules := gateway.Configuration.EdgeGatewayServiceConfiguration.FirewallService.FirewallRule

	rulesCount := d.Get("rule.#").(int)
	for i := 0; i < rulesCount; i++ {
		prefix := fmt.Sprintf("rule.%d", i)

		var protocol *types.FirewallRuleProtocols
		switch d.Get(prefix + ".protocol").(string) {
		case "tcp":
			protocol = &types.FirewallRuleProtocols{
				TCP: true,
			}
		case "udp":
			protocol = &types.FirewallRuleProtocols{
				UDP: true,
			}
		case "icmp":
			protocol = &types.FirewallRuleProtocols{
				ICMP: true,
			}
		default:
			protocol = &types.FirewallRuleProtocols{
				Any: true,
			}
		}
		rule := &types.FirewallRule{
			//ID: strconv.Itoa(len(configured) - i),
			IsEnabled:            true,
			MatchOnTranslate:     false,
			Description:          d.Get(prefix + ".description").(string),
			Policy:               d.Get(prefix + ".policy").(string),
			Protocols:            protocol,
			Port:                 getNumericPort(d.Get(prefix + ".destination_port")),
			DestinationPortRange: d.Get(prefix + ".destination_port").(string),
			DestinationIP:        d.Get(prefix + ".destination_ip").(string),
			SourcePort:           getNumericPort(d.Get(prefix + ".source_port")),
			SourcePortRange:      d.Get(prefix + ".source_port").(string),
			SourceIP:             d.Get(prefix + ".source_ip").(string),
			EnableLogging:        false,
		}
		firewallRules = append(firewallRules, rule)
	}

	return firewallRules, nil
}

func getProtocol(protocol types.FirewallRuleProtocols) string {
	if protocol.TCP {
		return "tcp"
	}
	if protocol.UDP {
		return "udp"
	}
	if protocol.ICMP {
		return "icmp"
	}
	return "any"
}

func getNumericPort(portrange interface{}) int {
	i, err := strconv.Atoi(portrange.(string))
	if err != nil {
		return -1
	}
	return i
}

func getPortString(port int) string {
	if port == -1 {
		return "any"
	}
	portstring := strconv.Itoa(port)
	return portstring
}

func retryCall(seconds int, f resource.RetryFunc) error {
	return resource.Retry(time.Duration(seconds)*time.Second, f)
}
