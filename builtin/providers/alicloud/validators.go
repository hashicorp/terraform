package alicloud

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/denverdino/aliyungo/common"
	"github.com/denverdino/aliyungo/ecs"
	"github.com/denverdino/aliyungo/slb"
	"github.com/hashicorp/terraform/helper/schema"
	"regexp"
)

// common
func validateInstancePort(v interface{}, k string) (ws []string, errors []error) {
	value := v.(int)
	if value < 1 || value > 65535 {
		errors = append(errors, fmt.Errorf(
			"%q must be a valid port between 1 and 65535",
			k))
		return
	}
	return
}

func validateInstanceProtocol(v interface{}, k string) (ws []string, errors []error) {
	protocol := v.(string)
	if !isProtocolValid(protocol) {
		errors = append(errors, fmt.Errorf(
			"%q is an invalid value. Valid values are either http, https, tcp or udp",
			k))
		return
	}
	return
}

// ecs
func validateDiskCategory(v interface{}, k string) (ws []string, errors []error) {
	category := ecs.DiskCategory(v.(string))
	if category != ecs.DiskCategoryCloud && category != ecs.DiskCategoryCloudEfficiency && category != ecs.DiskCategoryCloudSSD {
		errors = append(errors, fmt.Errorf("%s must be one of %s %s %s", k, ecs.DiskCategoryCloud, ecs.DiskCategoryCloudEfficiency, ecs.DiskCategoryCloudSSD))
	}

	return
}

func validateInstanceName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if len(value) < 2 || len(value) > 128 {
		errors = append(errors, fmt.Errorf("%q cannot be longer than 128 characters", k))
	}

	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		errors = append(errors, fmt.Errorf("%s cannot starts with http:// or https://", k))
	}

	return
}

func validateInstanceDescription(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if len(value) < 2 || len(value) > 256 {
		errors = append(errors, fmt.Errorf("%q cannot be longer than 256 characters", k))

	}
	return
}

func validateDiskName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)

	if value == "" {
		return
	}

	if len(value) < 2 || len(value) > 128 {
		errors = append(errors, fmt.Errorf("%q cannot be longer than 128 characters", k))
	}

	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		errors = append(errors, fmt.Errorf("%s cannot starts with http:// or https://", k))
	}

	return
}

func validateDiskDescription(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if len(value) < 2 || len(value) > 256 {
		errors = append(errors, fmt.Errorf("%q cannot be longer than 256 characters", k))

	}
	return
}

//security group
func validateSecurityGroupName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if len(value) < 2 || len(value) > 128 {
		errors = append(errors, fmt.Errorf("%q cannot be longer than 128 characters", k))
	}

	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		errors = append(errors, fmt.Errorf("%s cannot starts with http:// or https://", k))
	}

	return
}

func validateSecurityGroupDescription(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if len(value) < 2 || len(value) > 256 {
		errors = append(errors, fmt.Errorf("%q cannot be longer than 256 characters", k))

	}
	return
}

func validateSecurityRuleType(v interface{}, k string) (ws []string, errors []error) {
	rt := GroupRuleDirection(v.(string))
	if rt != GroupRuleIngress && rt != GroupRuleEgress {
		errors = append(errors, fmt.Errorf("%s must be one of %s %s", k, GroupRuleIngress, GroupRuleEgress))
	}

	return
}

func validateSecurityRuleIpProtocol(v interface{}, k string) (ws []string, errors []error) {
	pt := GroupRuleIpProtocol(v.(string))
	if pt != GroupRuleTcp && pt != GroupRuleUdp && pt != GroupRuleIcmp && pt != GroupRuleGre && pt != GroupRuleAll {
		errors = append(errors, fmt.Errorf("%s must be one of %s %s %s %s %s", k,
			GroupRuleTcp, GroupRuleUdp, GroupRuleIcmp, GroupRuleGre, GroupRuleAll))
	}

	return
}

func validateSecurityRuleNicType(v interface{}, k string) (ws []string, errors []error) {
	pt := GroupRuleNicType(v.(string))
	if pt != GroupRuleInternet && pt != GroupRuleIntranet {
		errors = append(errors, fmt.Errorf("%s must be one of %s %s", k, GroupRuleInternet, GroupRuleIntranet))
	}

	return
}

