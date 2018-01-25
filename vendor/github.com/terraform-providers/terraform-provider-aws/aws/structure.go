package aws

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/private/protocol/json/jsonutil"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/cognitoidentity"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go/service/configservice"
	"github.com/aws/aws-sdk-go/service/directoryservice"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/aws/aws-sdk-go/service/elasticbeanstalk"
	elasticsearch "github.com/aws/aws-sdk-go/service/elasticsearchservice"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/mq"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/redshift"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/waf"
	"github.com/beevik/etree"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/structure"
	"gopkg.in/yaml.v2"
)

// Takes the result of flatmap.Expand for an array of listeners and
// returns ELB API compatible objects
func expandListeners(configured []interface{}) ([]*elb.Listener, error) {
	listeners := make([]*elb.Listener, 0, len(configured))

	// Loop over our configured listeners and create
	// an array of aws-sdk-go compatible objects
	for _, lRaw := range configured {
		data := lRaw.(map[string]interface{})

		ip := int64(data["instance_port"].(int))
		lp := int64(data["lb_port"].(int))
		l := &elb.Listener{
			InstancePort:     &ip,
			InstanceProtocol: aws.String(data["instance_protocol"].(string)),
			LoadBalancerPort: &lp,
			Protocol:         aws.String(data["lb_protocol"].(string)),
		}

		if v, ok := data["ssl_certificate_id"]; ok {
			l.SSLCertificateId = aws.String(v.(string))
		}

		var valid bool
		if l.SSLCertificateId != nil && *l.SSLCertificateId != "" {
			// validate the protocol is correct
			for _, p := range []string{"https", "ssl"} {
				if (strings.ToLower(*l.InstanceProtocol) == p) || (strings.ToLower(*l.Protocol) == p) {
					valid = true
				}
			}
		} else {
			valid = true
		}

		if valid {
			listeners = append(listeners, l)
		} else {
			return nil, fmt.Errorf("[ERR] ELB Listener: ssl_certificate_id may be set only when protocol is 'https' or 'ssl'")
		}
	}

	return listeners, nil
}

// Takes the result of flatmap. Expand for an array of listeners and
// returns ECS Volume compatible objects
func expandEcsVolumes(configured []interface{}) ([]*ecs.Volume, error) {
	volumes := make([]*ecs.Volume, 0, len(configured))

	// Loop over our configured volumes and create
	// an array of aws-sdk-go compatible objects
	for _, lRaw := range configured {
		data := lRaw.(map[string]interface{})

		l := &ecs.Volume{
			Name: aws.String(data["name"].(string)),
		}

		hostPath := data["host_path"].(string)
		if hostPath != "" {
			l.Host = &ecs.HostVolumeProperties{
				SourcePath: aws.String(hostPath),
			}
		}

		volumes = append(volumes, l)
	}

	return volumes, nil
}

// Takes JSON in a string. Decodes JSON into
// an array of ecs.ContainerDefinition compatible objects
func expandEcsContainerDefinitions(rawDefinitions string) ([]*ecs.ContainerDefinition, error) {
	var definitions []*ecs.ContainerDefinition

	err := json.Unmarshal([]byte(rawDefinitions), &definitions)
	if err != nil {
		return nil, fmt.Errorf("Error decoding JSON: %s", err)
	}

	return definitions, nil
}

// Takes the result of flatmap. Expand for an array of load balancers and
// returns ecs.LoadBalancer compatible objects
func expandEcsLoadBalancers(configured []interface{}) []*ecs.LoadBalancer {
	loadBalancers := make([]*ecs.LoadBalancer, 0, len(configured))

	// Loop over our configured load balancers and create
	// an array of aws-sdk-go compatible objects
	for _, lRaw := range configured {
		data := lRaw.(map[string]interface{})

		l := &ecs.LoadBalancer{
			ContainerName: aws.String(data["container_name"].(string)),
			ContainerPort: aws.Int64(int64(data["container_port"].(int))),
		}

		if v, ok := data["elb_name"]; ok && v.(string) != "" {
			l.LoadBalancerName = aws.String(v.(string))
		}
		if v, ok := data["target_group_arn"]; ok && v.(string) != "" {
			l.TargetGroupArn = aws.String(v.(string))
		}

		loadBalancers = append(loadBalancers, l)
	}

	return loadBalancers
}

// Takes the result of flatmap.Expand for an array of ingress/egress security
// group rules and returns EC2 API compatible objects. This function will error
// if it finds invalid permissions input, namely a protocol of "-1" with either
// to_port or from_port set to a non-zero value.
func expandIPPerms(
	group *ec2.SecurityGroup, configured []interface{}) ([]*ec2.IpPermission, error) {
	vpc := group.VpcId != nil && *group.VpcId != ""

	perms := make([]*ec2.IpPermission, len(configured))
	for i, mRaw := range configured {
		var perm ec2.IpPermission
		m := mRaw.(map[string]interface{})

		perm.FromPort = aws.Int64(int64(m["from_port"].(int)))
		perm.ToPort = aws.Int64(int64(m["to_port"].(int)))
		perm.IpProtocol = aws.String(m["protocol"].(string))

		// When protocol is "-1", AWS won't store any ports for the
		// rule, but also won't error if the user specifies ports other
		// than '0'. Force the user to make a deliberate '0' port
		// choice when specifying a "-1" protocol, and tell them about
		// AWS's behavior in the error message.
		if *perm.IpProtocol == "-1" && (*perm.FromPort != 0 || *perm.ToPort != 0) {
			return nil, fmt.Errorf(
				"from_port (%d) and to_port (%d) must both be 0 to use the 'ALL' \"-1\" protocol!",
				*perm.FromPort, *perm.ToPort)
		}

		var groups []string
		if raw, ok := m["security_groups"]; ok {
			list := raw.(*schema.Set).List()
			for _, v := range list {
				groups = append(groups, v.(string))
			}
		}
		if v, ok := m["self"]; ok && v.(bool) {
			if vpc {
				groups = append(groups, *group.GroupId)
			} else {
				groups = append(groups, *group.GroupName)
			}
		}

		if len(groups) > 0 {
			perm.UserIdGroupPairs = make([]*ec2.UserIdGroupPair, len(groups))
			for i, name := range groups {
				ownerId, id := "", name
				if items := strings.Split(id, "/"); len(items) > 1 {
					ownerId, id = items[0], items[1]
				}

				perm.UserIdGroupPairs[i] = &ec2.UserIdGroupPair{
					GroupId: aws.String(id),
				}

				if ownerId != "" {
					perm.UserIdGroupPairs[i].UserId = aws.String(ownerId)
				}

				if !vpc {
					perm.UserIdGroupPairs[i].GroupId = nil
					perm.UserIdGroupPairs[i].GroupName = aws.String(id)
				}
			}
		}

		if raw, ok := m["cidr_blocks"]; ok {
			list := raw.([]interface{})
			for _, v := range list {
				perm.IpRanges = append(perm.IpRanges, &ec2.IpRange{CidrIp: aws.String(v.(string))})
			}
		}
		if raw, ok := m["ipv6_cidr_blocks"]; ok {
			list := raw.([]interface{})
			for _, v := range list {
				perm.Ipv6Ranges = append(perm.Ipv6Ranges, &ec2.Ipv6Range{CidrIpv6: aws.String(v.(string))})
			}
		}

		if raw, ok := m["prefix_list_ids"]; ok {
			list := raw.([]interface{})
			for _, v := range list {
				perm.PrefixListIds = append(perm.PrefixListIds, &ec2.PrefixListId{PrefixListId: aws.String(v.(string))})
			}
		}

		if raw, ok := m["description"]; ok {
			description := raw.(string)
			if description != "" {
				for _, v := range perm.IpRanges {
					v.Description = aws.String(description)
				}
				for _, v := range perm.Ipv6Ranges {
					v.Description = aws.String(description)
				}
				for _, v := range perm.PrefixListIds {
					v.Description = aws.String(description)
				}
				for _, v := range perm.UserIdGroupPairs {
					v.Description = aws.String(description)
				}
			}
		}

		perms[i] = &perm
	}

	return perms, nil
}

// Takes the result of flatmap.Expand for an array of parameters and
// returns Parameter API compatible objects
func expandParameters(configured []interface{}) ([]*rds.Parameter, error) {
	var parameters []*rds.Parameter

	// Loop over our configured parameters and create
	// an array of aws-sdk-go compatible objects
	for _, pRaw := range configured {
		data := pRaw.(map[string]interface{})

		if data["name"].(string) == "" {
			continue
		}

		p := &rds.Parameter{
			ApplyMethod:    aws.String(data["apply_method"].(string)),
			ParameterName:  aws.String(data["name"].(string)),
			ParameterValue: aws.String(data["value"].(string)),
		}

		parameters = append(parameters, p)
	}

	return parameters, nil
}

func expandRedshiftParameters(configured []interface{}) ([]*redshift.Parameter, error) {
	var parameters []*redshift.Parameter

	// Loop over our configured parameters and create
	// an array of aws-sdk-go compatible objects
	for _, pRaw := range configured {
		data := pRaw.(map[string]interface{})

		if data["name"].(string) == "" {
			continue
		}

		p := &redshift.Parameter{
			ParameterName:  aws.String(data["name"].(string)),
			ParameterValue: aws.String(data["value"].(string)),
		}

		parameters = append(parameters, p)
	}

	return parameters, nil
}

func expandOptionConfiguration(configured []interface{}) ([]*rds.OptionConfiguration, error) {
	var option []*rds.OptionConfiguration

	for _, pRaw := range configured {
		data := pRaw.(map[string]interface{})

		o := &rds.OptionConfiguration{
			OptionName: aws.String(data["option_name"].(string)),
		}

		if raw, ok := data["port"]; ok {
			port := raw.(int)
			if port != 0 {
				o.Port = aws.Int64(int64(port))
			}
		}

		if raw, ok := data["db_security_group_memberships"]; ok {
			memberships := expandStringList(raw.(*schema.Set).List())
			if len(memberships) > 0 {
				o.DBSecurityGroupMemberships = memberships
			}
		}

		if raw, ok := data["vpc_security_group_memberships"]; ok {
			memberships := expandStringList(raw.(*schema.Set).List())
			if len(memberships) > 0 {
				o.VpcSecurityGroupMemberships = memberships
			}
		}

		if raw, ok := data["option_settings"]; ok {
			o.OptionSettings = expandOptionSetting(raw.(*schema.Set).List())
		}

		option = append(option, o)
	}

	return option, nil
}

func expandOptionSetting(list []interface{}) []*rds.OptionSetting {
	options := make([]*rds.OptionSetting, 0, len(list))

	for _, oRaw := range list {
		data := oRaw.(map[string]interface{})

		o := &rds.OptionSetting{
			Name:  aws.String(data["name"].(string)),
			Value: aws.String(data["value"].(string)),
		}

		options = append(options, o)
	}

	return options
}

// Takes the result of flatmap.Expand for an array of parameters and
// returns Parameter API compatible objects
func expandElastiCacheParameters(configured []interface{}) ([]*elasticache.ParameterNameValue, error) {
	parameters := make([]*elasticache.ParameterNameValue, 0, len(configured))

	// Loop over our configured parameters and create
	// an array of aws-sdk-go compatible objects
	for _, pRaw := range configured {
		data := pRaw.(map[string]interface{})

		p := &elasticache.ParameterNameValue{
			ParameterName:  aws.String(data["name"].(string)),
			ParameterValue: aws.String(data["value"].(string)),
		}

		parameters = append(parameters, p)
	}

	return parameters, nil
}

