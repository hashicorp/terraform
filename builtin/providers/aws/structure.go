package aws

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/directoryservice"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/aws/aws-sdk-go/service/elasticbeanstalk"
	elasticsearch "github.com/aws/aws-sdk-go/service/elasticsearchservice"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/redshift"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/hashicorp/terraform/helper/schema"
)

// Takes the result of flatmap.Expand for an array of listeners and
// returns ELB API compatible objects
func expandListeners(configured []interface{}) ([]*elb.Listener, error) {
	listeners := make([]*elb.Listener, 0, len(configured))

	// Loop over our configured listeners and create
	// an array of aws-sdk-go compatabile objects
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
			ContainerName:    aws.String(data["container_name"].(string)),
			ContainerPort:    aws.Int64(int64(data["container_port"].(int))),
			LoadBalancerName: aws.String(data["elb_name"].(string)),
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
				"from_port (%d) and to_port (%d) must both be 0 to use the the 'ALL' \"-1\" protocol!",
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

		perms[i] = &perm
	}

	return perms, nil
}

// Takes the result of flatmap.Expand for an array of parameters and
// returns Parameter API compatible objects
func expandParameters(configured []interface{}) ([]*rds.Parameter, error) {
	var parameters []*rds.Parameter

	// Loop over our configured parameters and create
	// an array of aws-sdk-go compatabile objects
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
	// an array of aws-sdk-go compatabile objects
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
	// an array of aws-sdk-go compatabile objects
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

	if l != nil && *l.Enabled {
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

		result = append(result, r)
	}

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

// Flattens an array of UserSecurityGroups into a []*ec2.GroupIdentifier
func flattenSecurityGroups(list []*ec2.UserIdGroupPair, ownerId *string) []*ec2.GroupIdentifier {
	result := make([]*ec2.GroupIdentifier, 0, len(list))
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
			result = append(result, &ec2.GroupIdentifier{
				GroupId: id,
			})
		} else {
			result = append(result, &ec2.GroupIdentifier{
				GroupId:   g.GroupId,
				GroupName: id,
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
			"elb_name":       *loadBalancer.LoadBalancerName,
			"container_name": *loadBalancer.ContainerName,
			"container_port": *loadBalancer.ContainerPort,
		}
		result = append(result, l)
	}
	return result
}

// Encodes an array of ecs.ContainerDefinitions into a JSON string
func flattenEcsContainerDefinitions(definitions []*ecs.ContainerDefinition) (string, error) {
	byteArray, err := json.Marshal(definitions)
	if err != nil {
		return "", fmt.Errorf("Error encoding to JSON: %s", err)
	}

	n := bytes.Index(byteArray, []byte{0})
	return string(byteArray[:n]), nil
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
					settings = append(settings, map[string]interface{}{
						"name":  *j.Name,
						"value": *j.Value,
					})
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
				"value": strings.ToLower(*i.ParameterValue),
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
		vs = append(vs, aws.String(v.(string)))
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
	att["instance"] = *a.InstanceId
	att["device_index"] = *a.DeviceIndex
	att["attachment_id"] = *a.AttachmentId
	return att
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

func flattenResourceRecords(recs []*route53.ResourceRecord) []string {
	strs := make([]string, 0, len(recs))
	for _, r := range recs {
		if r.Value != nil {
			s := strings.Replace(*r.Value, "\"", "", 2)
			strs = append(strs, s)
		}
	}
	return strs
}

func expandResourceRecords(recs []interface{}, typeStr string) []*route53.ResourceRecord {
	records := make([]*route53.ResourceRecord, 0, len(recs))
	for _, r := range recs {
		s := r.(string)
		switch typeStr {
		case "TXT", "SPF":
			str := fmt.Sprintf("\"%s\"", s)
			records = append(records, &route53.ResourceRecord{Value: aws.String(str)})
		default:
			records = append(records, &route53.ResourceRecord{Value: aws.String(s)})
		}
	}
	return records
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

func flattenLambdaVpcConfigResponse(s *lambda.VpcConfigResponse) []map[string]interface{} {
	settings := make(map[string]interface{}, 0)

	if s == nil {
		return nil
	}

	if len(s.SubnetIds) == 0 && len(s.SecurityGroupIds) == 0 && s.VpcId == nil {
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

func flattenAsgEnabledMetrics(list []*autoscaling.EnabledMetric) []string {
	strs := make([]string, 0, len(list))
	for _, r := range list {
		if r.Metric != nil {
			strs = append(strs, *r.Metric)
		}
	}
	return strs
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

func expandApiGatewayMethodParametersJSONOperations(d *schema.ResourceData, key string, prefix string) ([]*apigateway.PatchOperation, error) {
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

func expandCloudWachLogMetricTransformations(m map[string]interface{}) []*cloudwatchlogs.MetricTransformation {
	transformation := cloudwatchlogs.MetricTransformation{
		MetricName:      aws.String(m["name"].(string)),
		MetricNamespace: aws.String(m["namespace"].(string)),
		MetricValue:     aws.String(m["value"].(string)),
	}

	return []*cloudwatchlogs.MetricTransformation{&transformation}
}

func flattenCloudWachLogMetricTransformations(ts []*cloudwatchlogs.MetricTransformation) map[string]string {
	m := make(map[string]string, 0)

	m["name"] = *ts[0].MetricName
	m["namespace"] = *ts[0].MetricNamespace
	m["value"] = *ts[0].MetricValue

	return m
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
// causing diffs inbetweeen resources that use lists. This avoids a bit of
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