func validateSecurityRulePolicy(v interface{}, k string) (ws []string, errors []error) {
	pt := GroupRulePolicy(v.(string))
	if pt != GroupRulePolicyAccept && pt != GroupRulePolicyDrop {
		errors = append(errors, fmt.Errorf("%s must be one of %s %s", k, GroupRulePolicyAccept, GroupRulePolicyDrop))
	}

	return
}

func validateSecurityPriority(v interface{}, k string) (ws []string, errors []error) {
	value := v.(int)
	if value < 1 || value > 100 {
		errors = append(errors, fmt.Errorf(
			"%q must be a valid authorization policy priority between 1 and 100",
			k))
		return
	}
	return
}

// validateCIDRNetworkAddress ensures that the string value is a valid CIDR that
// represents a network address - it adds an error otherwise
func validateCIDRNetworkAddress(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	_, ipnet, err := net.ParseCIDR(value)
	if err != nil {
		errors = append(errors, fmt.Errorf(
			"%q must contain a valid CIDR, got error parsing: %s", k, err))
		return
	}

	if ipnet == nil || value != ipnet.String() {
		errors = append(errors, fmt.Errorf(
			"%q must contain a valid network CIDR, expected %q, got %q",
			k, ipnet, value))
	}

	return
}

func validateRouteEntryNextHopType(v interface{}, k string) (ws []string, errors []error) {
	nht := ecs.NextHopType(v.(string))
	if nht != ecs.NextHopIntance && nht != ecs.NextHopTunnel {
		errors = append(errors, fmt.Errorf("%s must be one of %s %s", k,
			ecs.NextHopIntance, ecs.NextHopTunnel))
	}

	return
}

func validateSwitchCIDRNetworkAddress(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	_, ipnet, err := net.ParseCIDR(value)
	if err != nil {
		errors = append(errors, fmt.Errorf(
			"%q must contain a valid CIDR, got error parsing: %s", k, err))
		return
	}

	if ipnet == nil || value != ipnet.String() {
		errors = append(errors, fmt.Errorf(
			"%q must contain a valid network CIDR, expected %q, got %q",
			k, ipnet, value))
		return
	}

	mark, _ := strconv.Atoi(strings.Split(ipnet.String(), "/")[1])
	if mark < 16 || mark > 29 {
		errors = append(errors, fmt.Errorf(
			"%q must contain a network CIDR which mark between 16 and 29",
			k))
	}

	return
}

// validateIoOptimized ensures that the string value is a valid IoOptimized that
// represents a IoOptimized - it adds an error otherwise
func validateIoOptimized(v interface{}, k string) (ws []string, errors []error) {
	if value := v.(string); value != "" {
		ioOptimized := ecs.IoOptimized(value)
		if ioOptimized != ecs.IoOptimizedNone &&
			ioOptimized != ecs.IoOptimizedOptimized {
			errors = append(errors, fmt.Errorf(
				"%q must contain a valid IoOptimized, expected %s or %s, got %q",
				k, ecs.IoOptimizedNone, ecs.IoOptimizedOptimized, ioOptimized))
		}
	}

	return
}

// validateInstanceNetworkType ensures that the string value is a classic or vpc
func validateInstanceNetworkType(v interface{}, k string) (ws []string, errors []error) {
	if value := v.(string); value != "" {
		network := InstanceNetWork(value)
		if network != ClassicNet &&
			network != VpcNet {
			errors = append(errors, fmt.Errorf(
				"%q must contain a valid InstanceNetworkType, expected %s or %s, go %q",
				k, ClassicNet, VpcNet, network))
		}
	}
	return
}

func validateInstanceChargeType(v interface{}, k string) (ws []string, errors []error) {
	if value := v.(string); value != "" {
		chargeType := common.InstanceChargeType(value)
		if chargeType != common.PrePaid &&
			chargeType != common.PostPaid {
			errors = append(errors, fmt.Errorf(
				"%q must contain a valid InstanceChargeType, expected %s or %s, got %q",
				k, common.PrePaid, common.PostPaid, chargeType))
		}
	}

	return
}

