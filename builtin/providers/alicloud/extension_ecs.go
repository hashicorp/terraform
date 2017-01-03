package alicloud

type GroupRuleDirection string

const (
	GroupRuleIngress = GroupRuleDirection("ingress")
	GroupRuleEgress  = GroupRuleDirection("egress")
)

type GroupRuleIpProtocol string

const (
	GroupRuleTcp  = GroupRuleIpProtocol("tcp")
	GroupRuleUdp  = GroupRuleIpProtocol("udp")
	GroupRuleIcmp = GroupRuleIpProtocol("icmp")
	GroupRuleGre  = GroupRuleIpProtocol("gre")
	GroupRuleAll  = GroupRuleIpProtocol("all")
)

type GroupRuleNicType string

const (
	GroupRuleInternet = GroupRuleNicType("internet")
	GroupRuleIntranet = GroupRuleNicType("intranet")
)

type GroupRulePolicy string

const (
	GroupRulePolicyAccept = GroupRulePolicy("accept")
	GroupRulePolicyDrop   = GroupRulePolicy("drop")
)
