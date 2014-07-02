package aws

import (
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
