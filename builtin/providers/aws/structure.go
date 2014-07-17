package aws

import (
	"strings"

	"github.com/mitchellh/goamz/autoscaling"
	"github.com/mitchellh/goamz/ec2"
	"github.com/mitchellh/goamz/elb"
)

// Takes the result of flatmap.Expand for an array of listeners and
// returns ELB API compatible objects
func expandListeners(configured []interface{}) []elb.Listener {
	listeners := make([]elb.Listener, 0, len(configured))

	// Loop over our configured listeners and create
	// an array of goamz compatabile objects
	for _, listener := range configured {
		newL := listener.(map[string]interface{})

		l := elb.Listener{
			InstancePort:     int64(newL["instance_port"].(int)),
			InstanceProtocol: newL["instance_protocol"].(string),
			LoadBalancerPort: int64(newL["lb_port"].(int)),
			Protocol:         newL["lb_protocol"].(string),
		}

		listeners = append(listeners, l)
	}

	return listeners
}

// Takes the result of flatmap.Expand for an array of ingress/egress
// security group rules and returns EC2 API compatible objects
func expandIPPerms(configured []interface{}) []ec2.IPPerm {
	perms := make([]ec2.IPPerm, 0, len(configured))

	// Loop over our configured permissions and create
	// an array of goamz/ec2 compatabile objects
	for _, perm := range configured {
		newP := perm.(map[string]interface{})
		// Loop over the array of sg ids and built
		// compatibile goamz objects
		expandedGroups := []ec2.UserSecurityGroup{}
		configGroups, ok := newP["security_groups"].([]interface{})
		if ok {
			gs := expandStringList(configGroups)
			for _, g := range gs {
				newG := ec2.UserSecurityGroup{
					Id: g,
				}
				expandedGroups = append(expandedGroups, newG)
			}
		}

		// Create the permission objet
		p := ec2.IPPerm{
			Protocol:     newP["protocol"].(string),
			FromPort:     newP["from_port"].(int),
			ToPort:       newP["to_port"].(int),
			SourceIPs:    expandStringList(newP["cidr_blocks"].([]interface{})),
			SourceGroups: expandedGroups,
		}

		perms = append(perms, p)
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
		n["cidr_blocks"] = perm.SourceIPs
		n["security_groups"] = flattenSecurityGroups(perm.SourceGroups)
		result = append(result, n)
	}

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
