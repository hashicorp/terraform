package aws

import (
	"fmt"
	"net"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func expandNetworkAclEntries(configured []interface{}, entryType string) ([]*ec2.NetworkAclEntry, error) {
	entries := make([]*ec2.NetworkAclEntry, 0, len(configured))
	for _, eRaw := range configured {
		data := eRaw.(map[string]interface{})
		protocol := data["protocol"].(string)
		p, err := strconv.Atoi(protocol)
		if err != nil {
			var ok bool
			p, ok = protocolIntegers()[protocol]
			if !ok {
				return nil, fmt.Errorf("Invalid Protocol %s for rule %#v", protocol, data)
			}
		}

		e := &ec2.NetworkAclEntry{
			Protocol: aws.String(strconv.Itoa(p)),
			PortRange: &ec2.PortRange{
				From: aws.Int64(int64(data["from_port"].(int))),
				To:   aws.Int64(int64(data["to_port"].(int))),
			},
			Egress:     aws.Bool(entryType == "egress"),
			RuleAction: aws.String(data["action"].(string)),
			RuleNumber: aws.Int64(int64(data["rule_no"].(int))),
		}

		if v, ok := data["ipv6_cidr_block"]; ok {
			e.Ipv6CidrBlock = aws.String(v.(string))
		}

		if v, ok := data["cidr_block"]; ok {
			e.CidrBlock = aws.String(v.(string))
		}

		// Specify additional required fields for ICMP
		if p == 1 {
			e.IcmpTypeCode = &ec2.IcmpTypeCode{}
			if v, ok := data["icmp_code"]; ok {
				e.IcmpTypeCode.Code = aws.Int64(int64(v.(int)))
			}
			if v, ok := data["icmp_type"]; ok {
				e.IcmpTypeCode.Type = aws.Int64(int64(v.(int)))
			}
		}

		entries = append(entries, e)
	}
	return entries, nil
}

func flattenNetworkAclEntries(list []*ec2.NetworkAclEntry) []map[string]interface{} {
	entries := make([]map[string]interface{}, 0, len(list))

	for _, entry := range list {

		newEntry := map[string]interface{}{
			"from_port": *entry.PortRange.From,
			"to_port":   *entry.PortRange.To,
			"action":    *entry.RuleAction,
			"rule_no":   *entry.RuleNumber,
			"protocol":  *entry.Protocol,
		}

		if entry.CidrBlock != nil {
			newEntry["cidr_block"] = *entry.CidrBlock
		}

		if entry.Ipv6CidrBlock != nil {
			newEntry["ipv6_cidr_block"] = *entry.Ipv6CidrBlock
		}

		entries = append(entries, newEntry)
	}

	return entries

}

func protocolStrings(protocolIntegers map[string]int) map[int]string {
	protocolStrings := make(map[int]string, len(protocolIntegers))
	for k, v := range protocolIntegers {
		protocolStrings[v] = k
	}

	return protocolStrings
}

func protocolIntegers() map[string]int {
	var protocolIntegers = make(map[string]int)
	protocolIntegers = map[string]int{
		// defined at https://www.iana.org/assignments/protocol-numbers/protocol-numbers.xhtml
		"all":             -1,
		"hopopt":          0,
		"icmp":            1,
		"igmp":            2,
		"ggp":             3,
		"ipv4":            4,
		"st":              5,
		"tcp":             6,
		"cbt":             7,
		"egp":             8,
		"igp":             9,
		"bbn-rcc-mon":     10,
		"nvp-ii":          11,
		"pup":             12,
		"argus":           13,
		"emcon":           14,
		"xnet":            15,
		"chaos":           16,
		"udp":             17,
		"mux":             18,
		"dcn-meas":        19,
		"hmp":             20,
		"prm":             21,
		"xns-idp":         22,
		"trunk-1":         23,
		"trunk-2":         24,
		"leaf-1":          25,
		"leaf-2":          26,
		"rdp":             27,
		"irtp":            28,
		"iso-tp4":         29,
		"netblt":          30,
		"mfe-nsp":         31,
		"merit-inp":       32,
		"dccp":            33,
		"3pc":             34,
		"idpr":            35,
		"xtp":             36,
		"ddp":             37,
		"idpr-cmtp":       38,
		"tp++":            39,
		"il":              40,
		"ipv6":            41,
		"sdrp":            42,
		"ipv6-route":      43,
		"ipv6-frag":       44,
		"idrp":            45,
		"rsvp":            46,
		"gre":             47,
		"dsr":             48,
		"bna":             49,
		"esp":             50,
		"ah":              51,
		"i-nlsp":          52,
		"swipe":           53,
		"narp":            54,
		"mobile":          55,
		"tlsp":            56,
		"ipv6-icmp":       58,
		"ipv6-nonxt":      59,
		"ipv6-opts":       60,
		"61":              61,
		"cftp":            62,
		"63":              63,
		"sat-expak":       64,
		"kryptolan":       65,
		"rvd":             66,
		"ippc":            67,
		"68":              68,
		"sat-mon":         69,
		"visa":            70,
		"ipcv":            71,
		"cpnx":            72,
		"cphb":            73,
		"wsn":             74,
		"pvp":             75,
		"br-sat-mon":      76,
		"sun-nd":          77,
		"wb-mon":          78,
		"wb-expak":        79,
		"iso-ip":          80,
		"vmtp":            81,
		"secure-vmtp":     82,
		"vines":           83,
		"ttp":             84,
		"nsfnet-igp":      85,
		"dgp":             86,
		"tcf":             87,
		"eigrp":           88,
		"ospfigp":         89,
		"sprite-rpc":      90,
		"larp":            91,
		"mtp":             92,
		"ax.25":           93,
		"ipip":            94,
		"micp":            95,
		"scc-sp":          96,
		"etherip":         97,
		"encap":           98,
		"99":              99,
		"gmtp":            100,
		"ifmp":            101,
		"pnni":            102,
		"pim":             103,
		"aris":            104,
		"scps":            105,
		"qnx":             106,
		"a/n":             107,
		"ipcomp":          108,
		"snp":             109,
		"compaq-peer":     110,
		"ipx-in-ip":       111,
		"vrrp":            112,
		"pgm":             113,
		"114":             114,
		"l2tp":            115,
		"dd":              116,
		"iatp":            117,
		"stp":             118,
		"srp":             119,
		"uti":             120,
		"smp":             121,
		"sm":              122,
		"ptp":             123,
		"isis-over-ipv4":  124,
		"fire":            125,
		"crtp":            126,
		"crudp":           127,
		"sscopmce":        128,
		"iplt":            129,
		"sps":             130,
		"pipe":            131,
		"sctp":            132,
		"fc":              133,
		"rsvp-e2e-ignore": 134,
		"mobility-header": 135,
		"udplite":         136,
		"mpls-in-ip":      137,
		"manet":           138,
		"hip":             139,
		"shim6":           140,
		"wesp":            141,
		"rohc":            142,
		"253":             253,
		"254":             254,
	}
	return protocolIntegers
}

// expectedPortPair stores a pair of ports we expect to see together.
type expectedPortPair struct {
	to_port   int64
	from_port int64
}

// validatePorts ensures the ports and protocol match expected
// values.
func validatePorts(to int64, from int64, expected expectedPortPair) bool {
	if to != expected.to_port || from != expected.from_port {
		return false
	}

	return true
}

// validateCIDRBlock ensures the passed CIDR block represents an implied
// network, and not an overly-specified IP address.
func validateCIDRBlock(cidr string) error {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return err
	}
	if ipnet.String() != cidr {
		return fmt.Errorf("%s is not a valid mask; did you mean %s?", cidr, ipnet)
	}

	return nil
}