// Flattens an access log into something that flatmap.Flatten() can handle
func flattenAccessLog(l *elb.AccessLog) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, 1)

	if l == nil {
		return nil
	}

	r := make(map[string]interface{})
	if l.S3BucketName != nil {
		r["bucket"] = *l.S3BucketName
	}

	if l.S3BucketPrefix != nil {
		r["bucket_prefix"] = *l.S3BucketPrefix
	}

	if l.EmitInterval != nil {
		r["interval"] = *l.EmitInterval
	}

	if l.Enabled != nil {
		r["enabled"] = *l.Enabled
	}

	result = append(result, r)

	return result
}

// Takes the result of flatmap.Expand for an array of step adjustments and
// returns a []*autoscaling.StepAdjustment.
func expandStepAdjustments(configured []interface{}) ([]*autoscaling.StepAdjustment, error) {
	var adjustments []*autoscaling.StepAdjustment

	// Loop over our configured step adjustments and create an array
	// of aws-sdk-go compatible objects. We're forced to convert strings
	// to floats here because there's no way to detect whether or not
	// an uninitialized, optional schema element is "0.0" deliberately.
	// With strings, we can test for "", which is definitely an empty
	// struct value.
	for _, raw := range configured {
		data := raw.(map[string]interface{})
		a := &autoscaling.StepAdjustment{
			ScalingAdjustment: aws.Int64(int64(data["scaling_adjustment"].(int))),
		}
		if data["metric_interval_lower_bound"] != "" {
			bound := data["metric_interval_lower_bound"]
			switch bound := bound.(type) {
			case string:
				f, err := strconv.ParseFloat(bound, 64)
				if err != nil {
					return nil, fmt.Errorf(
						"metric_interval_lower_bound must be a float value represented as a string")
				}
				a.MetricIntervalLowerBound = aws.Float64(f)
			default:
				return nil, fmt.Errorf(
					"metric_interval_lower_bound isn't a string. This is a bug. Please file an issue.")
			}
		}
		if data["metric_interval_upper_bound"] != "" {
			bound := data["metric_interval_upper_bound"]
			switch bound := bound.(type) {
			case string:
				f, err := strconv.ParseFloat(bound, 64)
				if err != nil {
					return nil, fmt.Errorf(
						"metric_interval_upper_bound must be a float value represented as a string")
				}
				a.MetricIntervalUpperBound = aws.Float64(f)
			default:
				return nil, fmt.Errorf(
					"metric_interval_upper_bound isn't a string. This is a bug. Please file an issue.")
			}
		}
		adjustments = append(adjustments, a)
	}

	return adjustments, nil
}

// Flattens a health check into something that flatmap.Flatten()
// can handle
func flattenHealthCheck(check *elb.HealthCheck) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, 1)

	chk := make(map[string]interface{})
	chk["unhealthy_threshold"] = *check.UnhealthyThreshold
	chk["healthy_threshold"] = *check.HealthyThreshold
	chk["target"] = *check.Target
	chk["timeout"] = *check.Timeout
	chk["interval"] = *check.Interval

	result = append(result, chk)

	return result
}

// Flattens an array of UserSecurityGroups into a []*GroupIdentifier
func flattenSecurityGroups(list []*ec2.UserIdGroupPair, ownerId *string) []*GroupIdentifier {
	result := make([]*GroupIdentifier, 0, len(list))
	for _, g := range list {
		var userId *string
		if g.UserId != nil && *g.UserId != "" && (ownerId == nil || *ownerId != *g.UserId) {
			userId = g.UserId
		}
		// userid nil here for same vpc groups

		vpc := g.GroupName == nil || *g.GroupName == ""
		var id *string
		if vpc {
			id = g.GroupId
		} else {
			id = g.GroupName
		}

		// id is groupid for vpcs
		// id is groupname for non vpc (classic)

		if userId != nil {
			id = aws.String(*userId + "/" + *id)
		}

		if vpc {
			result = append(result, &GroupIdentifier{
				GroupId:     id,
				Description: g.Description,
			})
		} else {
			result = append(result, &GroupIdentifier{
				GroupId:     g.GroupId,
				GroupName:   id,
				Description: g.Description,
			})
		}
	}
	return result
}

// Flattens an array of Instances into a []string
func flattenInstances(list []*elb.Instance) []string {
	result := make([]string, 0, len(list))
	for _, i := range list {
		result = append(result, *i.InstanceId)
	}
	return result
}

// Expands an array of String Instance IDs into a []Instances
func expandInstanceString(list []interface{}) []*elb.Instance {
	result := make([]*elb.Instance, 0, len(list))
	for _, i := range list {
		result = append(result, &elb.Instance{InstanceId: aws.String(i.(string))})
	}
	return result
}

// Flattens an array of Backend Descriptions into a a map of instance_port to policy names.
func flattenBackendPolicies(backends []*elb.BackendServerDescription) map[int64][]string {
	policies := make(map[int64][]string)
	for _, i := range backends {
		for _, p := range i.PolicyNames {
			policies[*i.InstancePort] = append(policies[*i.InstancePort], *p)
		}
		sort.Strings(policies[*i.InstancePort])
	}
	return policies
}

// Flattens an array of Listeners into a []map[string]interface{}
func flattenListeners(list []*elb.ListenerDescription) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(list))
	for _, i := range list {
		l := map[string]interface{}{
			"instance_port":     *i.Listener.InstancePort,
			"instance_protocol": strings.ToLower(*i.Listener.InstanceProtocol),
			"lb_port":           *i.Listener.LoadBalancerPort,
			"lb_protocol":       strings.ToLower(*i.Listener.Protocol),
		}
		// SSLCertificateID is optional, and may be nil
		if i.Listener.SSLCertificateId != nil {
			l["ssl_certificate_id"] = *i.Listener.SSLCertificateId
		}
		result = append(result, l)
	}
	return result
}

// Flattens an array of Volumes into a []map[string]interface{}
func flattenEcsVolumes(list []*ecs.Volume) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(list))
	for _, volume := range list {
		l := map[string]interface{}{
			"name": *volume.Name,
		}

		if volume.Host.SourcePath != nil {
			l["host_path"] = *volume.Host.SourcePath
		}

		result = append(result, l)
	}
	return result
}

// Flattens an array of ECS LoadBalancers into a []map[string]interface{}
func flattenEcsLoadBalancers(list []*ecs.LoadBalancer) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(list))
	for _, loadBalancer := range list {
		l := map[string]interface{}{
			"container_name": *loadBalancer.ContainerName,
			"container_port": *loadBalancer.ContainerPort,
		}

		if loadBalancer.LoadBalancerName != nil {
			l["elb_name"] = *loadBalancer.LoadBalancerName
		}

		if loadBalancer.TargetGroupArn != nil {
			l["target_group_arn"] = *loadBalancer.TargetGroupArn
		}

		result = append(result, l)
	}
	return result
}

// Encodes an array of ecs.ContainerDefinitions into a JSON string
func flattenEcsContainerDefinitions(definitions []*ecs.ContainerDefinition) (string, error) {
	b, err := jsonutil.BuildJSON(definitions)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

// Flattens an array of Options into a []map[string]interface{}
func flattenOptions(list []*rds.Option) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(list))
	for _, i := range list {
		if i.OptionName != nil {
			r := make(map[string]interface{})
			r["option_name"] = strings.ToLower(*i.OptionName)
			// Default empty string, guard against nil parameter values
			r["port"] = ""
			if i.Port != nil {
				r["port"] = int(*i.Port)
			}
			if i.VpcSecurityGroupMemberships != nil {
				vpcs := make([]string, 0, len(i.VpcSecurityGroupMemberships))
				for _, vpc := range i.VpcSecurityGroupMemberships {
					id := vpc.VpcSecurityGroupId
					vpcs = append(vpcs, *id)
				}

				r["vpc_security_group_memberships"] = vpcs
			}
			if i.DBSecurityGroupMemberships != nil {
				dbs := make([]string, 0, len(i.DBSecurityGroupMemberships))
				for _, db := range i.DBSecurityGroupMemberships {
					id := db.DBSecurityGroupName
					dbs = append(dbs, *id)
				}

				r["db_security_group_memberships"] = dbs
			}
			if i.OptionSettings != nil {
				settings := make([]map[string]interface{}, 0, len(i.OptionSettings))
				for _, j := range i.OptionSettings {
					setting := map[string]interface{}{
						"name": *j.Name,
					}
					if j.Value != nil {
						setting["value"] = *j.Value
					}

					settings = append(settings, setting)
				}

				r["option_settings"] = settings
			}
			result = append(result, r)
		}
	}
	return result
}

// Flattens an array of Parameters into a []map[string]interface{}
func flattenParameters(list []*rds.Parameter) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(list))
	for _, i := range list {
		if i.ParameterName != nil {
			r := make(map[string]interface{})
			r["name"] = strings.ToLower(*i.ParameterName)
			// Default empty string, guard against nil parameter values
			r["value"] = ""
			if i.ParameterValue != nil {
				r["value"] = strings.ToLower(*i.ParameterValue)
			}
			if i.ApplyMethod != nil {
				r["apply_method"] = strings.ToLower(*i.ApplyMethod)
			}

			result = append(result, r)
		}
	}
	return result
}

// Flattens an array of Redshift Parameters into a []map[string]interface{}
func flattenRedshiftParameters(list []*redshift.Parameter) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(list))
	for _, i := range list {
		result = append(result, map[string]interface{}{
			"name":  strings.ToLower(*i.ParameterName),
			"value": strings.ToLower(*i.ParameterValue),
		})
	}
	return result
}

// Flattens an array of Parameters into a []map[string]interface{}
func flattenElastiCacheParameters(list []*elasticache.Parameter) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(list))
	for _, i := range list {
		if i.ParameterValue != nil {
			result = append(result, map[string]interface{}{
				"name":  strings.ToLower(*i.ParameterName),
				"value": *i.ParameterValue,
			})
		}
	}
	return result
}

// Takes the result of flatmap.Expand for an array of strings
// and returns a []*string
func expandStringList(configured []interface{}) []*string {
	vs := make([]*string, 0, len(configured))
	for _, v := range configured {
		val, ok := v.(string)
		if ok && val != "" {
			vs = append(vs, aws.String(v.(string)))
		}
	}
	return vs
}

// Takes the result of schema.Set of strings and returns a []*string
func expandStringSet(configured *schema.Set) []*string {
	return expandStringList(configured.List())
}

// Takes list of pointers to strings. Expand to an array
// of raw strings and returns a []interface{}
// to keep compatibility w/ schema.NewSetschema.NewSet
func flattenStringList(list []*string) []interface{} {
	vs := make([]interface{}, 0, len(list))
	for _, v := range list {
		vs = append(vs, *v)
	}
	return vs
}

//Flattens an array of private ip addresses into a []string, where the elements returned are the IP strings e.g. "192.168.0.0"
func flattenNetworkInterfacesPrivateIPAddresses(dtos []*ec2.NetworkInterfacePrivateIpAddress) []string {
	ips := make([]string, 0, len(dtos))
	for _, v := range dtos {
		ip := *v.PrivateIpAddress
		ips = append(ips, ip)
	}
	return ips
}

//Flattens security group identifiers into a []string, where the elements returned are the GroupIDs
func flattenGroupIdentifiers(dtos []*ec2.GroupIdentifier) []string {
	ids := make([]string, 0, len(dtos))
	for _, v := range dtos {
		group_id := *v.GroupId
		ids = append(ids, group_id)
	}
	return ids
}

