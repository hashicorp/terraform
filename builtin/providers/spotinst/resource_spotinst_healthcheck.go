package spotinst

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/spotinst/spotinst-sdk-go/spotinst"
)

func resourceSpotinstHealthCheck() *schema.Resource {
	return &schema.Resource{
		Create: resourceSpotinstHealthCheckCreate,
		Update: resourceSpotinstHealthCheckUpdate,
		Read:   resourceSpotinstHealthCheckRead,
		Delete: resourceSpotinstHealthCheckDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"resource_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"check": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"protocol": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"endpoint": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"port": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},

						"interval": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},

						"timeout": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
					},
				},
			},

			"threshold": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"healthy": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},

						"unhealthy": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
					},
				},
			},

			"proxy": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"addr": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"port": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func resourceSpotinstHealthCheckCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*spotinst.Client)
	newHealthCheck, err := buildHealthCheckOpts(d, meta)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] HealthCheck create configuration: %#v\n", newHealthCheck)
	res, _, err := client.HealthCheck.Create(newHealthCheck)
	if err != nil || len(res) == 0 {
		return fmt.Errorf("[ERROR] Error creating health check: %s", err)
	}
	d.SetId(*res[0].ID)
	log.Printf("[INFO] HealthCheck created successfully: %s\n", d.Id())
	return resourceSpotinstHealthCheckRead(d, meta)
}

func resourceSpotinstHealthCheckRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*spotinst.Client)
	healthChecks, _, err := client.HealthCheck.Get(d.Id())
	if err != nil {
		if serr, ok := err.(*spotinst.ErrorResponse); ok {
			if serr.Response.StatusCode == 400 {
				d.SetId("")
				return nil
			} else {
				return fmt.Errorf("[ERROR] Error retrieving health check: %s", err)
			}
		} else {
			return fmt.Errorf("[ERROR] Error retrieving health check: %s", err)
		}
	}
	if len(healthChecks) == 0 {
		//return fmt.Errorf("[ERROR] No matching health check %s", d.Id())
		d.SetId("")
		return nil
	} else if len(healthChecks) > 1 {
		return fmt.Errorf("[ERROR] Got %d results, only one is allowed", len(healthChecks))
	} else if hc := healthChecks[0]; hc != nil {
		d.Set("name", hc.Name)
		d.Set("resource_id", hc.ResourceID)

		// Set the check.
		check := make([]map[string]interface{}, 0, 1)
		check = append(check, map[string]interface{}{
			"protocol": hc.Check.Protocol,
			"endpoint": hc.Check.Endpoint,
			"port":     hc.Check.Port,
			"interval": hc.Check.Interval,
			"timeout":  hc.Check.Timeout,
		})
		d.Set("check", check)

		// Set the threshold.
		threshold := make([]map[string]interface{}, 0, 1)
		threshold = append(threshold, map[string]interface{}{
			"healthy":   hc.Check.Healthy,
			"unhealthy": hc.Check.Unhealthy,
		})
		d.Set("threshold", threshold)

		// Set the proxy.
		proxy := make([]map[string]interface{}, 0, 1)
		proxy = append(proxy, map[string]interface{}{
			"addr": hc.Addr,
			"port": hc.Port,
		})
		d.Set("proxy", proxy)
	} else {
		d.SetId("")
		return nil
	}
	return nil
}

func resourceSpotinstHealthCheckUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*spotinst.Client)
	healthCheck := &spotinst.HealthCheck{ID: spotinst.String(d.Id())}
	hasChange := false

	if d.HasChange("name") {
		healthCheck.Name = spotinst.String(d.Get("name").(string))
		hasChange = true
	}

	if d.HasChange("resource_id") {
		healthCheck.ResourceID = spotinst.String(d.Get("resource_id").(string))
		hasChange = true
	}

	if d.HasChange("check") {
		if v, ok := d.GetOk("check"); ok {
			if check, err := expandHealthCheckConfig(v); err != nil {
				return err
			} else {
				healthCheck.Check = check
				hasChange = true
			}
		}
	}

	if d.HasChange("threshold") {
		if v, ok := d.GetOk("threshold"); ok {
			if threshold, err := expandHealthCheckThreshold(v); err != nil {
				return err
			} else {
				healthCheck.Check.HealthCheckThreshold = threshold
				hasChange = true
			}
		}
	}

	if d.HasChange("proxy") {
		if v, ok := d.GetOk("proxy"); ok {
			if proxy, err := expandHealthCheckProxy(v); err != nil {
				return err
			} else {
				healthCheck.HealthCheckProxy = proxy
				hasChange = true
			}
		}
	}

	if hasChange {
		log.Printf("[DEBUG] HealthCheck update configuration: %#v\n", healthCheck)
		_, _, err := client.HealthCheck.Update(healthCheck)
		if err != nil {
			return fmt.Errorf("[ERROR] Error updating health check: %s", err)
		}
	}

	return resourceSpotinstHealthCheckRead(d, meta)
}

func resourceSpotinstHealthCheckDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*spotinst.Client)
	log.Printf("[INFO] Deleting health check: %s\n", d.Id())
	healthCheck := &spotinst.HealthCheck{ID: spotinst.String(d.Id())}
	_, err := client.HealthCheck.Delete(healthCheck)
	if err != nil {
		return fmt.Errorf("[ERROR] Error deleting health check: %s", err)
	}
	return nil
}

// buildHealthCheckOpts builds the Spotinst HealthCheck options.
func buildHealthCheckOpts(d *schema.ResourceData, meta interface{}) (*spotinst.HealthCheck, error) {
	healthCheck := &spotinst.HealthCheck{
		Name:       spotinst.String(d.Get("name").(string)),
		ResourceID: spotinst.String(d.Get("resource_id").(string)),
	}

	if v, ok := d.GetOk("check"); ok {
		if check, err := expandHealthCheckConfig(v); err != nil {
			return nil, err
		} else {
			healthCheck.Check = check
		}
	}

	if v, ok := d.GetOk("threshold"); ok {
		if threshold, err := expandHealthCheckThreshold(v); err != nil {
			return nil, err
		} else {
			healthCheck.Check.HealthCheckThreshold = threshold
		}
	}

	if v, ok := d.GetOk("proxy"); ok {
		if proxy, err := expandHealthCheckProxy(v); err != nil {
			return nil, err
		} else {
			healthCheck.HealthCheckProxy = proxy
		}
	}

	return healthCheck, nil
}

// expandHealthCheckConfig expands the Check block.
func expandHealthCheckConfig(data interface{}) (*spotinst.HealthCheckConfig, error) {
	if list := data.(*schema.Set).List(); len(list) != 1 {
		return nil, fmt.Errorf("Only a single check block is expected")
	} else {
		m := list[0].(map[string]interface{})
		check := &spotinst.HealthCheckConfig{}

		if v, ok := m["protocol"].(string); ok && v != "" {
			check.Protocol = spotinst.String(v)
		}

		if v, ok := m["endpoint"].(string); ok && v != "" {
			check.Endpoint = spotinst.String(v)
		}

		if v, ok := m["port"].(int); ok && v >= 0 {
			check.Port = spotinst.Int(v)
		}

		if v, ok := m["interval"].(int); ok && v >= 0 {
			check.Interval = spotinst.Int(v)
		}

		if v, ok := m["timeout"].(int); ok && v >= 0 {
			check.Timeout = spotinst.Int(v)
		}

		log.Printf("[DEBUG] HealthCheck check configuration: %#v\n", check)
		return check, nil
	}
}

// expandHealthCheckThreshold expands the Threshold block.
func expandHealthCheckThreshold(data interface{}) (*spotinst.HealthCheckThreshold, error) {
	if list := data.(*schema.Set).List(); len(list) != 1 {
		return nil, fmt.Errorf("Only a single threshold block is expected")
	} else {
		m := list[0].(map[string]interface{})
		threshold := &spotinst.HealthCheckThreshold{}

		if v, ok := m["healthy"].(int); ok && v >= 0 {
			threshold.Healthy = spotinst.Int(v)
		}

		if v, ok := m["unhealthy"].(int); ok && v >= 0 {
			threshold.Unhealthy = spotinst.Int(v)
		}

		log.Printf("[DEBUG] HealthCheck threshold configuration: %#v\n", threshold)
		return threshold, nil
	}
}

// expandHealthCheckProxy expands the Proxy block.
func expandHealthCheckProxy(data interface{}) (*spotinst.HealthCheckProxy, error) {
	if list := data.(*schema.Set).List(); len(list) != 1 {
		return nil, fmt.Errorf("Only a single proxy block is expected")
	} else {
		m := list[0].(map[string]interface{})
		proxy := &spotinst.HealthCheckProxy{}

		if v, ok := m["addr"].(string); ok && v != "" {
			proxy.Addr = spotinst.String(v)
		}

		if v, ok := m["port"].(int); ok && v > 0 {
			proxy.Port = spotinst.Int(v)
		}

		log.Printf("[DEBUG] HealthCheck proxy configuration: %#v\n", proxy)
		return proxy, nil
	}
}
