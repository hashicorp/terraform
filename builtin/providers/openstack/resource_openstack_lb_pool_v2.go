package openstack

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/gophercloud/gophercloud"
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
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: schema.EnvDefaultFunc("OS_REGION_NAME", ""),
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
							Required: true,
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
	networkingClient, err := config.networkingV2Client(GetRegion(d))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	adminStateUp := d.Get("admin_state_up").(bool)
	var persistence pools.SessionPersistence
	if p, ok := d.GetOk("persistence"); ok {
		pV := (p.([]interface{}))[0].(map[string]interface{})

		persistence = pools.SessionPersistence{
			Type:       pV["type"].(string),
			CookieName: pV["cookie_name"].(string),
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

	var pool *pools.Pool
	err = resource.Retry(d.Timeout(schema.TimeoutCreate), func() *resource.RetryError {
		var err error
		log.Printf("[DEBUG] Attempting to create LBaaSV2 pool")
		pool, err = pools.Create(networkingClient, createOpts).Extract()
		if err != nil {
			switch errCode := err.(type) {
			case gophercloud.ErrDefault500:
				log.Printf("[DEBUG] OpenStack LBaaSV2 pool is still creating.")
				return resource.RetryableError(err)
			case gophercloud.ErrUnexpectedResponseCode:
				if errCode.Actual == 409 {
					log.Printf("[DEBUG] OpenStack LBaaSV2 pool is still creating.")
					return resource.RetryableError(err)
				}
			default:
				return resource.NonRetryableError(err)
			}
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("Error creating OpenStack LBaaSV2 pool: %s", err)
	}

	log.Printf("[INFO] pool ID: %s", pool.ID)

	log.Printf("[DEBUG] Waiting for Openstack LBaaSV2 pool (%s) to become available.", pool.ID)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"PENDING_CREATE"},
		Target:     []string{"ACTIVE"},
		Refresh:    waitForPoolActive(networkingClient, pool.ID),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return err
	}

	d.SetId(pool.ID)

	return resourcePoolV2Read(d, meta)
}

func resourcePoolV2Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	pool, err := pools.Get(networkingClient, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "LBV2 Pool")
	}

	log.Printf("[DEBUG] Retrieved OpenStack LBaaSV2 Pool %s: %+v", d.Id(), pool)

	d.Set("lb_method", pool.LBMethod)
	d.Set("protocol", pool.Protocol)
	d.Set("description", pool.Description)
	d.Set("tenant_id", pool.TenantID)
	d.Set("admin_state_up", pool.AdminStateUp)
	d.Set("name", pool.Name)
	d.Set("id", pool.ID)
	d.Set("persistence", pool.Persistence)

	return nil
}

func resourcePoolV2Update(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d))
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

	log.Printf("[DEBUG] Updating OpenStack LBaaSV2 Pool %s with options: %+v", d.Id(), updateOpts)

	_, err = pools.Update(networkingClient, d.Id(), updateOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error updating OpenStack LBaaSV2 Pool: %s", err)
	}

	return resourcePoolV2Read(d, meta)
}

func resourcePoolV2Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ACTIVE", "PENDING_DELETE"},
		Target:     []string{"DELETED"},
		Refresh:    waitForPoolDelete(networkingClient, d.Id()),
		Timeout:    d.Timeout(schema.TimeoutDelete),
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error deleting OpenStack LBaaSV2 Pool: %s", err)
	}

	d.SetId("")
	return nil
}

func waitForPoolActive(networkingClient *gophercloud.ServiceClient, poolID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		pool, err := pools.Get(networkingClient, poolID).Extract()
		if err != nil {
			return nil, "", err
		}

		// The pool resource has no Status attribute, so a successful Get is the best we can do
		log.Printf("[DEBUG] OpenStack LBaaSV2 Pool: %+v", pool)
		return pool, "ACTIVE", nil
	}
}

func waitForPoolDelete(networkingClient *gophercloud.ServiceClient, poolID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		log.Printf("[DEBUG] Attempting to delete OpenStack LBaaSV2 Pool %s", poolID)

		pool, err := pools.Get(networkingClient, poolID).Extract()
		if err != nil {
			if _, ok := err.(gophercloud.ErrDefault404); ok {
				log.Printf("[DEBUG] Successfully deleted OpenStack LBaaSV2 Pool %s", poolID)
				return pool, "DELETED", nil
			}
			return pool, "ACTIVE", err
		}

		log.Printf("[DEBUG] Openstack LBaaSV2 Pool: %+v", pool)
		err = pools.Delete(networkingClient, poolID).ExtractErr()
		if err != nil {
			if _, ok := err.(gophercloud.ErrDefault404); ok {
				log.Printf("[DEBUG] Successfully deleted OpenStack LBaaSV2 Pool %s", poolID)
				return pool, "DELETED", nil
			}

			if errCode, ok := err.(gophercloud.ErrUnexpectedResponseCode); ok {
				if errCode.Actual == 409 {
					log.Printf("[DEBUG] OpenStack LBaaSV2 Pool (%s) is still in use.", poolID)
					return pool, "ACTIVE", nil
				}
			}

			return pool, "ACTIVE", err
		}

		log.Printf("[DEBUG] OpenStack LBaaSV2 Pool %s still active.", poolID)
		return pool, "ACTIVE", nil
	}
}