//Expands an array of IPs into a ec2 Private IP Address Spec
func expandPrivateIPAddresses(ips []interface{}) []*ec2.PrivateIpAddressSpecification {
	dtos := make([]*ec2.PrivateIpAddressSpecification, 0, len(ips))
	for i, v := range ips {
		new_private_ip := &ec2.PrivateIpAddressSpecification{
			PrivateIpAddress: aws.String(v.(string)),
		}

		new_private_ip.Primary = aws.Bool(i == 0)

		dtos = append(dtos, new_private_ip)
	}
	return dtos
}

//Flattens network interface attachment into a map[string]interface
func flattenAttachment(a *ec2.NetworkInterfaceAttachment) map[string]interface{} {
	att := make(map[string]interface{})
	if a.InstanceId != nil {
		att["instance"] = *a.InstanceId
	}
	att["device_index"] = *a.DeviceIndex
	att["attachment_id"] = *a.AttachmentId
	return att
}

func flattenEc2NetworkInterfaceAssociation(a *ec2.NetworkInterfaceAssociation) []interface{} {
	att := make(map[string]interface{})
	if a.AllocationId != nil {
		att["allocation_id"] = *a.AllocationId
	}
	if a.AssociationId != nil {
		att["association_id"] = *a.AssociationId
	}
	if a.IpOwnerId != nil {
		att["ip_owner_id"] = *a.IpOwnerId
	}
	if a.PublicDnsName != nil {
		att["public_dns_name"] = *a.PublicDnsName
	}
	if a.PublicIp != nil {
		att["public_ip"] = *a.PublicIp
	}
	return []interface{}{att}
}

func flattenEc2NetworkInterfaceIpv6Address(niia []*ec2.NetworkInterfaceIpv6Address) []string {
	ips := make([]string, 0, len(niia))
	for _, v := range niia {
		ips = append(ips, *v.Ipv6Address)
	}
	return ips
}

func flattenElastiCacheSecurityGroupNames(securityGroups []*elasticache.CacheSecurityGroupMembership) []string {
	result := make([]string, 0, len(securityGroups))
	for _, sg := range securityGroups {
		if sg.CacheSecurityGroupName != nil {
			result = append(result, *sg.CacheSecurityGroupName)
		}
	}
	return result
}

func flattenElastiCacheSecurityGroupIds(securityGroups []*elasticache.SecurityGroupMembership) []string {
	result := make([]string, 0, len(securityGroups))
	for _, sg := range securityGroups {
		if sg.SecurityGroupId != nil {
			result = append(result, *sg.SecurityGroupId)
		}
	}
	return result
}

// Flattens step adjustments into a list of map[string]interface.
func flattenStepAdjustments(adjustments []*autoscaling.StepAdjustment) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(adjustments))
	for _, raw := range adjustments {
		a := map[string]interface{}{
			"scaling_adjustment": *raw.ScalingAdjustment,
		}
		if raw.MetricIntervalUpperBound != nil {
			a["metric_interval_upper_bound"] = *raw.MetricIntervalUpperBound
		}
		if raw.MetricIntervalLowerBound != nil {
			a["metric_interval_lower_bound"] = *raw.MetricIntervalLowerBound
		}
		result = append(result, a)
	}
	return result
}

func flattenResourceRecords(recs []*route53.ResourceRecord, typeStr string) []string {
	strs := make([]string, 0, len(recs))
	for _, r := range recs {
		if r.Value != nil {
			s := *r.Value
			if typeStr == "TXT" || typeStr == "SPF" {
				s = expandTxtEntry(s)
			}
			strs = append(strs, s)
		}
	}
	return strs
}

func expandResourceRecords(recs []interface{}, typeStr string) []*route53.ResourceRecord {
	records := make([]*route53.ResourceRecord, 0, len(recs))
	for _, r := range recs {
		s := r.(string)
		if typeStr == "TXT" || typeStr == "SPF" {
			s = flattenTxtEntry(s)
		}
		records = append(records, &route53.ResourceRecord{Value: aws.String(s)})
	}
	return records
}

// How 'flattenTxtEntry' and 'expandTxtEntry' work.
//
// In the Route 53, TXT entries are written using quoted strings, one per line.
// Example:
//     "x=foo"
//     "bar=12"
//
// In Terraform, there are two differences:
// - We use a list of strings instead of separating strings with newlines.
// - Within each string, we dont' include the surrounding quotes.
// Example:
//     records = ["x=foo", "bar=12"]    # Instead of ["\"x=foo\", \"bar=12\""]
//
// When we pull from Route 53, `expandTxtEntry` removes the surrounding quotes;
// when we push to Route 53, `flattenTxtEntry` adds them back.
//
// One complication is that a single TXT entry can have multiple quoted strings.
// For example, here are two TXT entries, one with two quoted strings and the
// other with three.
//     "x=" "foo"
//     "ba" "r" "=12"
//
// DNS clients are expected to merge the quoted strings before interpreting the
// value.  Since `expandTxtEntry` only removes the quotes at the end we can still
// (hackily) represent the above configuration in Terraform:
//      records = ["x=\" \"foo", "ba\" \"r\" \"=12"]
//
// The primary reason to use multiple strings for an entry is that DNS (and Route
// 53) doesn't allow a quoted string to be more than 255 characters long.  If you
// want a longer TXT entry, you must use multiple quoted strings.
//
// It would be nice if this Terraform automatically split strings longer than 255
// characters.  For example, imagine "xxx..xxx" has 256 "x" characters.
//      records = ["xxx..xxx"]
// When pushing to Route 53, this could be converted to:
//      "xxx..xx" "x"
//
// This could also work when the user is already using multiple quoted strings:
//      records = ["xxx.xxx\" \"yyy..yyy"]
// When pushing to Route 53, this could be converted to:
//       "xxx..xx" "xyyy...y" "yy"
//
// If you want to add this feature, make sure to follow all the quoting rules in
// <https://tools.ietf.org/html/rfc1464#section-2>.  If you make a mistake, people
// might end up relying on that mistake so fixing it would be a breaking change.

func flattenTxtEntry(s string) string {
	return fmt.Sprintf(`"%s"`, s)
}

func expandTxtEntry(s string) string {
	last := len(s) - 1
	if last != 0 && s[0] == '"' && s[last] == '"' {
		s = s[1:last]
	}
	return s
}

func expandESClusterConfig(m map[string]interface{}) *elasticsearch.ElasticsearchClusterConfig {
	config := elasticsearch.ElasticsearchClusterConfig{}

	if v, ok := m["dedicated_master_enabled"]; ok {
		isEnabled := v.(bool)
		config.DedicatedMasterEnabled = aws.Bool(isEnabled)

		if isEnabled {
			if v, ok := m["dedicated_master_count"]; ok && v.(int) > 0 {
				config.DedicatedMasterCount = aws.Int64(int64(v.(int)))
			}
			if v, ok := m["dedicated_master_type"]; ok && v.(string) != "" {
				config.DedicatedMasterType = aws.String(v.(string))
			}
		}
	}

	if v, ok := m["instance_count"]; ok {
		config.InstanceCount = aws.Int64(int64(v.(int)))
	}
	if v, ok := m["instance_type"]; ok {
		config.InstanceType = aws.String(v.(string))
	}

	if v, ok := m["zone_awareness_enabled"]; ok {
		config.ZoneAwarenessEnabled = aws.Bool(v.(bool))
	}

	return &config
}

func flattenESClusterConfig(c *elasticsearch.ElasticsearchClusterConfig) []map[string]interface{} {
	m := map[string]interface{}{}

	if c.DedicatedMasterCount != nil {
		m["dedicated_master_count"] = *c.DedicatedMasterCount
	}
	if c.DedicatedMasterEnabled != nil {
		m["dedicated_master_enabled"] = *c.DedicatedMasterEnabled
	}
	if c.DedicatedMasterType != nil {
		m["dedicated_master_type"] = *c.DedicatedMasterType
	}
	if c.InstanceCount != nil {
		m["instance_count"] = *c.InstanceCount
	}
	if c.InstanceType != nil {
		m["instance_type"] = *c.InstanceType
	}
	if c.ZoneAwarenessEnabled != nil {
		m["zone_awareness_enabled"] = *c.ZoneAwarenessEnabled
	}

	return []map[string]interface{}{m}
}

func flattenESEBSOptions(o *elasticsearch.EBSOptions) []map[string]interface{} {
	m := map[string]interface{}{}

	if o.EBSEnabled != nil {
		m["ebs_enabled"] = *o.EBSEnabled
	}
	if o.Iops != nil {
		m["iops"] = *o.Iops
	}
	if o.VolumeSize != nil {
		m["volume_size"] = *o.VolumeSize
	}
	if o.VolumeType != nil {
		m["volume_type"] = *o.VolumeType
	}

	return []map[string]interface{}{m}
}

func expandESEBSOptions(m map[string]interface{}) *elasticsearch.EBSOptions {
	options := elasticsearch.EBSOptions{}

	if v, ok := m["ebs_enabled"]; ok {
		options.EBSEnabled = aws.Bool(v.(bool))
	}
	if v, ok := m["iops"]; ok && v.(int) > 0 {
		options.Iops = aws.Int64(int64(v.(int)))
	}
	if v, ok := m["volume_size"]; ok && v.(int) > 0 {
		options.VolumeSize = aws.Int64(int64(v.(int)))
	}
	if v, ok := m["volume_type"]; ok && v.(string) != "" {
		options.VolumeType = aws.String(v.(string))
	}

	return &options
}

func flattenESEncryptAtRestOptions(o *elasticsearch.EncryptionAtRestOptions) []map[string]interface{} {
	if o == nil {
		return []map[string]interface{}{}
	}

	m := map[string]interface{}{}

	if o.Enabled != nil {
		m["enabled"] = *o.Enabled
	}
	if o.KmsKeyId != nil {
		m["kms_key_id"] = *o.KmsKeyId
	}

	return []map[string]interface{}{m}
}

func expandESEncryptAtRestOptions(m map[string]interface{}) *elasticsearch.EncryptionAtRestOptions {
	options := elasticsearch.EncryptionAtRestOptions{}

	if v, ok := m["enabled"]; ok {
		options.Enabled = aws.Bool(v.(bool))
	}
	if v, ok := m["kms_key_id"]; ok && v.(string) != "" {
		options.KmsKeyId = aws.String(v.(string))
	}

	return &options
}

func flattenESVPCDerivedInfo(o *elasticsearch.VPCDerivedInfo) []map[string]interface{} {
	m := map[string]interface{}{}

	if o.AvailabilityZones != nil {
		m["availability_zones"] = schema.NewSet(schema.HashString, flattenStringList(o.AvailabilityZones))
	}
	if o.SecurityGroupIds != nil {
		m["security_group_ids"] = schema.NewSet(schema.HashString, flattenStringList(o.SecurityGroupIds))
	}
	if o.SubnetIds != nil {
		m["subnet_ids"] = schema.NewSet(schema.HashString, flattenStringList(o.SubnetIds))
	}
	if o.VPCId != nil {
		m["vpc_id"] = *o.VPCId
	}

	return []map[string]interface{}{m}
}

func expandESVPCOptions(m map[string]interface{}) *elasticsearch.VPCOptions {
	options := elasticsearch.VPCOptions{}

	if v, ok := m["security_group_ids"]; ok {
		options.SecurityGroupIds = expandStringList(v.(*schema.Set).List())
	}
	if v, ok := m["subnet_ids"]; ok {
		options.SubnetIds = expandStringList(v.(*schema.Set).List())
	}

	return &options
}