func validateInternetChargeType(v interface{}, k string) (ws []string, errors []error) {
	if value := v.(string); value != "" {
		chargeType := common.InternetChargeType(value)
		if chargeType != common.PayByBandwidth &&
			chargeType != common.PayByTraffic {
			errors = append(errors, fmt.Errorf(
				"%q must contain a valid InstanceChargeType, expected %s or %s, got %q",
				k, common.PayByBandwidth, common.PayByTraffic, chargeType))
		}
	}

	return
}

func validateInternetMaxBandWidthOut(v interface{}, k string) (ws []string, errors []error) {
	value := v.(int)
	if value < 0 || value > 100 {
		errors = append(errors, fmt.Errorf(
			"%q must be a valid internet bandwidth out between 0 and 100",
			k))
		return
	}
	return
}

// SLB
func validateSlbName(v interface{}, k string) (ws []string, errors []error) {
	if value := v.(string); value != "" {
		if len(value) < 1 || len(value) > 80 {
			errors = append(errors, fmt.Errorf(
				"%q must be a valid load balancer name characters between 1 and 80",
				k))
			return
		}
	}

	return
}

func validateSlbInternetChargeType(v interface{}, k string) (ws []string, errors []error) {
	if value := v.(string); value != "" {
		chargeType := common.InternetChargeType(value)

		if chargeType != "paybybandwidth" &&
			chargeType != "paybytraffic" {
			errors = append(errors, fmt.Errorf(
				"%q must contain a valid InstanceChargeType, expected %s or %s, got %q",
				k, "paybybandwidth", "paybytraffic", value))
		}
	}

	return
}

func validateSlbBandwidth(v interface{}, k string) (ws []string, errors []error) {
	value := v.(int)
	if value < 1 || value > 1000 {
		errors = append(errors, fmt.Errorf(
			"%q must be a valid load balancer bandwidth between 1 and 1000",
			k))
		return
	}
	return
}

func validateSlbListenerBandwidth(v interface{}, k string) (ws []string, errors []error) {
	value := v.(int)
	if (value < 1 || value > 1000) && value != -1 {
		errors = append(errors, fmt.Errorf(
			"%q must be a valid load balancer bandwidth between 1 and 1000 or -1",
			k))
		return
	}
	return
}

func validateSlbListenerScheduler(v interface{}, k string) (ws []string, errors []error) {
	if value := v.(string); value != "" {
		scheduler := slb.SchedulerType(value)

		if scheduler != "wrr" && scheduler != "wlc" {
			errors = append(errors, fmt.Errorf(
				"%q must contain a valid SchedulerType, expected %s or %s, got %q",
				k, "wrr", "wlc", value))
		}
	}

	return
}

func validateSlbListenerCookie(v interface{}, k string) (ws []string, errors []error) {
	if value := v.(string); value != "" {
		if len(value) < 1 || len(value) > 200 {
			errors = append(errors, fmt.Errorf("%q cannot be longer than 200 characters", k))
		}
	}
	return
}

func validateSlbListenerCookieTimeout(v interface{}, k string) (ws []string, errors []error) {
	value := v.(int)
	if value < 0 || value > 86400 {
		errors = append(errors, fmt.Errorf(
			"%q must be a valid load balancer cookie timeout between 0 and 86400",
			k))
		return
	}
	return
}

func validateSlbListenerPersistenceTimeout(v interface{}, k string) (ws []string, errors []error) {
	value := v.(int)
	if value < 0 || value > 3600 {
		errors = append(errors, fmt.Errorf(
			"%q must be a valid load balancer persistence timeout between 0 and 86400",
			k))
		return
	}
	return
}

func validateSlbListenerHealthCheckDomain(v interface{}, k string) (ws []string, errors []error) {
	if value := v.(string); value != "" {
		//the len add "$_ip",so to max is 84
		if len(value) < 1 || len(value) > 84 {
			errors = append(errors, fmt.Errorf("%q cannot be longer than 84 characters", k))
		}
	}
	return
}

func validateSlbListenerHealthCheckUri(v interface{}, k string) (ws []string, errors []error) {
	if value := v.(string); value != "" {
		if len(value) < 1 || len(value) > 80 {
			errors = append(errors, fmt.Errorf("%q cannot be longer than 80 characters", k))
		}
	}
	return
}

func validateSlbListenerHealthCheckConnectPort(v interface{}, k string) (ws []string, errors []error) {
	value := v.(int)
	if value < 1 || value > 65535 {
		if value != -520 {
			errors = append(errors, fmt.Errorf(
				"%q must be a valid load balancer health check connect port between 1 and 65535 or -520",
				k))
			return
		}

	}
	return
}

