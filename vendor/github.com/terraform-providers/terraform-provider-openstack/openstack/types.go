package openstack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/keypairs"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/servergroups"
	"github.com/gophercloud/gophercloud/openstack/dns/v2/recordsets"
	"github.com/gophercloud/gophercloud/openstack/dns/v2/zones"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/fwaas/firewalls"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/fwaas/policies"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/fwaas/routerinsertion"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/fwaas/rules"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/floatingips"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
)

// LogRoundTripper satisfies the http.RoundTripper interface and is used to
// customize the default http client RoundTripper to allow for logging.
type LogRoundTripper struct {
	Rt      http.RoundTripper
	OsDebug bool
}

// RoundTrip performs a round-trip HTTP request and logs relevant information about it.
func (lrt *LogRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	defer func() {
		if request.Body != nil {
			request.Body.Close()
		}
	}()

	// for future reference, this is how to access the Transport struct:
	//tlsconfig := lrt.Rt.(*http.Transport).TLSClientConfig

	var err error

	if lrt.OsDebug {
		log.Printf("[DEBUG] OpenStack Request URL: %s %s", request.Method, request.URL)
		log.Printf("[DEBUG] Openstack Request Headers:\n%s", FormatHeaders(request.Header, "\n"))

		if request.Body != nil {
			request.Body, err = lrt.logRequest(request.Body, request.Header.Get("Content-Type"))
			if err != nil {
				return nil, err
			}
		}
	}

	response, err := lrt.Rt.RoundTrip(request)
	if response == nil {
		return nil, err
	}

	if lrt.OsDebug {
		log.Printf("[DEBUG] Openstack Response Code: %d", response.StatusCode)
		log.Printf("[DEBUG] Openstack Response Headers:\n%s", FormatHeaders(response.Header, "\n"))

		response.Body, err = lrt.logResponse(response.Body, response.Header.Get("Content-Type"))
	}

	return response, err
}

// logRequest will log the HTTP Request details.
// If the body is JSON, it will attempt to be pretty-formatted.
func (lrt *LogRoundTripper) logRequest(original io.ReadCloser, contentType string) (io.ReadCloser, error) {
	defer original.Close()

	var bs bytes.Buffer
	_, err := io.Copy(&bs, original)
	if err != nil {
		return nil, err
	}

	// Handle request contentType
	if strings.HasPrefix(contentType, "application/json") {
		debugInfo := lrt.formatJSON(bs.Bytes())
		log.Printf("[DEBUG] OpenStack Request Body: %s", debugInfo)
	} else {
		log.Printf("[DEBUG] OpenStack Request Body: %s", bs.String())
	}

	return ioutil.NopCloser(strings.NewReader(bs.String())), nil
}

// logResponse will log the HTTP Response details.
// If the body is JSON, it will attempt to be pretty-formatted.
func (lrt *LogRoundTripper) logResponse(original io.ReadCloser, contentType string) (io.ReadCloser, error) {
	if strings.HasPrefix(contentType, "application/json") {
		var bs bytes.Buffer
		defer original.Close()
		_, err := io.Copy(&bs, original)
		if err != nil {
			return nil, err
		}
		debugInfo := lrt.formatJSON(bs.Bytes())
		if debugInfo != "" {
			log.Printf("[DEBUG] OpenStack Response Body: %s", debugInfo)
		}
		return ioutil.NopCloser(strings.NewReader(bs.String())), nil
	}

	log.Printf("[DEBUG] Not logging because OpenStack response body isn't JSON")
	return original, nil
}

// formatJSON will try to pretty-format a JSON body.
// It will also mask known fields which contain sensitive information.
func (lrt *LogRoundTripper) formatJSON(raw []byte) string {
	var data map[string]interface{}

	err := json.Unmarshal(raw, &data)
	if err != nil {
		log.Printf("[DEBUG] Unable to parse OpenStack JSON: %s", err)
		return string(raw)
	}

	// Mask known password fields
	if v, ok := data["auth"].(map[string]interface{}); ok {
		if v, ok := v["identity"].(map[string]interface{}); ok {
			if v, ok := v["password"].(map[string]interface{}); ok {
				if v, ok := v["user"].(map[string]interface{}); ok {
					v["password"] = "***"
				}
			}
		}
	}

	// Ignore the catalog
	if v, ok := data["token"].(map[string]interface{}); ok {
		if _, ok := v["catalog"]; ok {
			return ""
		}
	}

	pretty, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Printf("[DEBUG] Unable to re-marshal OpenStack JSON: %s", err)
		return string(raw)
	}

	return string(pretty)
}