func expandConfigRecordingGroup(configured []interface{}) *configservice.RecordingGroup {
	recordingGroup := configservice.RecordingGroup{}
	group := configured[0].(map[string]interface{})

	if v, ok := group["all_supported"]; ok {
		recordingGroup.AllSupported = aws.Bool(v.(bool))
	}

	if v, ok := group["include_global_resource_types"]; ok {
		recordingGroup.IncludeGlobalResourceTypes = aws.Bool(v.(bool))
	}

	if v, ok := group["resource_types"]; ok {
		recordingGroup.ResourceTypes = expandStringList(v.(*schema.Set).List())
	}
	return &recordingGroup
}

func flattenConfigRecordingGroup(g *configservice.RecordingGroup) []map[string]interface{} {
	m := make(map[string]interface{}, 1)

	if g.AllSupported != nil {
		m["all_supported"] = *g.AllSupported
	}

	if g.IncludeGlobalResourceTypes != nil {
		m["include_global_resource_types"] = *g.IncludeGlobalResourceTypes
	}

	if g.ResourceTypes != nil && len(g.ResourceTypes) > 0 {
		m["resource_types"] = schema.NewSet(schema.HashString, flattenStringList(g.ResourceTypes))
	}

	return []map[string]interface{}{m}
}

func flattenConfigSnapshotDeliveryProperties(p *configservice.ConfigSnapshotDeliveryProperties) []map[string]interface{} {
	m := make(map[string]interface{}, 0)

	if p.DeliveryFrequency != nil {
		m["delivery_frequency"] = *p.DeliveryFrequency
	}

	return []map[string]interface{}{m}
}

func pointersMapToStringList(pointers map[string]*string) map[string]interface{} {
	list := make(map[string]interface{}, len(pointers))
	for i, v := range pointers {
		list[i] = *v
	}
	return list
}

func stringMapToPointers(m map[string]interface{}) map[string]*string {
	list := make(map[string]*string, len(m))
	for i, v := range m {
		list[i] = aws.String(v.(string))
	}
	return list
}

func flattenDSVpcSettings(
	s *directoryservice.DirectoryVpcSettingsDescription) []map[string]interface{} {
	settings := make(map[string]interface{}, 0)

	if s == nil {
		return nil
	}

	settings["subnet_ids"] = schema.NewSet(schema.HashString, flattenStringList(s.SubnetIds))
	settings["vpc_id"] = *s.VpcId

	return []map[string]interface{}{settings}
}

func flattenLambdaEnvironment(lambdaEnv *lambda.EnvironmentResponse) []interface{} {
	envs := make(map[string]interface{})
	en := make(map[string]string)

	if lambdaEnv == nil {
		return nil
	}

	for k, v := range lambdaEnv.Variables {
		en[k] = *v
	}
	if len(en) > 0 {
		envs["variables"] = en
	}

	return []interface{}{envs}
}

func flattenLambdaVpcConfigResponse(s *lambda.VpcConfigResponse) []map[string]interface{} {
	settings := make(map[string]interface{}, 0)

	if s == nil {
		return nil
	}

	var emptyVpc bool
	if s.VpcId == nil || *s.VpcId == "" {
		emptyVpc = true
	}
	if len(s.SubnetIds) == 0 && len(s.SecurityGroupIds) == 0 && emptyVpc {
		return nil
	}

	settings["subnet_ids"] = schema.NewSet(schema.HashString, flattenStringList(s.SubnetIds))
	settings["security_group_ids"] = schema.NewSet(schema.HashString, flattenStringList(s.SecurityGroupIds))
	if s.VpcId != nil {
		settings["vpc_id"] = *s.VpcId
	}

	return []map[string]interface{}{settings}
}

func flattenDSConnectSettings(
	customerDnsIps []*string,
	s *directoryservice.DirectoryConnectSettingsDescription) []map[string]interface{} {
	if s == nil {
		return nil
	}

	settings := make(map[string]interface{}, 0)

	settings["customer_dns_ips"] = schema.NewSet(schema.HashString, flattenStringList(customerDnsIps))
	settings["connect_ips"] = schema.NewSet(schema.HashString, flattenStringList(s.ConnectIps))
	settings["customer_username"] = *s.CustomerUserName
	settings["subnet_ids"] = schema.NewSet(schema.HashString, flattenStringList(s.SubnetIds))
	settings["vpc_id"] = *s.VpcId

	return []map[string]interface{}{settings}
}

func expandCloudFormationParameters(params map[string]interface{}) []*cloudformation.Parameter {
	var cfParams []*cloudformation.Parameter
	for k, v := range params {
		cfParams = append(cfParams, &cloudformation.Parameter{
			ParameterKey:   aws.String(k),
			ParameterValue: aws.String(v.(string)),
		})
	}

	return cfParams
}

// flattenCloudFormationParameters is flattening list of
// *cloudformation.Parameters and only returning existing
// parameters to avoid clash with default values
func flattenCloudFormationParameters(cfParams []*cloudformation.Parameter,
	originalParams map[string]interface{}) map[string]interface{} {
	params := make(map[string]interface{}, len(cfParams))
	for _, p := range cfParams {
		_, isConfigured := originalParams[*p.ParameterKey]
		if isConfigured {
			params[*p.ParameterKey] = *p.ParameterValue
		}
	}
	return params
}

func flattenAllCloudFormationParameters(cfParams []*cloudformation.Parameter) map[string]interface{} {
	params := make(map[string]interface{}, len(cfParams))
	for _, p := range cfParams {
		params[*p.ParameterKey] = *p.ParameterValue
	}
	return params
}

func expandCloudFormationTags(tags map[string]interface{}) []*cloudformation.Tag {
	var cfTags []*cloudformation.Tag
	for k, v := range tags {
		cfTags = append(cfTags, &cloudformation.Tag{
			Key:   aws.String(k),
			Value: aws.String(v.(string)),
		})
	}
	return cfTags
}

func flattenCloudFormationTags(cfTags []*cloudformation.Tag) map[string]string {
	tags := make(map[string]string, len(cfTags))
	for _, t := range cfTags {
		tags[*t.Key] = *t.Value
	}
	return tags
}

func flattenCloudFormationOutputs(cfOutputs []*cloudformation.Output) map[string]string {
	outputs := make(map[string]string, len(cfOutputs))
	for _, o := range cfOutputs {
		outputs[*o.OutputKey] = *o.OutputValue
	}
	return outputs
}

func flattenAsgSuspendedProcesses(list []*autoscaling.SuspendedProcess) []string {
	strs := make([]string, 0, len(list))
	for _, r := range list {
		if r.ProcessName != nil {
			strs = append(strs, *r.ProcessName)
		}
	}
	return strs
}

func flattenAsgEnabledMetrics(list []*autoscaling.EnabledMetric) []string {
	strs := make([]string, 0, len(list))
	for _, r := range list {
		if r.Metric != nil {
			strs = append(strs, *r.Metric)
		}
	}
	return strs
}

func flattenKinesisShardLevelMetrics(list []*kinesis.EnhancedMetrics) []string {
	if len(list) == 0 {
		return []string{}
	}
	strs := make([]string, 0, len(list[0].ShardLevelMetrics))
	for _, s := range list[0].ShardLevelMetrics {
		strs = append(strs, *s)
	}
	return strs
}

func flattenApiGatewayStageKeys(keys []*string) []map[string]interface{} {
	stageKeys := make([]map[string]interface{}, 0, len(keys))
	for _, o := range keys {
		key := make(map[string]interface{})
		parts := strings.Split(*o, "/")
		key["stage_name"] = parts[1]
		key["rest_api_id"] = parts[0]

		stageKeys = append(stageKeys, key)
	}
	return stageKeys
}

func expandApiGatewayStageKeys(d *schema.ResourceData) []*apigateway.StageKey {
	var stageKeys []*apigateway.StageKey

	if stageKeyData, ok := d.GetOk("stage_key"); ok {
		params := stageKeyData.(*schema.Set).List()
		for k := range params {
			data := params[k].(map[string]interface{})
			stageKeys = append(stageKeys, &apigateway.StageKey{
				RestApiId: aws.String(data["rest_api_id"].(string)),
				StageName: aws.String(data["stage_name"].(string)),
			})
		}
	}

	return stageKeys
}

func expandApiGatewayRequestResponseModelOperations(d *schema.ResourceData, key string, prefix string) []*apigateway.PatchOperation {
	operations := make([]*apigateway.PatchOperation, 0)

	oldModels, newModels := d.GetChange(key)
	oldModelMap := oldModels.(map[string]interface{})
	newModelMap := newModels.(map[string]interface{})

	for k, _ := range oldModelMap {
		operation := apigateway.PatchOperation{
			Op:   aws.String("remove"),
			Path: aws.String(fmt.Sprintf("/%s/%s", prefix, strings.Replace(k, "/", "~1", -1))),
		}

		for nK, nV := range newModelMap {
			if nK == k {
				operation.Op = aws.String("replace")
				operation.Value = aws.String(nV.(string))
			}
		}

		operations = append(operations, &operation)
	}

	for nK, nV := range newModelMap {
		exists := false
		for k, _ := range oldModelMap {
			if k == nK {
				exists = true
			}
		}
		if !exists {
			operation := apigateway.PatchOperation{
				Op:    aws.String("add"),
				Path:  aws.String(fmt.Sprintf("/%s/%s", prefix, strings.Replace(nK, "/", "~1", -1))),
				Value: aws.String(nV.(string)),
			}
			operations = append(operations, &operation)
		}
	}

	return operations
}

func deprecatedExpandApiGatewayMethodParametersJSONOperations(d *schema.ResourceData, key string, prefix string) ([]*apigateway.PatchOperation, error) {
	operations := make([]*apigateway.PatchOperation, 0)
	oldParameters, newParameters := d.GetChange(key)
	oldParametersMap := make(map[string]interface{})
	newParametersMap := make(map[string]interface{})

	if err := json.Unmarshal([]byte(oldParameters.(string)), &oldParametersMap); err != nil {
		err := fmt.Errorf("Error unmarshaling old %s: %s", key, err)
		return operations, err
	}

	if err := json.Unmarshal([]byte(newParameters.(string)), &newParametersMap); err != nil {
		err := fmt.Errorf("Error unmarshaling new %s: %s", key, err)
		return operations, err
	}

	for k, _ := range oldParametersMap {
		operation := apigateway.PatchOperation{
			Op:   aws.String("remove"),
			Path: aws.String(fmt.Sprintf("/%s/%s", prefix, k)),
		}

		for nK, nV := range newParametersMap {
			if nK == k {
				operation.Op = aws.String("replace")
				operation.Value = aws.String(strconv.FormatBool(nV.(bool)))
			}
		}

		operations = append(operations, &operation)
	}

	for nK, nV := range newParametersMap {
		exists := false
		for k, _ := range oldParametersMap {
			if k == nK {
				exists = true
			}
		}
		if !exists {
			operation := apigateway.PatchOperation{
				Op:    aws.String("add"),
				Path:  aws.String(fmt.Sprintf("/%s/%s", prefix, nK)),
				Value: aws.String(strconv.FormatBool(nV.(bool))),
			}
			operations = append(operations, &operation)
		}
	}

	return operations, nil
}

