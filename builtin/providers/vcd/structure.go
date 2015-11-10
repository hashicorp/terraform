package vcd

import (
	"github.com/hashicorp/terraform/helper/resource"
	types "github.com/hmrc/vmware-govcd/types/v56"
	"strconv"
	"time"
)

func expandIpRange(configured []interface{}) types.IPRanges {
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

func expandFirewallRules(configured []interface{}, gateway *types.EdgeGateway) ([]*types.FirewallRule, error) {
	//firewallRules := make([]*types.FirewallRule, 0, len(configured))
	firewallRules := gateway.Configuration.EdgeGatewayServiceConfiguration.FirewallService.FirewallRule

	for i := len(configured) - 1; i >= 0; i-- {
		data := configured[i].(map[string]interface{})

		var protocol *types.FirewallRuleProtocols
		switch data["protocol"].(string) {
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
			Description:          data["description"].(string),
			Policy:               data["policy"].(string),
			Protocols:            protocol,
			Port:                 getNumericPort(data["destination_port"]),
			DestinationPortRange: data["destination_port"].(string),
			DestinationIP:        data["destination_ip"].(string),
			SourcePort:           getNumericPort(data["source_port"]),
			SourcePortRange:      data["source_port"].(string),
			SourceIP:             data["source_ip"].(string),
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

func retryCall(min int, f resource.RetryFunc) error {
	return resource.Retry(time.Duration(min)*time.Minute, f)
}
