package azurerm

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/Azure/azure-sdk-for-go/arm/trafficmanager"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceArmTrafficManagerProfile() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmTrafficManagerProfileCreate,
		Read:   resourceArmTrafficManagerProfileRead,
		Update: resourceArmTrafficManagerProfileCreate,
		Delete: resourceArmTrafficManagerProfileDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"profile_status": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validateAzureRMTrafficManagerStatus,
			},

			"traffic_routing_method": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateAzureRMTrafficManagerRoutingMethod,
			},

			"dns_config": {
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"relative_name": {
							Type:     schema.TypeString,
							ForceNew: true,
							Required: true,
						},
						"ttl": {
							Type:         schema.TypeInt,
							Required:     true,
							ValidateFunc: validateAzureRMTrafficManagerTTL,
						},
					},
				},
				Set: resourceAzureRMTrafficManagerDNSConfigHash,
			},

			// inlined from dns_config for ease of use
			"fqdn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"monitor_config": {
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"protocol": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateAzureRMTrafficManagerMonitorProtocol,
						},
						"port": {
							Type:         schema.TypeInt,
							Required:     true,
							ValidateFunc: validateAzureRMTrafficManagerMonitorPort,
						},
						"path": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
				Set: resourceAzureRMTrafficManagerMonitorConfigHash,
			},

			"resource_group_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceArmTrafficManagerProfileCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).trafficManagerProfilesClient

	log.Printf("[INFO] preparing arguments for Azure ARM virtual network creation.")

	name := d.Get("name").(string)
	// must be provided in request
	location := "global"
	resGroup := d.Get("resource_group_name").(string)
	tags := d.Get("tags").(map[string]interface{})

	profile := trafficmanager.Profile{
		Name:       &name,
		Location:   &location,
		Properties: getArmTrafficManagerProfileProperties(d),
		Tags:       expandTags(tags),
	}

	_, err := client.CreateOrUpdate(resGroup, name, profile)
	if err != nil {
		return err
	}

	read, err := client.Get(resGroup, name)
	if err != nil {
		return err
	}
	if read.ID == nil {
		return fmt.Errorf("Cannot read TrafficManager profile %s (resource group %s) ID", name, resGroup)
	}

	d.SetId(*read.ID)

	return resourceArmTrafficManagerProfileRead(d, meta)
}

func resourceArmTrafficManagerProfileRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).trafficManagerProfilesClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["trafficManagerProfiles"]

	resp, err := client.Get(resGroup, name)
	if resp.StatusCode == http.StatusNotFound {
		d.SetId("")
		return nil
	}
	if err != nil {
		return fmt.Errorf("Error making Read request on Traffic Manager Profile %s: %s", name, err)
	}
	profile := *resp.Properties

	// update appropriate values
	d.Set("name", resp.Name)
	d.Set("profile_status", profile.ProfileStatus)
	d.Set("traffic_routing_method", profile.TrafficRoutingMethod)

	dnsFlat := flattenAzureRMTrafficManagerProfileDNSConfig(profile.DNSConfig)
	d.Set("dns_config", schema.NewSet(resourceAzureRMTrafficManagerDNSConfigHash, dnsFlat))

	// fqdn is actually inside DNSConfig, inlined for simpler reference
	if profile.DNSConfig.Fqdn != nil {
		d.Set("fqdn", *profile.DNSConfig.Fqdn)
	}

	monitorFlat := flattenAzureRMTrafficManagerProfileMonitorConfig(profile.MonitorConfig)
	d.Set("monitor_config", schema.NewSet(resourceAzureRMTrafficManagerMonitorConfigHash, monitorFlat))

	flattenAndSetTags(d, resp.Tags)

	return nil
}

func resourceArmTrafficManagerProfileDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).trafficManagerProfilesClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["trafficManagerProfiles"]

	_, err = client.Delete(resGroup, name)

	return err
}

func getArmTrafficManagerProfileProperties(d *schema.ResourceData) *trafficmanager.ProfileProperties {
	routingMethod := d.Get("traffic_routing_method").(string)
	props := &trafficmanager.ProfileProperties{
		TrafficRoutingMethod: &routingMethod,
		DNSConfig:            expandArmTrafficManagerDNSConfig(d),
		MonitorConfig:        expandArmTrafficManagerMonitorConfig(d),
	}

	if status, ok := d.GetOk("profile_status"); ok {
		s := status.(string)
		props.ProfileStatus = &s
	}

	return props
}