func expandApiGatewayMethodParametersOperations(d *schema.ResourceData, key string, prefix string) ([]*apigateway.PatchOperation, error) {
	operations := make([]*apigateway.PatchOperation, 0)

	oldParameters, newParameters := d.GetChange(key)
	oldParametersMap := oldParameters.(map[string]interface{})
	newParametersMap := newParameters.(map[string]interface{})

	for k, _ := range oldParametersMap {
		operation := apigateway.PatchOperation{
			Op:   aws.String("remove"),
			Path: aws.String(fmt.Sprintf("/%s/%s", prefix, k)),
		}

		for nK, nV := range newParametersMap {
			b, ok := nV.(bool)
			if !ok {
				value, _ := strconv.ParseBool(nV.(string))
				b = value
			}
			if nK == k {
				operation.Op = aws.String("replace")
				operation.Value = aws.String(strconv.FormatBool(b))
			}
		}

		operations = append(operations, &operation)
	}

	for nK, nV := range newParametersMap {
		exists := false
		for k, _ := range oldParametersMap {
			if k == nK {
				exists = true
			}
		}
		if !exists {
			b, ok := nV.(bool)
			if !ok {
				value, _ := strconv.ParseBool(nV.(string))
				b = value
			}
			operation := apigateway.PatchOperation{
				Op:    aws.String("add"),
				Path:  aws.String(fmt.Sprintf("/%s/%s", prefix, nK)),
				Value: aws.String(strconv.FormatBool(b)),
			}
			operations = append(operations, &operation)
		}
	}

	return operations, nil
}

func expandApiGatewayStageKeyOperations(d *schema.ResourceData) []*apigateway.PatchOperation {
	operations := make([]*apigateway.PatchOperation, 0)

	prev, curr := d.GetChange("stage_key")
	prevList := prev.(*schema.Set).List()
	currList := curr.(*schema.Set).List()

	for i := range prevList {
		p := prevList[i].(map[string]interface{})
		exists := false

		for j := range currList {
			c := currList[j].(map[string]interface{})
			if c["rest_api_id"].(string) == p["rest_api_id"].(string) && c["stage_name"].(string) == p["stage_name"].(string) {
				exists = true
			}
		}

		if !exists {
			operations = append(operations, &apigateway.PatchOperation{
				Op:    aws.String("remove"),
				Path:  aws.String("/stages"),
				Value: aws.String(fmt.Sprintf("%s/%s", p["rest_api_id"].(string), p["stage_name"].(string))),
			})
		}
	}

	for i := range currList {
		c := currList[i].(map[string]interface{})
		exists := false

		for j := range prevList {
			p := prevList[j].(map[string]interface{})
			if c["rest_api_id"].(string) == p["rest_api_id"].(string) && c["stage_name"].(string) == p["stage_name"].(string) {
				exists = true
			}
		}

		if !exists {
			operations = append(operations, &apigateway.PatchOperation{
				Op:    aws.String("add"),
				Path:  aws.String("/stages"),
				Value: aws.String(fmt.Sprintf("%s/%s", c["rest_api_id"].(string), c["stage_name"].(string))),
			})
		}
	}

	return operations
}

func expandCloudWachLogMetricTransformations(m map[string]interface{}) ([]*cloudwatchlogs.MetricTransformation, error) {
	transformation := cloudwatchlogs.MetricTransformation{
		MetricName:      aws.String(m["name"].(string)),
		MetricNamespace: aws.String(m["namespace"].(string)),
		MetricValue:     aws.String(m["value"].(string)),
	}

	if m["default_value"] != "" {
		transformation.DefaultValue = aws.Float64(m["default_value"].(float64))
	}

	return []*cloudwatchlogs.MetricTransformation{&transformation}, nil
}

func flattenCloudWachLogMetricTransformations(ts []*cloudwatchlogs.MetricTransformation) []interface{} {
	mts := make([]interface{}, 0)
	m := make(map[string]interface{}, 0)

	m["name"] = *ts[0].MetricName
	m["namespace"] = *ts[0].MetricNamespace
	m["value"] = *ts[0].MetricValue

	if ts[0].DefaultValue != nil {
		m["default_value"] = *ts[0].DefaultValue
	}

	mts = append(mts, m)

	return mts
}

func flattenBeanstalkAsg(list []*elasticbeanstalk.AutoScalingGroup) []string {
	strs := make([]string, 0, len(list))
	for _, r := range list {
		if r.Name != nil {
			strs = append(strs, *r.Name)
		}
	}
	return strs
}

func flattenBeanstalkInstances(list []*elasticbeanstalk.Instance) []string {
	strs := make([]string, 0, len(list))
	for _, r := range list {
		if r.Id != nil {
			strs = append(strs, *r.Id)
		}
	}
	return strs
}

func flattenBeanstalkLc(list []*elasticbeanstalk.LaunchConfiguration) []string {
	strs := make([]string, 0, len(list))
	for _, r := range list {
		if r.Name != nil {
			strs = append(strs, *r.Name)
		}
	}
	return strs
}

func flattenBeanstalkElb(list []*elasticbeanstalk.LoadBalancer) []string {
	strs := make([]string, 0, len(list))
	for _, r := range list {
		if r.Name != nil {
			strs = append(strs, *r.Name)
		}
	}
	return strs
}

func flattenBeanstalkSqs(list []*elasticbeanstalk.Queue) []string {
	strs := make([]string, 0, len(list))
	for _, r := range list {
		if r.URL != nil {
			strs = append(strs, *r.URL)
		}
	}
	return strs
}

func flattenBeanstalkTrigger(list []*elasticbeanstalk.Trigger) []string {
	strs := make([]string, 0, len(list))
	for _, r := range list {
		if r.Name != nil {
			strs = append(strs, *r.Name)
		}
	}
	return strs
}

// There are several parts of the AWS API that will sort lists of strings,
// causing diffs inbetween resources that use lists. This avoids a bit of
// code duplication for pre-sorts that can be used for things like hash
// functions, etc.
func sortInterfaceSlice(in []interface{}) []interface{} {
	a := []string{}
	b := []interface{}{}
	for _, v := range in {
		a = append(a, v.(string))
	}

	sort.Strings(a)

	for _, v := range a {
		b = append(b, v)
	}

	return b
}

// This function sorts List A to look like a list found in the tf file.
func sortListBasedonTFFile(in []string, d *schema.ResourceData, listName string) ([]string, error) {
	if attributeCount, ok := d.Get(listName + ".#").(int); ok {
		for i := 0; i < attributeCount; i++ {
			currAttributeId := d.Get(listName + "." + strconv.Itoa(i))
			for j := 0; j < len(in); j++ {
				if currAttributeId == in[j] {
					in[i], in[j] = in[j], in[i]
				}
			}
		}
		return in, nil
	}
	return in, fmt.Errorf("Could not find list: %s", listName)
}

func flattenApiGatewayThrottleSettings(settings *apigateway.ThrottleSettings) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, 1)

	if settings != nil {
		r := make(map[string]interface{})
		if settings.BurstLimit != nil {
			r["burst_limit"] = *settings.BurstLimit
		}

		if settings.RateLimit != nil {
			r["rate_limit"] = *settings.RateLimit
		}

		result = append(result, r)
	}

	return result
}

// TODO: refactor some of these helper functions and types in the terraform/helper packages

// getStringPtr returns a *string version of the value taken from m, where m
// can be a map[string]interface{} or a *schema.ResourceData. If the key isn't
// present or is empty, getNilString returns nil.
func getStringPtr(m interface{}, key string) *string {
	switch m := m.(type) {
	case map[string]interface{}:
		v := m[key]

		if v == nil {
			return nil
		}

		s := v.(string)
		if s == "" {
			return nil
		}

		return &s

	case *schema.ResourceData:
		if v, ok := m.GetOk(key); ok {
			if v == nil || v.(string) == "" {
				return nil
			}
			s := v.(string)
			return &s
		}

	default:
		panic("unknown type in getStringPtr")
	}

	return nil
}

// getStringPtrList returns a []*string version of the map value. If the key
// isn't present, getNilStringList returns nil.
func getStringPtrList(m map[string]interface{}, key string) []*string {
	if v, ok := m[key]; ok {
		var stringList []*string
		for _, i := range v.([]interface{}) {
			s := i.(string)
			stringList = append(stringList, &s)
		}

		return stringList
	}

	return nil
}

// a convenience wrapper type for the schema.Set map[string]interface{}
// Set operations only alter the underlying map if the value is not nil
type setMap map[string]interface{}

// SetString sets m[key] = *value only if `value != nil`
func (s setMap) SetString(key string, value *string) {
	if value == nil {
		return
	}

	s[key] = *value
}

// SetStringMap sets key to value as a map[string]interface{}, stripping any nil
// values. The value parameter can be a map[string]interface{}, a
// map[string]*string, or a map[string]string.
func (s setMap) SetStringMap(key string, value interface{}) {
	// because these methods are meant to be chained without intermediate
	// checks for nil, we are likely to get interfaces with dynamic types but
	// a nil value.
	if reflect.ValueOf(value).IsNil() {
		return
	}

	m := make(map[string]interface{})

	switch value := value.(type) {
	case map[string]string:
		for k, v := range value {
			m[k] = v
		}
	case map[string]*string:
		for k, v := range value {
			if v == nil {
				continue
			}
			m[k] = *v
		}
	case map[string]interface{}:
		for k, v := range value {
			if v == nil {
				continue
			}

			switch v := v.(type) {
			case string:
				m[k] = v
			case *string:
				if v != nil {
					m[k] = *v
				}
			default:
				panic(fmt.Sprintf("unknown type for SetString: %T", v))
			}
		}
	}

	// catch the case where the interface wasn't nil, but we had no non-nil values
	if len(m) > 0 {
		s[key] = m
	}
}

// Set assigns value to s[key] if value isn't nil
func (s setMap) Set(key string, value interface{}) {
	if reflect.ValueOf(value).IsNil() {
		return
	}

	s[key] = value
}

// Map returns the raw map type for a shorter type conversion
func (s setMap) Map() map[string]interface{} {
	return map[string]interface{}(s)
}

// MapList returns the map[string]interface{} as a single element in a slice to
// match the schema.Set data type used for structs.
func (s setMap) MapList() []map[string]interface{} {
	return []map[string]interface{}{s.Map()}
}

// Takes the result of flatmap.Expand for an array of policy attributes and
// returns ELB API compatible objects
func expandPolicyAttributes(configured []interface{}) ([]*elb.PolicyAttribute, error) {
	attributes := make([]*elb.PolicyAttribute, 0, len(configured))

	// Loop over our configured attributes and create
	// an array of aws-sdk-go compatible objects
	for _, lRaw := range configured {
		data := lRaw.(map[string]interface{})

		a := &elb.PolicyAttribute{
			AttributeName:  aws.String(data["name"].(string)),
			AttributeValue: aws.String(data["value"].(string)),
		}

		attributes = append(attributes, a)

	}

	return attributes, nil
}

// Flattens an array of PolicyAttributes into a []interface{}
func flattenPolicyAttributes(list []*elb.PolicyAttributeDescription) []interface{} {
	attributes := []interface{}{}
	for _, attrdef := range list {
		attribute := map[string]string{
			"name":  *attrdef.AttributeName,
			"value": *attrdef.AttributeValue,
		}

		attributes = append(attributes, attribute)

	}

	return attributes
}

func flattenConfigRuleSource(source *configservice.Source) []interface{} {
	var result []interface{}
	m := make(map[string]interface{})
	m["owner"] = *source.Owner
	m["source_identifier"] = *source.SourceIdentifier
	if len(source.SourceDetails) > 0 {
		m["source_detail"] = schema.NewSet(configRuleSourceDetailsHash, flattenConfigRuleSourceDetails(source.SourceDetails))
	}
	result = append(result, m)
	return result
}

