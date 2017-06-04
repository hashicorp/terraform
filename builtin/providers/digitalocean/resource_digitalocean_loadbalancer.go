package digitalocean

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/digitalocean/godo"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDigitalOceanLoadbalancer() *schema.Resource {
	return &schema.Resource{
		Create: resourceDigitalOceanLoadbalancerCreate,
		Read:   resourceDigitalOceanLoadbalancerRead,
		Update: resourceDigitalOceanLoadbalancerUpdate,
		Delete: resourceDigitalOceanLoadbalancerDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"region": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"algorithm": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "round_robin",
			},

			"forwarding_rule": {
				Type:     schema.TypeList,
				Required: true,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"entry_protocol": {
							Type:     schema.TypeString,
							Required: true,
						},
						"entry_port": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"target_protocol": {
							Type:     schema.TypeString,
							Required: true,
						},
						"target_port": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"certificate_id": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"tls_passthrough": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
					},
				},
			},

			"healthcheck": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"protocol": {
							Type:     schema.TypeString,
							Required: true,
						},
						"port": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"path": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"check_interval_seconds": {
							Type:     schema.TypeInt,
							Optional: true,
							Default:  10,
						},
						"response_timeout_seconds": {
							Type:     schema.TypeInt,
							Optional: true,
							Default:  5,
						},
						"unhealthy_threshold": {
							Type:     schema.TypeInt,
							Optional: true,
							Default:  3,
						},
						"healthy_threshold": {
							Type:     schema.TypeInt,
							Optional: true,
							Default:  5,
						},
					},
				},
			},

			"sticky_sessions": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true, //this needs to be computed as the API returns a struct with none as the type
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "none",
						},
						"cookie_name": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"cookie_ttl_seconds": {
							Type:     schema.TypeInt,
							Optional: true,
						},
					},
				},
			},

			"droplet_ids": {
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
			},

			"droplet_tag": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"redirect_http_to_https": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"ip": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func buildLoadBalancerRequest(d *schema.ResourceData) (*godo.LoadBalancerRequest, error) {
	opts := &godo.LoadBalancerRequest{
		Name:                d.Get("name").(string),
		Region:              d.Get("region").(string),
		Algorithm:           d.Get("algorithm").(string),
		RedirectHttpToHttps: d.Get("redirect_http_to_https").(bool),
		ForwardingRules:     expandForwardingRules(d.Get("forwarding_rule").([]interface{})),
	}

	if v, ok := d.GetOk("droplet_ids"); ok {
		var droplets []int
		for _, id := range v.([]interface{}) {
			i, err := strconv.Atoi(id.(string))
			if err != nil {
				return nil, err
			}
			droplets = append(droplets, i)
		}

		opts.DropletIDs = droplets
	}

	if v, ok := d.GetOk("droplet_tag"); ok {
		opts.Tag = v.(string)
	}

	if v, ok := d.GetOk("healthcheck"); ok {
		opts.HealthCheck = expandHealthCheck(v.([]interface{}))
	}

	if v, ok := d.GetOk("sticky_sessions"); ok {
		opts.StickySessions = expandStickySessions(v.([]interface{}))
	}

	return opts, nil
}

func resourceDigitalOceanLoadbalancerCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*godo.Client)

	log.Printf("[INFO] Create a Loadbalancer Request")

	lbOpts, err := buildLoadBalancerRequest(d)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Loadbalancer Create: %#v", lbOpts)
	loadbalancer, _, err := client.LoadBalancers.Create(context.Background(), lbOpts)
	if err != nil {
		return fmt.Errorf("Error creating Load Balancer: %s", err)
	}

	d.SetId(loadbalancer.ID)

	log.Printf("[DEBUG] Waiting for Load Balancer (%s) to become active", d.Get("name"))
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"new"},
		Target:     []string{"active"},
		Refresh:    loadbalancerStateRefreshFunc(client, d.Id()),
		Timeout:    10 * time.Minute,
		MinTimeout: 15 * time.Second,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf("Error waiting for Load Balancer (%s) to become active: %s", d.Get("name"), err)
	}

	return resourceDigitalOceanLoadbalancerRead(d, meta)
}

func resourceDigitalOceanLoadbalancerRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*godo.Client)

	log.Printf("[INFO] Reading the details of the Loadbalancer %s", d.Id())
	loadbalancer, resp, err := client.LoadBalancers.Get(context.Background(), d.Id())
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			log.Printf("[WARN] DigitalOcean Load Balancer (%s) not found", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error retrieving Loadbalancer: %s", err)
	}

	d.Set("name", loadbalancer.Name)
	d.Set("ip", loadbalancer.IP)
	d.Set("algorithm", loadbalancer.Algorithm)
	d.Set("region", loadbalancer.Region.Slug)
	d.Set("redirect_http_to_https", loadbalancer.RedirectHttpToHttps)
	d.Set("droplet_ids", flattenDropletIds(loadbalancer.DropletIDs))
	d.Set("droplet_tag", loadbalancer.Tag)

	if err := d.Set("sticky_sessions", flattenStickySessions(loadbalancer.StickySessions)); err != nil {
		return fmt.Errorf("[DEBUG] Error setting Load Balancer sticky_sessions - error: %#v", err)
	}

	if err := d.Set("healthcheck", flattenHealthChecks(loadbalancer.HealthCheck)); err != nil {
		return fmt.Errorf("[DEBUG] Error setting Load Balancer healthcheck - error: %#v", err)
	}

	if err := d.Set("forwarding_rule", flattenForwardingRules(loadbalancer.ForwardingRules)); err != nil {
		return fmt.Errorf("[DEBUG] Error setting Load Balancer forwarding_rule - error: %#v", err)
	}

	return nil

}

func resourceDigitalOceanLoadbalancerUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*godo.Client)

	lbOpts, err := buildLoadBalancerRequest(d)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Load Balancer Update: %#v", lbOpts)
	_, _, err = client.LoadBalancers.Update(context.Background(), d.Id(), lbOpts)
	if err != nil {
		return fmt.Errorf("Error updating Load Balancer: %s", err)
	}

	return resourceDigitalOceanLoadbalancerRead(d, meta)
}

func resourceDigitalOceanLoadbalancerDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*godo.Client)

	log.Printf("[INFO] Deleting Load Balancer: %s", d.Id())
	_, err := client.LoadBalancers.Delete(context.Background(), d.Id())
	if err != nil {
		return fmt.Errorf("Error deleting Load Balancer: %s", err)
	}

	d.SetId("")
	return nil

}
