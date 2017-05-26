package alicloud

import (
	"fmt"
	"strings"

	"github.com/denverdino/aliyungo/slb"
)

type Listener struct {
	slb.HTTPListenerType

	InstancePort     int
	LoadBalancerPort int
	Protocol         string
	//tcp & udp
	PersistenceTimeout int

	//https
	SSLCertificateId string

	//tcp
	HealthCheckType slb.HealthCheckType

	//api interface: http & https is HealthCheckTimeout, tcp & udp is HealthCheckConnectTimeout
	HealthCheckConnectTimeout int
}

type ListenerErr struct {
	ErrType string
	Err     error
}

func (e *ListenerErr) Error() string {
	return e.ErrType + " " + e.Err.Error()

}

const (
	HealthCheckErrType   = "healthCheckErrType"
	StickySessionErrType = "stickySessionErrType"
	CookieTimeOutErrType = "cookieTimeoutErrType"
	CookieErrType        = "cookieErrType"
)

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
		}

		l.Bandwidth = data["bandwidth"].(int)

		if v, ok := data["scheduler"]; ok {
			l.Scheduler = slb.SchedulerType(v.(string))
		}

		if v, ok := data["ssl_certificate_id"]; ok {
			l.SSLCertificateId = v.(string)
		}

		if v, ok := data["sticky_session"]; ok {
			l.StickySession = slb.FlagType(v.(string))
		}

		if v, ok := data["sticky_session_type"]; ok {
			l.StickySessionType = slb.StickySessionType(v.(string))
		}

		if v, ok := data["cookie_timeout"]; ok {
			l.CookieTimeout = v.(int)
		}

		if v, ok := data["cookie"]; ok {
			l.Cookie = v.(string)
		}

		if v, ok := data["persistence_timeout"]; ok {
			l.PersistenceTimeout = v.(int)
		}

		if v, ok := data["health_check"]; ok {
			l.HealthCheck = slb.FlagType(v.(string))
		}

		if v, ok := data["health_check_type"]; ok {
			l.HealthCheckType = slb.HealthCheckType(v.(string))
		}

		if v, ok := data["health_check_domain"]; ok {
			l.HealthCheckDomain = v.(string)
		}

		if v, ok := data["health_check_uri"]; ok {
			l.HealthCheckURI = v.(string)
		}

		if v, ok := data["health_check_connect_port"]; ok {
			l.HealthCheckConnectPort = v.(int)
		}

		if v, ok := data["healthy_threshold"]; ok {
			l.HealthyThreshold = v.(int)
		}

		if v, ok := data["unhealthy_threshold"]; ok {
			l.UnhealthyThreshold = v.(int)
		}

		if v, ok := data["health_check_timeout"]; ok {
			l.HealthCheckTimeout = v.(int)
		}

		if v, ok := data["health_check_interval"]; ok {
			l.HealthCheckInterval = v.(int)
		}

		if v, ok := data["health_check_http_code"]; ok {
			l.HealthCheckHttpCode = slb.HealthCheckHttpCodeType(v.(string))
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