func validateDBBackupPeriod(v interface{}, k string) (ws []string, errors []error) {
	days := []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
	value := v.(string)
	exist := false
	for _, d := range days {
		if value == d {
			exist = true
			break
		}
	}
	if !exist {
		errors = append(errors, fmt.Errorf(
			"%q must contain a valid backup period value should in array %#v, got %q",
			k, days, value))
	}

	return
}

func validateAllowedStringValue(ss []string) schema.SchemaValidateFunc {
	return func(v interface{}, k string) (ws []string, errors []error) {
		value := v.(string)
		existed := false
		for _, s := range ss {
			if s == value {
				existed = true
				break
			}
		}
		if !existed {
			errors = append(errors, fmt.Errorf(
				"%q must contain a valid string value should in array %#v, got %q",
				k, ss, value))
		}
		return

	}
}

func validateAllowedSplitStringValue(ss []string, splitStr string) schema.SchemaValidateFunc {
	return func(v interface{}, k string) (ws []string, errors []error) {
		value := v.(string)
		existed := false
		tsList := strings.Split(value, splitStr)

		for _, ts := range tsList {
			existed = false
			for _, s := range ss {
				if ts == s {
					existed = true
					break
				}
			}
		}
		if !existed {
			errors = append(errors, fmt.Errorf(
				"%q must contain a valid string value should in %#v, got %q",
				k, ss, value))
		}
		return

	}
}

func validateAllowedIntValue(is []int) schema.SchemaValidateFunc {
	return func(v interface{}, k string) (ws []string, errors []error) {
		value := v.(int)
		existed := false
		for _, i := range is {
			if i == value {
				existed = true
				break
			}
		}
		if !existed {
			errors = append(errors, fmt.Errorf(
				"%q must contain a valid int value should in array %#v, got %q",
				k, is, value))
		}
		return

	}
}

func validateIntegerInRange(min, max int) schema.SchemaValidateFunc {
	return func(v interface{}, k string) (ws []string, errors []error) {
		value := v.(int)
		if value < min {
			errors = append(errors, fmt.Errorf(
				"%q cannot be lower than %d: %d", k, min, value))
		}
		if value > max {
			errors = append(errors, fmt.Errorf(
				"%q cannot be higher than %d: %d", k, max, value))
		}
		return
	}
}

//data source validate func
//data_source_alicloud_image
func validateNameRegex(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)

	if _, err := regexp.Compile(value); err != nil {
		errors = append(errors, fmt.Errorf(
			"%q contains an invalid regular expression: %s",
			k, err))
	}
	return
}

func validateImageOwners(v interface{}, k string) (ws []string, errors []error) {
	if value := v.(string); value != "" {
		owners := ecs.ImageOwnerAlias(value)
		if owners != ecs.ImageOwnerSystem &&
			owners != ecs.ImageOwnerSelf &&
			owners != ecs.ImageOwnerOthers &&
			owners != ecs.ImageOwnerMarketplace &&
			owners != ecs.ImageOwnerDefault {
			errors = append(errors, fmt.Errorf(
				"%q must contain a valid Image owner , expected %s, %s, %s, %s or %s, got %q",
				k, ecs.ImageOwnerSystem, ecs.ImageOwnerSelf, ecs.ImageOwnerOthers, ecs.ImageOwnerMarketplace, ecs.ImageOwnerDefault, owners))
		}
	}
	return
}

func validateRegion(v interface{}, k string) (ws []string, errors []error) {
	if value := v.(string); value != "" {
		region := common.Region(value)
		var valid string
		for _, re := range common.ValidRegions {
			if region == re {
				return
			}
			valid = valid + ", " + string(re)
		}
		errors = append(errors, fmt.Errorf(
			"%q must contain a valid Region ID , expected %#v, got %q",
			k, valid, value))

	}
	return
}

func validateForwardPort(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if value != "any" {
		valueConv, err := strconv.Atoi(value)
		if err != nil || valueConv < 1 || valueConv > 65535 {
			errors = append(errors, fmt.Errorf("%q must be a valid port between 1 and 65535 or any ", k))
		}
	}
	return
}
