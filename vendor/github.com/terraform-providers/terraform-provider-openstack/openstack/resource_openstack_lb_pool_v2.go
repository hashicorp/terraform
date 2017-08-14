package openstack

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/lbaas_v2/pools"
)

func resourcePoolV2() *schema.Resource {
	return &schema.Resource{
		Create: resourcePoolV2Create,
		Read:   resourcePoolV2Read,
		Update: resourcePoolV2Update,
		Delete: resourcePoolV2Delete,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"tenant_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"protocol": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					if value != "TCP" && value != "HTTP" && value != "HTTPS" {
						errors = append(errors, fmt.Errorf(
							"Only 'TCP', 'HTTP', and 'HTTPS' are supported values for 'protocol'"))
					}
					return
				},
			},

			// One of loadbalancer_id or listener_id must be provided
			"loadbalancer_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			// One of loadbalancer_id or listener_id must be provided
			"listener_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"lb_method": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					if value != "ROUND_ROBIN" && value != "LEAST_CONNECTIONS" && value != "SOURCE_IP" {
						errors = append(errors, fmt.Errorf(
							"Only 'ROUND_ROBIN', 'LEAST_CONNECTIONS', and 'SOURCE_IP' are supported values for 'lb_method'"))
					}
					return
				},
			},

			"persistence": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
							ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
								value := v.(string)
								if value != "SOURCE_IP" && value != "HTTP_COOKIE" && value != "APP_COOKIE" {
									errors = append(errors, fmt.Errorf(
										"Only 'SOURCE_IP', 'HTTP_COOKIE', and 'APP_COOKIE' are supported values for 'persistence'"))
								}
								return
							},
						},

						"cookie_name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
					},
				},
			},

			"admin_state_up": &schema.Schema{
				Type:     schema.TypeBool,
				Default:  true,
				Optional: true,
			},

			"id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourcePoolV2Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	adminStateUp := d.Get("admin_state_up").(bool)
	var persistence pools.SessionPersistence
	if p, ok := d.GetOk("persistence"); ok {
		pV := (p.([]interface{}))[0].(map[string]interface{})

		persistence = pools.SessionPersistence{
			Type: pV["type"].(string),
		}

		if persistence.Type == "APP_COOKIE" {
			if pV["cookie_name"].(string) == "" {
				return fmt.Errorf(
					"Persistence cookie_name needs to be set if using 'APP_COOKIE' persistence type.")
			} else {
				persistence.CookieName = pV["cookie_name"].(string)
			}
		} else {
			if pV["cookie_name"].(string) != "" {
				return fmt.Errorf(
					"Persistence cookie_name can only be set if using 'APP_COOKIE' persistence type.")
			}
		}
	}

	createOpts := pools.CreateOpts{
		TenantID:       d.Get("tenant_id").(string),
		Name:           d.Get("name").(string),
		Description:    d.Get("description").(string),
		Protocol:       pools.Protocol(d.Get("protocol").(string)),
		LoadbalancerID: d.Get("loadbalancer_id").(string),
		ListenerID:     d.Get("listener_id").(string),
		LBMethod:       pools.LBMethod(d.Get("lb_method").(string)),
		AdminStateUp:   &adminStateUp,
	}

	// Must omit if not set
	if persistence != (pools.SessionPersistence{}) {
		createOpts.Persistence = &persistence
	}

	log.Printf("[DEBUG] Create Options: %#v", createOpts)

	// Wait for LoadBalancer to become active before continuing
	timeout := d.Timeout(schema.TimeoutCreate)
	lbID := createOpts.LoadbalancerID
	listenerID := createOpts.ListenerID
	if lbID != "" {
		err = waitForLBV2LoadBalancer(networkingClient, lbID, "ACTIVE", nil, timeout)
		if err != nil {
			return err
		}
	} else if listenerID != "" {
		// Wait for Listener to become active before continuing
		err = waitForLBV2Listener(networkingClient, listenerID, "ACTIVE", nil, timeout)
		if err != nil {
			return err
		}
	}

	log.Printf("[DEBUG] Attempting to create pool")
	var pool *pools.Pool
	err = resource.Retry(timeout, func() *resource.RetryError {
		pool, err = pools.Create(networkingClient, createOpts).Extract()
		if err != nil {
			return checkForRetryableError(err)
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("Error creating pool: %s", err)
	}

	// Wait for LoadBalancer to become active before continuing
	if lbID != "" {
		err = waitForLBV2LoadBalancer(networkingClient, lbID, "ACTIVE", nil, timeout)
	} else {
		// Pool exists by now so we can ask for lbID
		err = waitForLBV2viaPool(networkingClient, pool.ID, "ACTIVE", timeout)
	}
	if err != nil {
		return err
	}

	d.SetId(pool.ID)

	return resourcePoolV2Read(d, meta)
}

func resourcePoolV2Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	pool, err := pools.Get(networkingClient, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "pool")
	}

	log.Printf("[DEBUG] Retrieved pool %s: %#v", d.Id(), pool)

	d.Set("lb_method", pool.LBMethod)
	d.Set("protocol", pool.Protocol)
	d.Set("description", pool.Description)
	d.Set("tenant_id", pool.TenantID)
	d.Set("admin_state_up", pool.AdminStateUp)
	d.Set("name", pool.Name)
	d.Set("id", pool.ID)
	d.Set("persistence", pool.Persistence)
	d.Set("region", GetRegion(d, config))

	return nil
}

