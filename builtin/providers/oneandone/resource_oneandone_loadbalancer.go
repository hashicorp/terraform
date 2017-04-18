package oneandone

import (
	"fmt"
	"github.com/1and1/oneandone-cloudserver-sdk-go"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"log"
	"strings"
)

func resourceOneandOneLoadbalancer() *schema.Resource {
	return &schema.Resource{
		Create: resourceOneandOneLoadbalancerCreate,
		Read:   resourceOneandOneLoadbalancerRead,
		Update: resourceOneandOneLoadbalancerUpdate,
		Delete: resourceOneandOneLoadbalancerDelete,
		Schema: map[string]*schema.Schema{

			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"method": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateMethod,
			},
			"datacenter": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"persistence": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"persistence_time": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"health_check_test": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"health_check_interval": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"health_check_path": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"health_check_path_parser": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"rules": {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"protocol": {
							Type:     schema.TypeString,
							Required: true,
						},
						"port_balancer": {
							Type:         schema.TypeInt,
							Required:     true,
							ValidateFunc: validation.IntBetween(1, 65535),
						},
						"port_server": {
							Type:         schema.TypeInt,
							Required:     true,
							ValidateFunc: validation.IntBetween(1, 65535),
						},
						"source_ip": {
							Type:     schema.TypeString,
							Required: true,
						},
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
				Required: true,
			},
		},
	}
}

func resourceOneandOneLoadbalancerCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	req := oneandone.LoadBalancerRequest{
		Name:  d.Get("name").(string),
		Rules: getLBRules(d),
	}

	if raw, ok := d.GetOk("description"); ok {
		req.Description = raw.(string)
	}

	if raw, ok := d.GetOk("datacenter"); ok {
		dcs, err := config.API.ListDatacenters()
		if err != nil {
			return fmt.Errorf("An error occured while fetching list of datacenters %s", err)
		}

		decenter := raw.(string)
		for _, dc := range dcs {
			if strings.ToLower(dc.CountryCode) == strings.ToLower(decenter) {
				req.DatacenterId = dc.Id
				break
			}
		}
	}

	if raw, ok := d.GetOk("method"); ok {
		req.Method = raw.(string)
	}

	if raw, ok := d.GetOk("persistence"); ok {
		req.Persistence = oneandone.Bool2Pointer(raw.(bool))
	}
	if raw, ok := d.GetOk("persistence_time"); ok {
		req.PersistenceTime = oneandone.Int2Pointer(raw.(int))
	}

	if raw, ok := d.GetOk("health_check_test"); ok {
		req.HealthCheckTest = raw.(string)
	}
	if raw, ok := d.GetOk("health_check_interval"); ok {
		req.HealthCheckInterval = oneandone.Int2Pointer(raw.(int))
	}
	if raw, ok := d.GetOk("health_check_path"); ok {
		req.HealthCheckPath = raw.(string)
	}
	if raw, ok := d.GetOk("health_check_path_parser"); ok {
		req.HealthCheckPathParser = raw.(string)
	}

	lb_id, lb, err := config.API.CreateLoadBalancer(&req)
	if err != nil {
		return err
	}

	err = config.API.WaitForState(lb, "ACTIVE", 10, config.Retries)
	if err != nil {
		return err
	}

	d.SetId(lb_id)

	return resourceOneandOneLoadbalancerRead(d, meta)
}

func getLBRules(d *schema.ResourceData) []oneandone.LoadBalancerRule {
	var rules []oneandone.LoadBalancerRule

	if raw, ok := d.GetOk("rules"); ok {
		rawRules := raw.([]interface{})
		log.Println("[DEBUG] raw rules:", raw)
		for _, raw := range rawRules {
			rl := raw.(map[string]interface{})
			rule := oneandone.LoadBalancerRule{
				Protocol: rl["protocol"].(string),
			}

			if rl["port_balancer"] != nil {
				rule.PortBalancer = uint16(rl["port_balancer"].(int))
			}
			if rl["port_server"] != nil {
				rule.PortServer = uint16(rl["port_server"].(int))
			}

			if rl["source_ip"] != nil {
				rule.Source = rl["source_ip"].(string)
			}

			rules = append(rules, rule)
		}
	}
	return rules
}

func resourceOneandOneLoadbalancerUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	if d.HasChange("name") || d.HasChange("description") || d.HasChange("method") || d.HasChange("persistence") || d.HasChange("persistence_time") || d.HasChange("health_check_test") || d.HasChange("health_check_interval") {
		lb := oneandone.LoadBalancerRequest{}
		if d.HasChange("name") {
			_, n := d.GetChange("name")
			lb.Name = n.(string)
		}
		if d.HasChange("description") {
			_, n := d.GetChange("description")
			lb.Description = n.(string)
		}
		if d.HasChange("method") {
			_, n := d.GetChange("method")
			lb.Method = (n.(string))
		}
		if d.HasChange("persistence") {
			_, n := d.GetChange("persistence")
			lb.Persistence = oneandone.Bool2Pointer(n.(bool))
		}
		if d.HasChange("persistence_time") {
			_, n := d.GetChange("persistence_time")
			lb.PersistenceTime = oneandone.Int2Pointer(n.(int))
		}
		if d.HasChange("health_check_test") {
			_, n := d.GetChange("health_check_test")
			lb.HealthCheckTest = n.(string)
		}
		if d.HasChange("health_check_path") {
			_, n := d.GetChange("health_check_path")
			lb.HealthCheckPath = n.(string)
		}
		if d.HasChange("health_check_path_parser") {
			_, n := d.GetChange("health_check_path_parser")
			lb.HealthCheckPathParser = n.(string)
		}

		ss, err := config.API.UpdateLoadBalancer(d.Id(), &lb)

		if err != nil {
			return err
		}
		err = config.API.WaitForState(ss, "ACTIVE", 10, 30)
		if err != nil {
			return err
		}

	}

	if d.HasChange("rules") {
		oldR, newR := d.GetChange("rules")
		oldValues := oldR.([]interface{})
		newValues := newR.([]interface{})
		if len(oldValues) > len(newValues) {
			diff := difference(oldValues, newValues)
			for _, old := range diff {
				o := old.(map[string]interface{})
				if o["id"] != nil {
					old_id := o["id"].(string)
					fw, err := config.API.DeleteLoadBalancerRule(d.Id(), old_id)
					if err != nil {
						return err
					}

					err = config.API.WaitForState(fw, "ACTIVE", 10, config.Retries)
					if err != nil {
						return err
					}
				}
			}
		} else {
			var rules []oneandone.LoadBalancerRule
			log.Println("[DEBUG] new values:", newValues)

			for _, raw := range newValues {
				rl := raw.(map[string]interface{})
				log.Println("[DEBUG] rl:", rl)

				if rl["id"].(string) == "" {
					rule := oneandone.LoadBalancerRule{
						Protocol: rl["protocol"].(string),
					}

					rule.PortServer = uint16(rl["port_server"].(int))
					rule.PortBalancer = uint16(rl["port_balancer"].(int))

					rule.Source = rl["source_ip"].(string)

					log.Println("[DEBUG] adding to list", rl["protocol"], rl["source_ip"], rl["port_balancer"], rl["port_server"])
					log.Println("[DEBUG] adding to list", rule)

					rules = append(rules, rule)
				}
			}

			log.Println("[DEBUG] new rules:", rules)

			if len(rules) > 0 {
				fw, err := config.API.AddLoadBalancerRules(d.Id(), rules)
				if err != nil {
					return err
				}

				err = config.API.WaitForState(fw, "ACTIVE", 10, config.Retries)
			}
		}
	}

	return resourceOneandOneLoadbalancerRead(d, meta)
}

func resourceOneandOneLoadbalancerRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	ss, err := config.API.GetLoadBalancer(d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("name", ss.Name)
	d.Set("description", ss.Description)
	d.Set("datacenter", ss.Datacenter.CountryCode)
	d.Set("method", ss.Method)
	d.Set("persistence", ss.Persistence)
	d.Set("persistence_time", ss.PersistenceTime)
	d.Set("health_check_test", ss.HealthCheckTest)
	d.Set("health_check_interval", ss.HealthCheckInterval)
	d.Set("rules", getLoadbalancerRules(ss.Rules))

	return nil
}

func getLoadbalancerRules(rules []oneandone.LoadBalancerRule) []map[string]interface{} {
	raw := make([]map[string]interface{}, 0, len(rules))

	for _, rule := range rules {

		toadd := map[string]interface{}{
			"id":            rule.Id,
			"port_balancer": rule.PortBalancer,
			"port_server":   rule.PortServer,
			"protocol":      rule.Protocol,
			"source_ip":     rule.Source,
		}

		raw = append(raw, toadd)
	}

	return raw

}

func resourceOneandOneLoadbalancerDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	lb, err := config.API.DeleteLoadBalancer(d.Id())
	if err != nil {
		return err
	}
	err = config.API.WaitUntilDeleted(lb)
	if err != nil {
		return err
	}

	return nil
}

func validateMethod(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)

	if value != "ROUND_ROBIN" && value != "LEAST_CONNECTIONS" {
		errors = append(errors, fmt.Errorf("%q value sholud be either 'ROUND_ROBIN' or 'LEAST_CONNECTIONS' not %q", k, value))
	}

	return
}
