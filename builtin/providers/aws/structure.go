package aws

import (
	"github.com/mitchellh/goamz/ec2"
	"github.com/mitchellh/goamz/elb"
	"log"
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
		log.Println(newP)

		// Loop over the array of sg ids and built
		// compatibile goamz objects
		groups := expandStringList(newP["security_groups"].([]interface{}))
		expandedGroups := make([]ec2.UserSecurityGroup, 0, len(groups))
		for _, g := range groups {
			newG := ec2.UserSecurityGroup{
				Id: g,
			}
			expandedGroups = append(expandedGroups, newG)
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

// Takes the result of flatmap.Expand for an array of strings
// and returns a []string
func expandStringList(configured []interface{}) []string {
	vs := make([]string, 0, len(configured))
	for _, v := range configured {
		vs = append(vs, v.(string))
	}
	return vs
}