func resourcePoolV2Update(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	var updateOpts pools.UpdateOpts
	if d.HasChange("lb_method") {
		updateOpts.LBMethod = pools.LBMethod(d.Get("lb_method").(string))
	}
	if d.HasChange("name") {
		updateOpts.Name = d.Get("name").(string)
	}
	if d.HasChange("description") {
		updateOpts.Description = d.Get("description").(string)
	}
	if d.HasChange("admin_state_up") {
		asu := d.Get("admin_state_up").(bool)
		updateOpts.AdminStateUp = &asu
	}

	// Wait for LoadBalancer to become active before continuing
	timeout := d.Timeout(schema.TimeoutUpdate)
	lbID := d.Get("loadbalancer_id").(string)
	if lbID != "" {
		err = waitForLBV2LoadBalancer(networkingClient, lbID, "ACTIVE", nil, timeout)
	} else {
		err = waitForLBV2viaPool(networkingClient, d.Id(), "ACTIVE", timeout)
	}
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Updating pool %s with options: %#v", d.Id(), updateOpts)
	err = resource.Retry(timeout, func() *resource.RetryError {
		_, err = pools.Update(networkingClient, d.Id(), updateOpts).Extract()
		if err != nil {
			return checkForRetryableError(err)
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("Unable to update pool %s: %s", d.Id(), err)
	}

	// Wait for LoadBalancer to become active before continuing
	if lbID != "" {
		err = waitForLBV2LoadBalancer(networkingClient, lbID, "ACTIVE", nil, timeout)
	} else {
		err = waitForLBV2viaPool(networkingClient, d.Id(), "ACTIVE", timeout)
	}
	if err != nil {
		return err
	}

	return resourcePoolV2Read(d, meta)
}

func resourcePoolV2Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	// Wait for LoadBalancer to become active before continuing
	timeout := d.Timeout(schema.TimeoutDelete)
	lbID := d.Get("loadbalancer_id").(string)
	if lbID != "" {
		err = waitForLBV2LoadBalancer(networkingClient, lbID, "ACTIVE", nil, timeout)
		if err != nil {
			return err
		}
	}

	log.Printf("[DEBUG] Attempting to delete pool %s", d.Id())
	err = resource.Retry(timeout, func() *resource.RetryError {
		err = pools.Delete(networkingClient, d.Id()).ExtractErr()
		if err != nil {
			return checkForRetryableError(err)
		}
		return nil
	})

	if lbID != "" {
		err = waitForLBV2LoadBalancer(networkingClient, lbID, "ACTIVE", nil, timeout)
	} else {
		// Wait for Pool to delete
		err = waitForLBV2Pool(networkingClient, d.Id(), "DELETED", nil, timeout)
	}
	if err != nil {
		return err
	}

	return nil
}