func flattenConfigRuleSourceDetails(details []*configservice.SourceDetail) []interface{} {
	var items []interface{}
	for _, d := range details {
		m := make(map[string]interface{})
		if d.MessageType != nil {
			m["message_type"] = *d.MessageType
		}
		if d.EventSource != nil {
			m["event_source"] = *d.EventSource
		}
		if d.MaximumExecutionFrequency != nil {
			m["maximum_execution_frequency"] = *d.MaximumExecutionFrequency
		}

		items = append(items, m)
	}

	return items
}

func expandConfigRuleSource(configured []interface{}) *configservice.Source {
	cfg := configured[0].(map[string]interface{})
	source := configservice.Source{
		Owner:            aws.String(cfg["owner"].(string)),
		SourceIdentifier: aws.String(cfg["source_identifier"].(string)),
	}
	if details, ok := cfg["source_detail"]; ok {
		source.SourceDetails = expandConfigRuleSourceDetails(details.(*schema.Set))
	}
	return &source
}

func expandConfigRuleSourceDetails(configured *schema.Set) []*configservice.SourceDetail {
	var results []*configservice.SourceDetail

	for _, item := range configured.List() {
		detail := item.(map[string]interface{})
		src := configservice.SourceDetail{}

		if msgType, ok := detail["message_type"].(string); ok && msgType != "" {
			src.MessageType = aws.String(msgType)
		}
		if eventSource, ok := detail["event_source"].(string); ok && eventSource != "" {
			src.EventSource = aws.String(eventSource)
		}
		if maxExecFreq, ok := detail["maximum_execution_frequency"].(string); ok && maxExecFreq != "" {
			src.MaximumExecutionFrequency = aws.String(maxExecFreq)
		}

		results = append(results, &src)
	}

	return results
}

func flattenConfigRuleScope(scope *configservice.Scope) []interface{} {
	var items []interface{}

	m := make(map[string]interface{})
	if scope.ComplianceResourceId != nil {
		m["compliance_resource_id"] = *scope.ComplianceResourceId
	}
	if scope.ComplianceResourceTypes != nil {
		m["compliance_resource_types"] = schema.NewSet(schema.HashString, flattenStringList(scope.ComplianceResourceTypes))
	}
	if scope.TagKey != nil {
		m["tag_key"] = *scope.TagKey
	}
	if scope.TagValue != nil {
		m["tag_value"] = *scope.TagValue
	}

	items = append(items, m)
	return items
}

func expandConfigRuleScope(configured map[string]interface{}) *configservice.Scope {
	scope := &configservice.Scope{}

	if v, ok := configured["compliance_resource_id"].(string); ok && v != "" {
		scope.ComplianceResourceId = aws.String(v)
	}
	if v, ok := configured["compliance_resource_types"]; ok {
		l := v.(*schema.Set)
		if l.Len() > 0 {
			scope.ComplianceResourceTypes = expandStringList(l.List())
		}
	}
	if v, ok := configured["tag_key"].(string); ok && v != "" {
		scope.TagKey = aws.String(v)
	}
	if v, ok := configured["tag_value"].(string); ok && v != "" {
		scope.TagValue = aws.String(v)
	}

	return scope
}

// Takes a value containing YAML string and passes it through
// the YAML parser. Returns either a parsing
// error or original YAML string.
func checkYamlString(yamlString interface{}) (string, error) {
	var y interface{}

	if yamlString == nil || yamlString.(string) == "" {
		return "", nil
	}

	s := yamlString.(string)

	err := yaml.Unmarshal([]byte(s), &y)
	if err != nil {
		return s, err
	}

	return s, nil
}

func normalizeCloudFormationTemplate(templateString interface{}) (string, error) {
	if looksLikeJsonString(templateString) {
		return structure.NormalizeJsonString(templateString.(string))
	}

	return checkYamlString(templateString)
}

func flattenInspectorTags(cfTags []*cloudformation.Tag) map[string]string {
	tags := make(map[string]string, len(cfTags))
	for _, t := range cfTags {
		tags[*t.Key] = *t.Value
	}
	return tags
}

func flattenApiGatewayUsageApiStages(s []*apigateway.ApiStage) []map[string]interface{} {
	stages := make([]map[string]interface{}, 0)

	for _, bd := range s {
		if bd.ApiId != nil && bd.Stage != nil {
			stage := make(map[string]interface{})
			stage["api_id"] = *bd.ApiId
			stage["stage"] = *bd.Stage

			stages = append(stages, stage)
		}
	}

	if len(stages) > 0 {
		return stages
	}

	return nil
}

func flattenApiGatewayUsagePlanThrottling(s *apigateway.ThrottleSettings) []map[string]interface{} {
	settings := make(map[string]interface{}, 0)

	if s == nil {
		return nil
	}

	if s.BurstLimit != nil {
		settings["burst_limit"] = *s.BurstLimit
	}

	if s.RateLimit != nil {
		settings["rate_limit"] = *s.RateLimit
	}

	return []map[string]interface{}{settings}
}

func flattenApiGatewayUsagePlanQuota(s *apigateway.QuotaSettings) []map[string]interface{} {
	settings := make(map[string]interface{}, 0)

	if s == nil {
		return nil
	}

	if s.Limit != nil {
		settings["limit"] = *s.Limit
	}

	if s.Offset != nil {
		settings["offset"] = *s.Offset
	}

	if s.Period != nil {
		settings["period"] = *s.Period
	}

	return []map[string]interface{}{settings}
}

func buildApiGatewayInvokeURL(restApiId, region, stageName string) string {
	return fmt.Sprintf("https://%s.execute-api.%s.amazonaws.com/%s",
		restApiId, region, stageName)
}

func buildApiGatewayExecutionARN(restApiId, region, accountId string) (string, error) {
	if accountId == "" {
		return "", fmt.Errorf("Unable to build execution ARN for %s as account ID is missing",
			restApiId)
	}
	return fmt.Sprintf("arn:aws:execute-api:%s:%s:%s",
		region, accountId, restApiId), nil
}

func expandCognitoSupportedLoginProviders(config map[string]interface{}) map[string]*string {
	m := map[string]*string{}
	for k, v := range config {
		s := v.(string)
		m[k] = &s
	}
	return m
}

func flattenCognitoSupportedLoginProviders(config map[string]*string) map[string]string {
	m := map[string]string{}
	for k, v := range config {
		m[k] = *v
	}
	return m
}

func expandCognitoIdentityProviders(s *schema.Set) []*cognitoidentity.Provider {
	ips := make([]*cognitoidentity.Provider, 0)

	for _, v := range s.List() {
		s := v.(map[string]interface{})

		ip := &cognitoidentity.Provider{}

		if sv, ok := s["client_id"].(string); ok {
			ip.ClientId = aws.String(sv)
		}

		if sv, ok := s["provider_name"].(string); ok {
			ip.ProviderName = aws.String(sv)
		}

		if sv, ok := s["server_side_token_check"].(bool); ok {
			ip.ServerSideTokenCheck = aws.Bool(sv)
		}

		ips = append(ips, ip)
	}

	return ips
}

func flattenCognitoIdentityProviders(ips []*cognitoidentity.Provider) []map[string]interface{} {
	values := make([]map[string]interface{}, 0)

	for _, v := range ips {
		ip := make(map[string]interface{})

		if v == nil {
			return nil
		}

		if v.ClientId != nil {
			ip["client_id"] = *v.ClientId
		}

		if v.ProviderName != nil {
			ip["provider_name"] = *v.ProviderName
		}

		if v.ServerSideTokenCheck != nil {
			ip["server_side_token_check"] = *v.ServerSideTokenCheck
		}

		values = append(values, ip)
	}

	return values
}

func flattenCognitoUserPoolEmailConfiguration(s *cognitoidentityprovider.EmailConfigurationType) []map[string]interface{} {
	m := make(map[string]interface{}, 0)

	if s == nil {
		return nil
	}

	if s.ReplyToEmailAddress != nil {
		m["reply_to_email_address"] = *s.ReplyToEmailAddress
	}

	if s.SourceArn != nil {
		m["source_arn"] = *s.SourceArn
	}

	if len(m) > 0 {
		return []map[string]interface{}{m}
	}

	return []map[string]interface{}{}
}

func expandCognitoUserPoolAdminCreateUserConfig(config map[string]interface{}) *cognitoidentityprovider.AdminCreateUserConfigType {
	configs := &cognitoidentityprovider.AdminCreateUserConfigType{}

	if v, ok := config["allow_admin_create_user_only"]; ok {
		configs.AllowAdminCreateUserOnly = aws.Bool(v.(bool))
	}

	if v, ok := config["invite_message_template"]; ok {
		data := v.([]interface{})

		if len(data) > 0 {
			m, ok := data[0].(map[string]interface{})

			if ok {
				imt := &cognitoidentityprovider.MessageTemplateType{}

				if v, ok := m["email_message"]; ok {
					imt.EmailMessage = aws.String(v.(string))
				}

				if v, ok := m["email_subject"]; ok {
					imt.EmailSubject = aws.String(v.(string))
				}

				if v, ok := m["sms_message"]; ok {
					imt.SMSMessage = aws.String(v.(string))
				}

				configs.InviteMessageTemplate = imt
			}
		}
	}

	configs.UnusedAccountValidityDays = aws.Int64(int64(config["unused_account_validity_days"].(int)))

	return configs
}

func flattenCognitoUserPoolAdminCreateUserConfig(s *cognitoidentityprovider.AdminCreateUserConfigType) []map[string]interface{} {
	config := map[string]interface{}{}

	if s == nil {
		return nil
	}

	if s.AllowAdminCreateUserOnly != nil {
		config["allow_admin_create_user_only"] = *s.AllowAdminCreateUserOnly
	}

	if s.InviteMessageTemplate != nil {
		subconfig := map[string]interface{}{}

		if s.InviteMessageTemplate.EmailMessage != nil {
			subconfig["email_message"] = *s.InviteMessageTemplate.EmailMessage
		}

		if s.InviteMessageTemplate.EmailSubject != nil {
			subconfig["email_subject"] = *s.InviteMessageTemplate.EmailSubject
		}

		if s.InviteMessageTemplate.SMSMessage != nil {
			subconfig["sms_message"] = *s.InviteMessageTemplate.SMSMessage
		}

		if len(subconfig) > 0 {
			config["invite_message_template"] = []map[string]interface{}{subconfig}
		}
	}

	config["unused_account_validity_days"] = *s.UnusedAccountValidityDays

	return []map[string]interface{}{config}
}

func expandCognitoUserPoolDeviceConfiguration(config map[string]interface{}) *cognitoidentityprovider.DeviceConfigurationType {
	configs := &cognitoidentityprovider.DeviceConfigurationType{}

	if v, ok := config["challenge_required_on_new_device"]; ok {
		configs.ChallengeRequiredOnNewDevice = aws.Bool(v.(bool))
	}

	if v, ok := config["device_only_remembered_on_user_prompt"]; ok {
		configs.DeviceOnlyRememberedOnUserPrompt = aws.Bool(v.(bool))
	}

	return configs
}

func flattenCognitoUserPoolDeviceConfiguration(s *cognitoidentityprovider.DeviceConfigurationType) []map[string]interface{} {
	config := map[string]interface{}{}

	if s == nil {
		return nil
	}

	if s.ChallengeRequiredOnNewDevice != nil {
		config["challenge_required_on_new_device"] = *s.ChallengeRequiredOnNewDevice
	}

	if s.DeviceOnlyRememberedOnUserPrompt != nil {
		config["device_only_remembered_on_user_prompt"] = *s.DeviceOnlyRememberedOnUserPrompt
	}

	return []map[string]interface{}{config}
}

