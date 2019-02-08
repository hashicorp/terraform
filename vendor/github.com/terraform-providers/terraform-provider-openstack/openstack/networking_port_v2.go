package openstack

import (
	"bytes"
	"fmt"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/extradhcpopts"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceNetworkingPortV2StateRefreshFunc(client *gophercloud.ServiceClient, portID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		n, err := ports.Get(client, portID).Extract()
		if err != nil {
			if _, ok := err.(gophercloud.ErrDefault404); ok {
				return n, "DELETED", nil
			}

			return n, "", err
		}

		return n, n.Status, nil
	}
}

func expandNetworkingPortDHCPOptsV2Create(dhcpOpts *schema.Set) []extradhcpopts.CreateExtraDHCPOpt {
	rawDHCPOpts := dhcpOpts.List()

	extraDHCPOpts := make([]extradhcpopts.CreateExtraDHCPOpt, len(rawDHCPOpts))
	for i, raw := range rawDHCPOpts {
		rawMap := raw.(map[string]interface{})

		ipVersion := rawMap["ip_version"].(int)
		optName := rawMap["name"].(string)
		optValue := rawMap["value"].(string)

		extraDHCPOpts[i] = extradhcpopts.CreateExtraDHCPOpt{
			OptName:   optName,
			OptValue:  optValue,
			IPVersion: gophercloud.IPVersion(ipVersion),
		}
	}

	return extraDHCPOpts
}

func expandNetworkingPortDHCPOptsV2Update(dhcpOpts *schema.Set) []extradhcpopts.UpdateExtraDHCPOpt {
	rawDHCPOpts := dhcpOpts.List()

	extraDHCPOpts := make([]extradhcpopts.UpdateExtraDHCPOpt, len(rawDHCPOpts))
	for i, raw := range rawDHCPOpts {
		rawMap := raw.(map[string]interface{})

		ipVersion := rawMap["ip_version"].(int)
		optName := rawMap["name"].(string)
		optValue := rawMap["value"].(string)

		extraDHCPOpts[i] = extradhcpopts.UpdateExtraDHCPOpt{
			OptName:   optName,
			OptValue:  &optValue,
			IPVersion: gophercloud.IPVersion(ipVersion),
		}
	}

	return extraDHCPOpts
}

func expandNetworkingPortDHCPOptsV2Delete(dhcpOpts *schema.Set) []extradhcpopts.UpdateExtraDHCPOpt {
	if dhcpOpts == nil {
		return []extradhcpopts.UpdateExtraDHCPOpt{}
	}

	rawDHCPOpts := dhcpOpts.List()

	extraDHCPOpts := make([]extradhcpopts.UpdateExtraDHCPOpt, len(rawDHCPOpts))
	for i, raw := range rawDHCPOpts {
		rawMap := raw.(map[string]interface{})
		extraDHCPOpts[i] = extradhcpopts.UpdateExtraDHCPOpt{
			OptName:  rawMap["name"].(string),
			OptValue: nil,
		}
	}

	return extraDHCPOpts
}

func flattenNetworkingPortDHCPOptsV2(dhcpOpts extradhcpopts.ExtraDHCPOptsExt) []map[string]interface{} {
	dhcpOptsSet := make([]map[string]interface{}, len(dhcpOpts.ExtraDHCPOpts))

	for i, dhcpOpt := range dhcpOpts.ExtraDHCPOpts {
		dhcpOptsSet[i] = map[string]interface{}{
			"ip_version": dhcpOpt.IPVersion,
			"name":       dhcpOpt.OptName,
			"value":      dhcpOpt.OptValue,
		}
	}

	return dhcpOptsSet
}

func expandNetworkingPortAllowedAddressPairsV2(allowedAddressPairs *schema.Set) []ports.AddressPair {
	rawPairs := allowedAddressPairs.List()

	pairs := make([]ports.AddressPair, len(rawPairs))
	for i, raw := range rawPairs {
		rawMap := raw.(map[string]interface{})
		pairs[i] = ports.AddressPair{
			IPAddress:  rawMap["ip_address"].(string),
			MACAddress: rawMap["mac_address"].(string),
		}
	}

	return pairs
}

func flattenNetworkingPortAllowedAddressPairsV2(mac string, allowedAddressPairs []ports.AddressPair) []map[string]interface{} {
	pairs := make([]map[string]interface{}, len(allowedAddressPairs))

	for i, pair := range allowedAddressPairs {
		pairs[i] = map[string]interface{}{
			"ip_address": pair.IPAddress,
		}
		// Only set the MAC address if it is different than the
		// port's MAC. This means that a specific MAC was set.
		if pair.MACAddress != mac {
			pairs[i]["mac_address"] = pair.MACAddress
		}
	}

	return pairs
}

func expandNetworkingPortFixedIPV2(d *schema.ResourceData) interface{} {
	// If no_fixed_ip was specified, then just return an empty array.
	// Since no_fixed_ip is mutually exclusive to fixed_ip,
	// we can safely do this.
	//
	// Since we're only concerned about no_fixed_ip being set to "true",
	// GetOk is used.
	if _, ok := d.GetOk("no_fixed_ip"); ok {
		return []interface{}{}
	}

	rawIP := d.Get("fixed_ip").([]interface{})

	if len(rawIP) == 0 {
		return nil
	}

	ip := make([]ports.IP, len(rawIP))
	for i, raw := range rawIP {
		rawMap := raw.(map[string]interface{})
		ip[i] = ports.IP{
			SubnetID:  rawMap["subnet_id"].(string),
			IPAddress: rawMap["ip_address"].(string),
		}
	}
	return ip
}

func resourceNetworkingPortV2AllowedAddressPairsHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-%s", m["ip_address"].(string), m["mac_address"].(string)))

	return hashcode.String(buf.String())
}

func expandNetworkingPortFixedIPToStringSlice(fixedIPs []ports.IP) []string {
	s := make([]string, len(fixedIPs))
	for i, fixedIP := range fixedIPs {
		s[i] = fixedIP.IPAddress
	}

	return s
}