func expandArmTrafficManagerMonitorConfig(d *schema.ResourceData) *trafficmanager.MonitorConfig {
	monitorSets := d.Get("monitor_config").(*schema.Set).List()
	monitor := monitorSets[0].(map[string]interface{})

	proto := monitor["protocol"].(string)
	port := int64(monitor["port"].(int))
	path := monitor["path"].(string)

	return &trafficmanager.MonitorConfig{
		Protocol: &proto,
		Port:     &port,
		Path:     &path,
	}
}

func expandArmTrafficManagerDNSConfig(d *schema.ResourceData) *trafficmanager.DNSConfig {
	dnsSets := d.Get("dns_config").(*schema.Set).List()
	dns := dnsSets[0].(map[string]interface{})

	name := dns["relative_name"].(string)
	ttl := int64(dns["ttl"].(int))

	return &trafficmanager.DNSConfig{
		RelativeName: &name,
		TTL:          &ttl,
	}
}

func flattenAzureRMTrafficManagerProfileDNSConfig(dns *trafficmanager.DNSConfig) []interface{} {
	result := make(map[string]interface{})

	result["relative_name"] = *dns.RelativeName
	result["ttl"] = int(*dns.TTL)

	return []interface{}{result}
}

func flattenAzureRMTrafficManagerProfileMonitorConfig(cfg *trafficmanager.MonitorConfig) []interface{} {
	result := make(map[string]interface{})

	result["protocol"] = *cfg.Protocol
	result["port"] = int(*cfg.Port)
	result["path"] = *cfg.Path

	return []interface{}{result}
}

func resourceAzureRMTrafficManagerDNSConfigHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	buf.WriteString(fmt.Sprintf("%s-", m["relative_name"].(string)))
	buf.WriteString(fmt.Sprintf("%d-", m["ttl"].(int)))

	return hashcode.String(buf.String())
}

func resourceAzureRMTrafficManagerMonitorConfigHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	buf.WriteString(fmt.Sprintf("%s-", strings.ToLower(m["protocol"].(string))))
	buf.WriteString(fmt.Sprintf("%d-", m["port"].(int)))
	buf.WriteString(fmt.Sprintf("%s-", m["path"].(string)))

	return hashcode.String(buf.String())
}

func validateAzureRMTrafficManagerStatus(i interface{}, k string) (s []string, errors []error) {
	status := strings.ToLower(i.(string))
	if status != "enabled" && status != "disabled" {
		errors = append(errors, fmt.Errorf("%s must be one of: Enabled, Disabled", k))
	}
	return
}

func validateAzureRMTrafficManagerRoutingMethod(i interface{}, k string) (s []string, errors []error) {
	valid := map[string]struct{}{
		"Performance": struct{}{},
		"Weighted":    struct{}{},
		"Priority":    struct{}{},
	}

	if _, ok := valid[i.(string)]; !ok {
		errors = append(errors, fmt.Errorf("traffic_routing_method must be one of (Performance, Weighted, Priority), got %s", i.(string)))
	}
	return
}

func validateAzureRMTrafficManagerTTL(i interface{}, k string) (s []string, errors []error) {
	ttl := i.(int)
	if ttl < 30 || ttl > 999999 {
		errors = append(errors, fmt.Errorf("ttl must be between 30 and 999,999 inclusive"))
	}
	return
}

func validateAzureRMTrafficManagerMonitorProtocol(i interface{}, k string) (s []string, errors []error) {
	p := i.(string)
	if p != "http" && p != "https" {
		errors = append(errors, fmt.Errorf("monitor_config.protocol must be one of: http, https"))
	}
	return
}

func validateAzureRMTrafficManagerMonitorPort(i interface{}, k string) (s []string, errors []error) {
	p := i.(int)
	if p < 1 || p > 65535 {
		errors = append(errors, fmt.Errorf("monitor_config.port must be between 1 - 65535 inclusive"))
	}
	return
}
