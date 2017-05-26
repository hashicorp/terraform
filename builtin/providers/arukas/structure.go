package arukas

import (
	API "github.com/arukasio/cli"
	"github.com/hashicorp/terraform/helper/schema"
	"net"
)

// Takes the result of flatmap.Expand for an array of strings
// and returns a []string
func expandStringList(configured []interface{}) []string {
	vs := make([]string, 0, len(configured))
	for _, v := range configured {
		vs = append(vs, string(v.(string)))
	}
	return vs
}

// Takes the result of schema.Set of strings and returns a []string
func expandStringSet(configured *schema.Set) []string {
	return expandStringList(configured.List())
}

// Takes list of pointers to strings. Expand to an array
// of raw strings and returns a []interface{}
// to keep compatibility w/ schema.NewSetschema.NewSet
func flattenStringList(list []string) []interface{} {
	vs := make([]interface{}, 0, len(list))
	for _, v := range list {
		vs = append(vs, v)
	}
	return vs
}

func expandEnvs(configured interface{}) API.Envs {
	var envs API.Envs
	if configured == nil {
		return envs
	}
	rawEnvs := configured.([]interface{})
	for _, raw := range rawEnvs {
		env := raw.(map[string]interface{})
		envs = append(envs, API.Env{Key: env["key"].(string), Value: env["value"].(string)})
	}
	return envs
}

func flattenEnvs(envs API.Envs) []interface{} {
	var ret []interface{}
	for _, env := range envs {
		r := map[string]interface{}{}
		r["key"] = env.Key
		r["value"] = env.Value
		ret = append(ret, r)
	}
	return ret
}

func expandPorts(configured interface{}) API.Ports {
	var ports API.Ports
	if configured == nil {
		return ports
	}
	rawPorts := configured.([]interface{})
	for _, raw := range rawPorts {
		port := raw.(map[string]interface{})
		ports = append(ports, API.Port{Protocol: port["protocol"].(string), Number: port["number"].(int)})
	}
	return ports
}

func flattenPorts(ports API.Ports) []interface{} {
	var ret []interface{}
	for _, port := range ports {
		r := map[string]interface{}{}
		r["protocol"] = port.Protocol
		r["number"] = port.Number
		ret = append(ret, r)
	}
	return ret
}
func flattenPortMappings(ports API.PortMappings) []interface{} {
	var ret []interface{}
	for _, tasks := range ports {
		for _, port := range tasks {
			r := map[string]interface{}{}
			ip := ""

			addrs, err := net.LookupHost(port.Host)
			if err == nil && len(addrs) > 0 {
				ip = addrs[0]
			}

			r["host"] = port.Host
			r["ipaddress"] = ip
			r["container_port"] = port.ContainerPort
			r["service_port"] = port.ServicePort
			ret = append(ret, r)
		}
	}
	return ret
}

func forceString(target interface{}) string {
	if target == nil {
		return ""
	}

	return target.(string)
}