// Firewall is an OpenStack firewall.
type Firewall struct {
	firewalls.Firewall
	routerinsertion.FirewallExt
}

// FirewallCreateOpts represents the attributes used when creating a new firewall.
type FirewallCreateOpts struct {
	firewalls.CreateOptsBuilder
	ValueSpecs map[string]string `json:"value_specs,omitempty"`
}

// ToFirewallCreateMap casts a CreateOptsExt struct to a map.
// It overrides firewalls.ToFirewallCreateMap to add the ValueSpecs field.
func (opts FirewallCreateOpts) ToFirewallCreateMap() (map[string]interface{}, error) {
	body, err := opts.CreateOptsBuilder.ToFirewallCreateMap()
	if err != nil {
		return nil, err
	}

	return AddValueSpecs(body), nil
}

//FirewallUpdateOpts
type FirewallUpdateOpts struct {
	firewalls.UpdateOptsBuilder
}

func (opts FirewallUpdateOpts) ToFirewallUpdateMap() (map[string]interface{}, error) {
	return BuildRequest(opts, "firewall")
}

// FloatingIPCreateOpts represents the attributes used when creating a new floating ip.
type FloatingIPCreateOpts struct {
	floatingips.CreateOpts
	ValueSpecs map[string]string `json:"value_specs,omitempty"`
}

// ToFloatingIPCreateMap casts a CreateOpts struct to a map.
// It overrides floatingips.ToFloatingIPCreateMap to add the ValueSpecs field.
func (opts FloatingIPCreateOpts) ToFloatingIPCreateMap() (map[string]interface{}, error) {
	return BuildRequest(opts, "floatingip")
}

// KeyPairCreateOpts represents the attributes used when creating a new keypair.
type KeyPairCreateOpts struct {
	keypairs.CreateOpts
	ValueSpecs map[string]string `json:"value_specs,omitempty"`
}

// ToKeyPairCreateMap casts a CreateOpts struct to a map.
// It overrides keypairs.ToKeyPairCreateMap to add the ValueSpecs field.
func (opts KeyPairCreateOpts) ToKeyPairCreateMap() (map[string]interface{}, error) {
	return BuildRequest(opts, "keypair")
}

// NetworkCreateOpts represents the attributes used when creating a new network.
type NetworkCreateOpts struct {
	networks.CreateOpts
	ValueSpecs map[string]string `json:"value_specs,omitempty"`
}

// ToNetworkCreateMap casts a CreateOpts struct to a map.
// It overrides networks.ToNetworkCreateMap to add the ValueSpecs field.
func (opts NetworkCreateOpts) ToNetworkCreateMap() (map[string]interface{}, error) {
	return BuildRequest(opts, "network")
}

// PolicyCreateOpts represents the attributes used when creating a new firewall policy.
type PolicyCreateOpts struct {
	policies.CreateOpts
	ValueSpecs map[string]string `json:"value_specs,omitempty"`
}

// ToPolicyCreateMap casts a CreateOpts struct to a map.
// It overrides policies.ToFirewallPolicyCreateMap to add the ValueSpecs field.
func (opts PolicyCreateOpts) ToFirewallPolicyCreateMap() (map[string]interface{}, error) {
	return BuildRequest(opts, "firewall_policy")
}

// PortCreateOpts represents the attributes used when creating a new port.
type PortCreateOpts struct {
	ports.CreateOpts
	ValueSpecs map[string]string `json:"value_specs,omitempty"`
}

// ToPortCreateMap casts a CreateOpts struct to a map.
// It overrides ports.ToPortCreateMap to add the ValueSpecs field.
func (opts PortCreateOpts) ToPortCreateMap() (map[string]interface{}, error) {
	return BuildRequest(opts, "port")
}

