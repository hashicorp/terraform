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
	"github.com/hashicorp/terraform/helper/validation"
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
				ValidateFunc: validation.StringInSlice([]string{"Enabled", "Disabled"}, true),
			},

			"traffic_routing_method": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice([]string{"Performance", "Weighted", "Priority"}, false),
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
							ValidateFunc: validation.IntBetween(30, 999999),
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
							ValidateFunc: validation.StringInSlice([]string{"http", "https"}, false),
						},
						"port": {
							Type:         schema.TypeInt,
							Required:     true,
							ValidateFunc: validation.IntBetween(1, 65535),
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
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				DiffSuppressFunc: resourceAzurermResourceGroupNameDiffSuppress,
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
		Name:              &name,
		Location:          &location,
		ProfileProperties: getArmTrafficManagerProfileProperties(d),
		Tags:              expandTags(tags),
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
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error making Read request on Traffic Manager Profile %s: %s", name, err)
	}

	profile := *resp.ProfileProperties

	// update appropriate values
	d.Set("resource_group_name", resGroup)
	d.Set("name", resp.Name)
	d.Set("profile_status", profile.ProfileStatus)
	d.Set("traffic_routing_method", profile.TrafficRoutingMethod)

	dnsFlat := flattenAzureRMTrafficManagerProfileDNSConfig(profile.DNSConfig)
	d.Set("dns_config", schema.NewSet(resourceAzureRMTrafficManagerDNSConfigHash, dnsFlat))

	// fqdn is actually inside DNSConfig, inlined for simpler reference
	d.Set("fqdn", profile.DNSConfig.Fqdn)

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
