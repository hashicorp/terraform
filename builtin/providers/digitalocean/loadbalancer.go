package digitalocean

import (
	"context"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/hashicorp/terraform/helper/resource"
)

func loadbalancerStateRefreshFunc(client *godo.Client, loadbalancerId string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		lb, _, err := client.LoadBalancers.Get(context.Background(), loadbalancerId)
		if err != nil {
			return nil, "", fmt.Errorf("Error issuing read request in LoadbalancerStateRefreshFunc to DigitalOcean for Load Balancer '%s': %s", loadbalancerId, err)
		}

		return lb, lb.Status, nil
	}
}

func expandStickySessions(config []interface{}) *godo.StickySessions {
	stickysessionConfig := config[0].(map[string]interface{})

	stickySession := &godo.StickySessions{
		Type: stickysessionConfig["type"].(string),
	}

	if v, ok := stickysessionConfig["cookie_name"]; ok {
		stickySession.CookieName = v.(string)
	}

	if v, ok := stickysessionConfig["cookie_ttl_seconds"]; ok {
		stickySession.CookieTtlSeconds = v.(int)
	}

	return stickySession
}

func expandHealthCheck(config []interface{}) *godo.HealthCheck {
	healthcheckConfig := config[0].(map[string]interface{})

	healthcheck := &godo.HealthCheck{
		Protocol:               healthcheckConfig["protocol"].(string),
		Port:                   healthcheckConfig["port"].(int),
		CheckIntervalSeconds:   healthcheckConfig["check_interval_seconds"].(int),
		ResponseTimeoutSeconds: healthcheckConfig["response_timeout_seconds"].(int),
		UnhealthyThreshold:     healthcheckConfig["unhealthy_threshold"].(int),
		HealthyThreshold:       healthcheckConfig["healthy_threshold"].(int),
	}

	if v, ok := healthcheckConfig["path"]; ok {
		healthcheck.Path = v.(string)
	}

	return healthcheck
}

func expandForwardingRules(config []interface{}) []godo.ForwardingRule {
	forwardingRules := make([]godo.ForwardingRule, 0, len(config))

	for _, rawRule := range config {
		rule := rawRule.(map[string]interface{})

		r := godo.ForwardingRule{
			EntryPort:      rule["entry_port"].(int),
			EntryProtocol:  rule["entry_protocol"].(string),
			TargetPort:     rule["target_port"].(int),
			TargetProtocol: rule["target_protocol"].(string),
			TlsPassthrough: rule["tls_passthrough"].(bool),
		}

		if v, ok := rule["certificate_id"]; ok {
			r.CertificateID = v.(string)
		}

		forwardingRules = append(forwardingRules, r)

	}

	return forwardingRules
}

func flattenDropletIds(list []int) []interface{} {
	vs := make([]interface{}, 0, len(list))
	for _, v := range list {
		vs = append(vs, v)
	}
	return vs
}

func flattenHealthChecks(health *godo.HealthCheck) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, 1)

	if health != nil {

		r := make(map[string]interface{})
		r["protocol"] = (*health).Protocol
		r["port"] = (*health).Port
		r["path"] = (*health).Path
		r["check_interval_seconds"] = (*health).CheckIntervalSeconds
		r["response_timeout_seconds"] = (*health).ResponseTimeoutSeconds
		r["unhealthy_threshold"] = (*health).UnhealthyThreshold
		r["healthy_threshold"] = (*health).HealthyThreshold

		result = append(result, r)
	}

	return result
}

func flattenStickySessions(session *godo.StickySessions) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, 1)

	if session != nil {

		r := make(map[string]interface{})
		r["type"] = (*session).Type
		r["cookie_name"] = (*session).CookieName
		r["cookie_ttl_seconds"] = (*session).CookieTtlSeconds

		result = append(result, r)
	}

	return result
}

func flattenForwardingRules(rules []godo.ForwardingRule) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, 1)

	if rules != nil {
		for _, rule := range rules {
			r := make(map[string]interface{})
			r["entry_protocol"] = rule.EntryProtocol
			r["entry_port"] = rule.EntryPort
			r["target_protocol"] = rule.TargetProtocol
			r["target_port"] = rule.TargetPort
			r["certificate_id"] = rule.CertificateID
			r["tls_passthrough"] = rule.TlsPassthrough

			result = append(result, r)
		}
	}

	return result
}