// RecordSetCreateOpts represents the attributes used when creating a new DNS record set.
type RecordSetCreateOpts struct {
	recordsets.CreateOpts
	ValueSpecs map[string]string `json:"value_specs,omitempty"`
}

// ToRecordSetCreateMap casts a CreateOpts struct to a map.
// It overrides recordsets.ToRecordSetCreateMap to add the ValueSpecs field.
func (opts RecordSetCreateOpts) ToRecordSetCreateMap() (map[string]interface{}, error) {
	b, err := BuildRequest(opts, "")
	if err != nil {
		return nil, err
	}

	if m, ok := b[""].(map[string]interface{}); ok {
		return m, nil
	}

	return nil, fmt.Errorf("Expected map but got %T", b[""])
}

// RouterCreateOpts represents the attributes used when creating a new router.
type RouterCreateOpts struct {
	routers.CreateOpts
	ValueSpecs map[string]string `json:"value_specs,omitempty"`
}

// ToRouterCreateMap casts a CreateOpts struct to a map.
// It overrides routers.ToRouterCreateMap to add the ValueSpecs field.
func (opts RouterCreateOpts) ToRouterCreateMap() (map[string]interface{}, error) {
	return BuildRequest(opts, "router")
}

// RuleCreateOpts represents the attributes used when creating a new firewall rule.
type RuleCreateOpts struct {
	rules.CreateOpts
	ValueSpecs map[string]string `json:"value_specs,omitempty"`
}

// ToRuleCreateMap casts a CreateOpts struct to a map.
// It overrides rules.ToRuleCreateMap to add the ValueSpecs field.
func (opts RuleCreateOpts) ToRuleCreateMap() (map[string]interface{}, error) {
	b, err := BuildRequest(opts, "firewall_rule")
	if err != nil {
		return nil, err
	}

	if m := b["firewall_rule"].(map[string]interface{}); m["protocol"] == "any" {
		m["protocol"] = nil
	}

	return b, nil
}

// ServerGroupCreateOpts represents the attributes used when creating a new router.
type ServerGroupCreateOpts struct {
	servergroups.CreateOpts
	ValueSpecs map[string]string `json:"value_specs,omitempty"`
}

// ToServerGroupCreateMap casts a CreateOpts struct to a map.
// It overrides routers.ToServerGroupCreateMap to add the ValueSpecs field.
func (opts ServerGroupCreateOpts) ToServerGroupCreateMap() (map[string]interface{}, error) {
	return BuildRequest(opts, "server_group")
}

// SubnetCreateOpts represents the attributes used when creating a new subnet.
type SubnetCreateOpts struct {
	subnets.CreateOpts
	ValueSpecs map[string]string `json:"value_specs,omitempty"`
}

// ToSubnetCreateMap casts a CreateOpts struct to a map.
// It overrides subnets.ToSubnetCreateMap to add the ValueSpecs field.
func (opts SubnetCreateOpts) ToSubnetCreateMap() (map[string]interface{}, error) {
	b, err := BuildRequest(opts, "subnet")
	if err != nil {
		return nil, err
	}

	if m := b["subnet"].(map[string]interface{}); m["gateway_ip"] == "" {
		m["gateway_ip"] = nil
	}

	return b, nil
}

// ZoneCreateOpts represents the attributes used when creating a new DNS zone.
type ZoneCreateOpts struct {
	zones.CreateOpts
	ValueSpecs map[string]string `json:"value_specs,omitempty"`
}

// ToZoneCreateMap casts a CreateOpts struct to a map.
// It overrides zones.ToZoneCreateMap to add the ValueSpecs field.
func (opts ZoneCreateOpts) ToZoneCreateMap() (map[string]interface{}, error) {
	b, err := BuildRequest(opts, "")
	if err != nil {
		return nil, err
	}

	if m, ok := b[""].(map[string]interface{}); ok {
		if opts.TTL > 0 {
			m["ttl"] = opts.TTL
		}

		return m, nil
	}

	return nil, fmt.Errorf("Expected map but got %T", b[""])
}
