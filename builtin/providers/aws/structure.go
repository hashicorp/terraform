package aws

import (
	"strconv"
	"strings"

	"github.com/mitchellh/goamz/autoscaling"
	"github.com/mitchellh/goamz/ec2"
	"github.com/mitchellh/goamz/elb"
)

// Takes the result of flatmap.Expand for an array of listeners and
// returns ELB API compatible objects
func expandListeners(configured []interface{}) ([]elb.Listener, error) {
	listeners := make([]elb.Listener, 0, len(configured))

	// Loop over our configured listeners and create
	// an array of goamz compatabile objects
	for _, listener := range configured {
		newL := listener.(map[string]interface{})

		instancePort, err := strconv.ParseInt(newL["instance_port"].(string), 0, 0)
		lbPort, err := strconv.ParseInt(newL["lb_port"].(string), 0, 0)

		if err != nil {
			return nil, err
		}

		l := elb.Listener{
			InstancePort:     instancePort,
			InstanceProtocol: newL["instance_protocol"].(string),
			LoadBalancerPort: lbPort,
			Protocol:         newL["lb_protocol"].(string),
		}

		if attr, ok := newL["ssl_certificate_id"].(string); ok {
			l.SSLCertificateId = attr
		}


		listeners = append(listeners, l)
	}

	return listeners, nil
}

// Takes the result of flatmap.Expand for an array of ingress/egress
// security group rules and returns EC2 API compatible objects
func expandIPPerms(id string, configured []interface{}) []ec2.IPPerm {
	perms := make([]ec2.IPPerm, len(configured))
	for i, mRaw := range configured {
		var perm ec2.IPPerm
		m := mRaw.(map[string]interface{})

		perm.FromPort = m["from_port"].(int)
		perm.ToPort = m["to_port"].(int)
		perm.Protocol = m["protocol"].(string)

		var groups []string
		if raw, ok := m["security_groups"]; ok {
			list := raw.([]interface{})
			for _, v := range list {
				groups = append(groups, v.(string))
			}
		}
		if v, ok := m["self"]; ok && v.(bool) {
			groups = append(groups, id)
		}

		if len(groups) > 0 {
			perm.SourceGroups = make([]ec2.UserSecurityGroup, len(groups))
			for i, name := range groups {
				ownerId, id := "", name
				if items := strings.Split(id, "/"); len(items) > 1 {
					ownerId, id = items[0], items[1]
				}

				perm.SourceGroups[i] = ec2.UserSecurityGroup{
					Id:      id,
					OwnerId: ownerId,
				}
			}
		}

		if raw, ok := m["cidr_blocks"]; ok {
			list := raw.([]interface{})
			perm.SourceIPs = make([]string, len(list))
			for i, v := range list {
				perm.SourceIPs[i] = v.(string)
			}
		}

		perms[i] = perm
	}

	return perms
}

// Flattens an array of ipPerms into a list of primitives that
// flatmap.Flatten() can handle
func flattenIPPerms(list []ec2.IPPerm) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(list))

	for _, perm := range list {
		n := make(map[string]interface{})
		n["from_port"] = perm.FromPort
		n["protocol"] = perm.Protocol
		n["to_port"] = perm.ToPort

		if len(perm.SourceIPs) > 0 {
			n["cidr_blocks"] = perm.SourceIPs
		}

		if v := flattenSecurityGroups(perm.SourceGroups); len(v) > 0 {
			n["security_groups"] = v
		}

		result = append(result, n)
	}

	return result
}

// Flattens a health check into something that flatmap.Flatten()
// can handle
func flattenHealthCheck(check elb.HealthCheck) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, 1)

	chk := make(map[string]interface{})
	chk["unhealthy_threshold"] = int(check.UnhealthyThreshold)
	chk["healthy_threshold"] = int(check.HealthyThreshold)
	chk["target"] = check.Target
	chk["timeout"] = int(check.Timeout)
	chk["interval"] = int(check.Interval)

	result = append(result, chk)

	return result
}

// Flattens an array of UserSecurityGroups into a []string
func flattenSecurityGroups(list []ec2.UserSecurityGroup) []string {
	result := make([]string, 0, len(list))
	for _, g := range list {
		result = append(result, g.Id)
	}
	return result
}

// Flattens an array of SecurityGroups into a []string
func flattenAutoscalingSecurityGroups(list []autoscaling.SecurityGroup) []string {
	result := make([]string, 0, len(list))
	for _, g := range list {
		result = append(result, g.SecurityGroup)
	}
	return result
}

// Flattens an array of AvailabilityZones into a []string
func flattenAvailabilityZones(list []autoscaling.AvailabilityZone) []string {
	result := make([]string, 0, len(list))
	for _, g := range list {
		result = append(result, g.AvailabilityZone)
	}
	return result
}

// Flattens an array of LoadBalancerName into a []string
func flattenLoadBalancers(list []autoscaling.LoadBalancerName) []string {
	result := make([]string, 0, len(list))
	for _, g := range list {
		if g.LoadBalancerName != "" {
			result = append(result, g.LoadBalancerName)
		}
	}
	return result
}

// Flattens an array of Instances into a []string
func flattenInstances(list []elb.Instance) []string {
	result := make([]string, 0, len(list))
	for _, i := range list {
		result = append(result, i.InstanceId)
	}
	return result
}

// Takes the result of flatmap.Expand for an array of strings
// and returns a []string
func expandStringList(configured []interface{}) []string {
	// here we special case the * expanded lists. For example:
	//
	//	 instances = ["${aws_instance.foo.*.id}"]
	//
	if len(configured) == 1 && strings.Contains(configured[0].(string), ",") {
		return strings.Split(configured[0].(string), ",")
	}

	vs := make([]string, 0, len(configured))
	for _, v := range configured {
		vs = append(vs, v.(string))
	}
	return vs
}