func expandCognitoUserPoolLambdaConfig(config map[string]interface{}) *cognitoidentityprovider.LambdaConfigType {
	configs := &cognitoidentityprovider.LambdaConfigType{}

	if v, ok := config["create_auth_challenge"]; ok && v.(string) != "" {
		configs.CreateAuthChallenge = aws.String(v.(string))
	}

	if v, ok := config["custom_message"]; ok && v.(string) != "" {
		configs.CustomMessage = aws.String(v.(string))
	}

	if v, ok := config["define_auth_challenge"]; ok && v.(string) != "" {
		configs.DefineAuthChallenge = aws.String(v.(string))
	}

	if v, ok := config["post_authentication"]; ok && v.(string) != "" {
		configs.PostAuthentication = aws.String(v.(string))
	}

	if v, ok := config["post_confirmation"]; ok && v.(string) != "" {
		configs.PostConfirmation = aws.String(v.(string))
	}

	if v, ok := config["pre_authentication"]; ok && v.(string) != "" {
		configs.PreAuthentication = aws.String(v.(string))
	}

	if v, ok := config["pre_sign_up"]; ok && v.(string) != "" {
		configs.PreSignUp = aws.String(v.(string))
	}

	if v, ok := config["pre_token_generation"]; ok && v.(string) != "" {
		configs.PreTokenGeneration = aws.String(v.(string))
	}

	if v, ok := config["verify_auth_challenge_response"]; ok && v.(string) != "" {
		configs.VerifyAuthChallengeResponse = aws.String(v.(string))
	}

	return configs
}

func flattenCognitoUserPoolLambdaConfig(s *cognitoidentityprovider.LambdaConfigType) []map[string]interface{} {
	m := map[string]interface{}{}

	if s == nil {
		return nil
	}

	if s.CreateAuthChallenge != nil {
		m["create_auth_challenge"] = *s.CreateAuthChallenge
	}

	if s.CustomMessage != nil {
		m["custom_message"] = *s.CustomMessage
	}

	if s.DefineAuthChallenge != nil {
		m["define_auth_challenge"] = *s.DefineAuthChallenge
	}

	if s.PostAuthentication != nil {
		m["post_authentication"] = *s.PostAuthentication
	}

	if s.PostConfirmation != nil {
		m["post_confirmation"] = *s.PostConfirmation
	}

	if s.PreAuthentication != nil {
		m["pre_authentication"] = *s.PreAuthentication
	}

	if s.PreSignUp != nil {
		m["pre_sign_up"] = *s.PreSignUp
	}

	if s.PreTokenGeneration != nil {
		m["pre_token_generation"] = *s.PreTokenGeneration
	}

	if s.VerifyAuthChallengeResponse != nil {
		m["verify_auth_challenge_response"] = *s.VerifyAuthChallengeResponse
	}

	if len(m) > 0 {
		return []map[string]interface{}{m}
	}

	return []map[string]interface{}{}
}

func expandCognitoUserPoolPasswordPolicy(config map[string]interface{}) *cognitoidentityprovider.PasswordPolicyType {
	configs := &cognitoidentityprovider.PasswordPolicyType{}

	if v, ok := config["minimum_length"]; ok {
		configs.MinimumLength = aws.Int64(int64(v.(int)))
	}

	if v, ok := config["require_lowercase"]; ok {
		configs.RequireLowercase = aws.Bool(v.(bool))
	}

	if v, ok := config["require_numbers"]; ok {
		configs.RequireNumbers = aws.Bool(v.(bool))
	}

	if v, ok := config["require_symbols"]; ok {
		configs.RequireSymbols = aws.Bool(v.(bool))
	}

	if v, ok := config["require_uppercase"]; ok {
		configs.RequireUppercase = aws.Bool(v.(bool))
	}

	return configs
}

func flattenCognitoUserPoolPasswordPolicy(s *cognitoidentityprovider.PasswordPolicyType) []map[string]interface{} {
	m := map[string]interface{}{}

	if s == nil {
		return nil
	}

	if s.MinimumLength != nil {
		m["minimum_length"] = *s.MinimumLength
	}

	if s.RequireLowercase != nil {
		m["require_lowercase"] = *s.RequireLowercase
	}

	if s.RequireNumbers != nil {
		m["require_numbers"] = *s.RequireNumbers
	}

	if s.RequireSymbols != nil {
		m["require_symbols"] = *s.RequireSymbols
	}

	if s.RequireUppercase != nil {
		m["require_uppercase"] = *s.RequireUppercase
	}

	if len(m) > 0 {
		return []map[string]interface{}{m}
	}

	return []map[string]interface{}{}
}

func expandCognitoUserPoolSchema(inputs []interface{}) []*cognitoidentityprovider.SchemaAttributeType {
	configs := make([]*cognitoidentityprovider.SchemaAttributeType, len(inputs), len(inputs))

	for i, input := range inputs {
		param := input.(map[string]interface{})
		config := &cognitoidentityprovider.SchemaAttributeType{}

		if v, ok := param["attribute_data_type"]; ok {
			config.AttributeDataType = aws.String(v.(string))
		}

		if v, ok := param["developer_only_attribute"]; ok {
			config.DeveloperOnlyAttribute = aws.Bool(v.(bool))
		}

		if v, ok := param["mutable"]; ok {
			config.Mutable = aws.Bool(v.(bool))
		}

		if v, ok := param["name"]; ok {
			config.Name = aws.String(v.(string))
		}

		if v, ok := param["required"]; ok {
			config.Required = aws.Bool(v.(bool))
		}

		if v, ok := param["number_attribute_constraints"]; ok {
			data := v.([]interface{})

			if len(data) > 0 {
				m, ok := data[0].(map[string]interface{})
				if ok {
					numberAttributeConstraintsType := &cognitoidentityprovider.NumberAttributeConstraintsType{}

					if v, ok := m["min_value"]; ok && v.(string) != "" {
						numberAttributeConstraintsType.MinValue = aws.String(v.(string))
					}

					if v, ok := m["max_value"]; ok && v.(string) != "" {
						numberAttributeConstraintsType.MaxValue = aws.String(v.(string))
					}

					config.NumberAttributeConstraints = numberAttributeConstraintsType
				}
			}
		}

		if v, ok := param["string_attribute_constraints"]; ok {
			data := v.([]interface{})

			if len(data) > 0 {
				m, _ := data[0].(map[string]interface{})
				if ok {
					stringAttributeConstraintsType := &cognitoidentityprovider.StringAttributeConstraintsType{}

					if l, ok := m["min_length"]; ok && l.(string) != "" {
						stringAttributeConstraintsType.MinLength = aws.String(l.(string))
					}

					if l, ok := m["max_length"]; ok && l.(string) != "" {
						stringAttributeConstraintsType.MaxLength = aws.String(l.(string))
					}

					config.StringAttributeConstraints = stringAttributeConstraintsType
				}
			}
		}

		configs[i] = config
	}

	return configs
}

func flattenCognitoUserPoolSchema(inputs []*cognitoidentityprovider.SchemaAttributeType) []map[string]interface{} {
	values := make([]map[string]interface{}, len(inputs), len(inputs))

	for i, input := range inputs {
		value := make(map[string]interface{})

		if input == nil {
			return nil
		}

		if input.AttributeDataType != nil {
			value["attribute_data_type"] = *input.AttributeDataType
		}

		if input.DeveloperOnlyAttribute != nil {
			value["developer_only_attribute"] = *input.DeveloperOnlyAttribute
		}

		if input.Mutable != nil {
			value["mutable"] = *input.Mutable
		}

		if input.Name != nil {
			value["name"] = *input.Name
		}

		if input.NumberAttributeConstraints != nil {
			subvalue := make(map[string]interface{})

			if input.NumberAttributeConstraints.MinValue != nil {
				subvalue["min_value"] = input.NumberAttributeConstraints.MinValue
			}

			if input.NumberAttributeConstraints.MaxValue != nil {
				subvalue["max_value"] = input.NumberAttributeConstraints.MaxValue
			}

			value["number_attribute_constraints"] = subvalue
		}

		if input.Required != nil {
			value["required"] = *input.Required
		}

		if input.StringAttributeConstraints != nil {
			subvalue := make(map[string]interface{})

			if input.StringAttributeConstraints.MinLength != nil {
				subvalue["min_length"] = input.StringAttributeConstraints.MinLength
			}

			if input.StringAttributeConstraints.MaxLength != nil {
				subvalue["max_length"] = input.StringAttributeConstraints.MaxLength
			}

			value["string_attribute_constraints"] = subvalue
		}

		values[i] = value
	}

	return values
}

func expandCognitoUserPoolSmsConfiguration(config map[string]interface{}) *cognitoidentityprovider.SmsConfigurationType {
	smsConfigurationType := &cognitoidentityprovider.SmsConfigurationType{
		SnsCallerArn: aws.String(config["sns_caller_arn"].(string)),
	}

	if v, ok := config["external_id"]; ok && v.(string) != "" {
		smsConfigurationType.ExternalId = aws.String(v.(string))
	}

	return smsConfigurationType
}

func flattenCognitoUserPoolSmsConfiguration(s *cognitoidentityprovider.SmsConfigurationType) []map[string]interface{} {
	m := map[string]interface{}{}

	if s == nil {
		return nil
	}

	if s.ExternalId != nil {
		m["external_id"] = *s.ExternalId
	}
	m["sns_caller_arn"] = *s.SnsCallerArn

	return []map[string]interface{}{m}
}

func expandCognitoUserPoolVerificationMessageTemplate(config map[string]interface{}) *cognitoidentityprovider.VerificationMessageTemplateType {
	verificationMessageTemplateType := &cognitoidentityprovider.VerificationMessageTemplateType{}

	if v, ok := config["default_email_option"]; ok && v.(string) != "" {
		verificationMessageTemplateType.DefaultEmailOption = aws.String(v.(string))
	}

	if v, ok := config["email_message"]; ok && v.(string) != "" {
		verificationMessageTemplateType.EmailMessage = aws.String(v.(string))
	}

	if v, ok := config["email_message_by_link"]; ok && v.(string) != "" {
		verificationMessageTemplateType.EmailMessageByLink = aws.String(v.(string))
	}

	if v, ok := config["email_subject"]; ok && v.(string) != "" {
		verificationMessageTemplateType.EmailSubject = aws.String(v.(string))
	}

	if v, ok := config["email_subject_by_link"]; ok && v.(string) != "" {
		verificationMessageTemplateType.EmailSubjectByLink = aws.String(v.(string))
	}

	if v, ok := config["sms_message"]; ok && v.(string) != "" {
		verificationMessageTemplateType.SmsMessage = aws.String(v.(string))
	}

	return verificationMessageTemplateType
}

func flattenCognitoUserPoolVerificationMessageTemplate(s *cognitoidentityprovider.VerificationMessageTemplateType) []map[string]interface{} {
	m := map[string]interface{}{}

	if s == nil {
		return nil
	}

	if s.DefaultEmailOption != nil {
		m["default_email_option"] = *s.DefaultEmailOption
	}

	if s.EmailMessage != nil {
		m["email_message"] = *s.EmailMessage
	}

	if s.EmailMessageByLink != nil {
		m["email_message_by_link"] = *s.EmailMessageByLink
	}

	if s.EmailSubject != nil {
		m["email_subject"] = *s.EmailSubject
	}

	if s.EmailSubjectByLink != nil {
		m["email_subject_by_link"] = *s.EmailSubjectByLink
	}

	if s.SmsMessage != nil {
		m["sms_message"] = *s.SmsMessage
	}

	if len(m) > 0 {
		return []map[string]interface{}{m}
	}

	return []map[string]interface{}{}
}

