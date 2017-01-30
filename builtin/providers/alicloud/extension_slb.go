package alicloud

import (
	"fmt"
	"strings"

	"github.com/denverdino/aliyungo/slb"
)

type Listener struct {
	InstancePort     int
	LoadBalancerPort int
	Protocol         string
	SSLCertificateId string
	Bandwidth        int
}

// Takes the result of flatmap.Expand for an array of listeners and
// returns ELB API compatible objects
func expandListeners(configured []interface{}) ([]*Listener, error) {
	listeners := make([]*Listener, 0, len(configured))

	// Loop over our configured listeners and create
	// an array of aws-sdk-go compatabile objects
	for _, lRaw := range configured {
		data := lRaw.(map[string]interface{})

		ip := data["instance_port"].(int)
		lp := data["lb_port"].(int)
		l := &Listener{
			InstancePort:     ip,
			LoadBalancerPort: lp,
			Protocol:         data["lb_protocol"].(string),
			Bandwidth:        data["bandwidth"].(int),
		}

		if v, ok := data["ssl_certificate_id"]; ok {
			l.SSLCertificateId = v.(string)
		}

		var valid bool
		if l.SSLCertificateId != "" {
			// validate the protocol is correct
			for _, p := range []string{"https", "ssl"} {
				if strings.ToLower(l.Protocol) == p {
					valid = true
				}
			}
		} else {
			valid = true
		}

		if valid {
			listeners = append(listeners, l)
		} else {
			return nil, fmt.Errorf("[ERR] SLB Listener: ssl_certificate_id may be set only when protocol is 'https' or 'ssl'")
		}
	}

	return listeners, nil
}

func expandBackendServers(list []interface{}) []slb.BackendServerType {
	result := make([]slb.BackendServerType, 0, len(list))
	for _, i := range list {
		if i.(string) != "" {
			result = append(result, slb.BackendServerType{ServerId: i.(string), Weight: 100})
		}
	}
	return result
}