func buildLambdaInvokeArn(lambdaArn, region string) string {
	apiVersion := "2015-03-31"
	return fmt.Sprintf("arn:aws:apigateway:%s:lambda:path/%s/functions/%s/invocations",
		region, apiVersion, lambdaArn)
}

func sliceContainsMap(l []interface{}, m map[string]interface{}) (int, bool) {
	for i, t := range l {
		if reflect.DeepEqual(m, t.(map[string]interface{})) {
			return i, true
		}
	}

	return -1, false
}

func expandAwsSsmTargets(d *schema.ResourceData) []*ssm.Target {
	targets := make([]*ssm.Target, 0)

	targetConfig := d.Get("targets").([]interface{})

	for _, tConfig := range targetConfig {
		config := tConfig.(map[string]interface{})

		target := &ssm.Target{
			Key:    aws.String(config["key"].(string)),
			Values: expandStringList(config["values"].([]interface{})),
		}

		targets = append(targets, target)
	}

	return targets
}

func flattenAwsSsmTargets(targets []*ssm.Target) []map[string]interface{} {
	if len(targets) == 0 {
		return nil
	}

	result := make([]map[string]interface{}, 0, len(targets))
	for _, target := range targets {
		t := make(map[string]interface{}, 1)
		t["key"] = *target.Key
		t["values"] = flattenStringList(target.Values)

		result = append(result, t)
	}

	return result
}

func expandFieldToMatch(d map[string]interface{}) *waf.FieldToMatch {
	ftm := &waf.FieldToMatch{
		Type: aws.String(d["type"].(string)),
	}
	if data, ok := d["data"].(string); ok && data != "" {
		ftm.Data = aws.String(data)
	}
	return ftm
}

func flattenFieldToMatch(fm *waf.FieldToMatch) []interface{} {
	m := make(map[string]interface{})
	if fm.Data != nil {
		m["data"] = *fm.Data
	}
	if fm.Type != nil {
		m["type"] = *fm.Type
	}
	return []interface{}{m}
}

// escapeJsonPointer escapes string per RFC 6901
// so it can be used as path in JSON patch operations
func escapeJsonPointer(path string) string {
	path = strings.Replace(path, "~", "~0", -1)
	path = strings.Replace(path, "/", "~1", -1)
	return path
}

// Like ec2.GroupIdentifier but with additional rule description.
type GroupIdentifier struct {
	// The ID of the security group.
	GroupId *string

	// The name of the security group.
	GroupName *string

	Description *string
}

func expandCognitoIdentityPoolRoles(config map[string]interface{}) map[string]*string {
	m := map[string]*string{}
	for k, v := range config {
		s := v.(string)
		m[k] = &s
	}
	return m
}

func flattenCognitoIdentityPoolRoles(config map[string]*string) map[string]string {
	m := map[string]string{}
	for k, v := range config {
		m[k] = *v
	}
	return m
}

func expandCognitoIdentityPoolRoleMappingsAttachment(rms []interface{}) map[string]*cognitoidentity.RoleMapping {
	values := make(map[string]*cognitoidentity.RoleMapping, 0)

	if len(rms) == 0 {
		return values
	}

	for _, v := range rms {
		rm := v.(map[string]interface{})
		key := rm["identity_provider"].(string)

		roleMapping := &cognitoidentity.RoleMapping{
			Type: aws.String(rm["type"].(string)),
		}

		if sv, ok := rm["ambiguous_role_resolution"].(string); ok {
			roleMapping.AmbiguousRoleResolution = aws.String(sv)
		}

		if mr, ok := rm["mapping_rule"].([]interface{}); ok && len(mr) > 0 {
			rct := &cognitoidentity.RulesConfigurationType{}
			mappingRules := make([]*cognitoidentity.MappingRule, 0)

			for _, r := range mr {
				rule := r.(map[string]interface{})
				mr := &cognitoidentity.MappingRule{
					Claim:     aws.String(rule["claim"].(string)),
					MatchType: aws.String(rule["match_type"].(string)),
					RoleARN:   aws.String(rule["role_arn"].(string)),
					Value:     aws.String(rule["value"].(string)),
				}

				mappingRules = append(mappingRules, mr)
			}

			rct.Rules = mappingRules
			roleMapping.RulesConfiguration = rct
		}

		values[key] = roleMapping
	}

	return values
}

func flattenCognitoIdentityPoolRoleMappingsAttachment(rms map[string]*cognitoidentity.RoleMapping) []map[string]interface{} {
	roleMappings := make([]map[string]interface{}, 0)

	if rms == nil {
		return roleMappings
	}

	for k, v := range rms {
		m := make(map[string]interface{})

		if v == nil {
			return nil
		}

		if v.Type != nil {
			m["type"] = *v.Type
		}

		if v.AmbiguousRoleResolution != nil {
			m["ambiguous_role_resolution"] = *v.AmbiguousRoleResolution
		}

		if v.RulesConfiguration != nil && v.RulesConfiguration.Rules != nil {
			m["mapping_rule"] = flattenCognitoIdentityPoolRolesAttachmentMappingRules(v.RulesConfiguration.Rules)
		}

		m["identity_provider"] = k
		roleMappings = append(roleMappings, m)
	}

	return roleMappings
}

func flattenCognitoIdentityPoolRolesAttachmentMappingRules(d []*cognitoidentity.MappingRule) []interface{} {
	rules := make([]interface{}, 0)

	for _, rule := range d {
		r := make(map[string]interface{})
		r["claim"] = *rule.Claim
		r["match_type"] = *rule.MatchType
		r["role_arn"] = *rule.RoleARN
		r["value"] = *rule.Value

		rules = append(rules, r)
	}

	return rules
}

func flattenRedshiftLogging(ls *redshift.LoggingStatus) []interface{} {
	if ls == nil {
		return []interface{}{}
	}

	cfg := make(map[string]interface{}, 0)
	cfg["enabled"] = *ls.LoggingEnabled
	if ls.BucketName != nil {
		cfg["bucket_name"] = *ls.BucketName
	}
	if ls.S3KeyPrefix != nil {
		cfg["s3_key_prefix"] = *ls.S3KeyPrefix
	}
	return []interface{}{cfg}
}

func flattenRedshiftSnapshotCopy(scs *redshift.ClusterSnapshotCopyStatus) []interface{} {
	if scs == nil {
		return []interface{}{}
	}

	cfg := make(map[string]interface{}, 0)
	if scs.DestinationRegion != nil {
		cfg["destination_region"] = *scs.DestinationRegion
	}
	if scs.RetentionPeriod != nil {
		cfg["retention_period"] = *scs.RetentionPeriod
	}
	if scs.SnapshotCopyGrantName != nil {
		cfg["grant_name"] = *scs.SnapshotCopyGrantName
	}

	return []interface{}{cfg}
}

// cannonicalXML reads XML in a string and re-writes it canonically, used for
// comparing XML for logical equivalency
func canonicalXML(s string) (string, error) {
	doc := etree.NewDocument()
	doc.WriteSettings.CanonicalEndTags = true
	if err := doc.ReadFromString(s); err != nil {
		return "", err
	}

	rawString, err := doc.WriteToString()
	if err != nil {
		return "", err
	}

	re := regexp.MustCompile(`\s`)
	results := re.ReplaceAllString(rawString, "")
	return results, nil
}

func expandMqUsers(cfg []interface{}) []*mq.User {
	users := make([]*mq.User, len(cfg), len(cfg))
	for i, m := range cfg {
		u := m.(map[string]interface{})
		user := mq.User{
			Username: aws.String(u["username"].(string)),
			Password: aws.String(u["password"].(string)),
		}
		if v, ok := u["console_access"]; ok {
			user.ConsoleAccess = aws.Bool(v.(bool))
		}
		if v, ok := u["groups"]; ok {
			user.Groups = expandStringList(v.(*schema.Set).List())
		}
		users[i] = &user
	}
	return users
}

// We use cfgdUsers to get & set the password
func flattenMqUsers(users []*mq.User, cfgUsers []interface{}) *schema.Set {
	existingPairs := make(map[string]string, 0)
	for _, u := range cfgUsers {
		user := u.(map[string]interface{})
		username := user["username"].(string)
		existingPairs[username] = user["password"].(string)
	}

	out := make([]interface{}, 0)
	for _, u := range users {
		password := ""
		if p, ok := existingPairs[*u.Username]; ok {
			password = p
		}
		m := map[string]interface{}{
			"username": *u.Username,
			"password": password,
		}
		if u.ConsoleAccess != nil {
			m["console_access"] = *u.ConsoleAccess
		}
		if len(u.Groups) > 0 {
			m["groups"] = schema.NewSet(schema.HashString, flattenStringList(u.Groups))
		}
		out = append(out, m)
	}
	return schema.NewSet(resourceAwsMqUserHash, out)
}

func expandMqWeeklyStartTime(cfg []interface{}) *mq.WeeklyStartTime {
	if len(cfg) < 1 {
		return nil
	}

	m := cfg[0].(map[string]interface{})
	return &mq.WeeklyStartTime{
		DayOfWeek: aws.String(m["day_of_week"].(string)),
		TimeOfDay: aws.String(m["time_of_day"].(string)),
		TimeZone:  aws.String(m["time_zone"].(string)),
	}
}

func flattenMqWeeklyStartTime(wst *mq.WeeklyStartTime) []interface{} {
	if wst == nil {
		return []interface{}{}
	}
	m := make(map[string]interface{}, 0)
	if wst.DayOfWeek != nil {
		m["day_of_week"] = *wst.DayOfWeek
	}
	if wst.TimeOfDay != nil {
		m["time_of_day"] = *wst.TimeOfDay
	}
	if wst.TimeZone != nil {
		m["time_zone"] = *wst.TimeZone
	}
	return []interface{}{m}
}

func expandMqConfigurationId(cfg []interface{}) *mq.ConfigurationId {
	if len(cfg) < 1 {
		return nil
	}

	m := cfg[0].(map[string]interface{})
	out := mq.ConfigurationId{
		Id: aws.String(m["id"].(string)),
	}
	if v, ok := m["revision"].(int); ok && v > 0 {
		out.Revision = aws.Int64(int64(v))
	}

	return &out
}

func flattenMqConfigurationId(cid *mq.ConfigurationId) []interface{} {
	if cid == nil {
		return []interface{}{}
	}
	m := make(map[string]interface{}, 0)
	if cid.Id != nil {
		m["id"] = *cid.Id
	}
	if cid.Revision != nil {
		m["revision"] = *cid.Revision
	}
	return []interface{}{m}
}

func flattenMqBrokerInstances(instances []*mq.BrokerInstance) []interface{} {
	if len(instances) == 0 {
		return []interface{}{}
	}
	l := make([]interface{}, len(instances), len(instances))
	for i, instance := range instances {
		m := make(map[string]interface{}, 0)
		if instance.ConsoleURL != nil {
			m["console_url"] = *instance.ConsoleURL
		}
		if len(instance.Endpoints) > 0 {
			m["endpoints"] = aws.StringValueSlice(instance.Endpoints)
		}
		l[i] = m
	}

	return l
}
