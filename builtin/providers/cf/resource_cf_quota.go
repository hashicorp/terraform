package cloudfoundry

import (
	"fmt"

	"github.com/hashicorp/terraform/builtin/providers/cf/cfapi"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceQuota() *schema.Resource {

	return &schema.Resource{

		Create: resourceQuotaCreate,
		Read:   resourceQuotaRead,
		Update: resourceQuotaUpdate,
		Delete: resourceQuotaDelete,

		Schema: map[string]*schema.Schema{

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"allow_paid_service_plans": &schema.Schema{
				Type:     schema.TypeBool,
				Required: true,
			},
			"instance_memory": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  -1,
			},
			"total_memory": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},
			"total_app_instances": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  -1,
			},
			"total_routes": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},
			"total_services": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},
			"total_route_ports": &schema.Schema{
				Type:          schema.TypeInt,
				Optional:      true,
				Default:       -1,
				ConflictsWith: []string{"org"},
			},
			"total_private_domains": &schema.Schema{
				Type:          schema.TypeInt,
				Optional:      true,
				Default:       0,
				ConflictsWith: []string{"org"},
			},
			"org": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceQuotaCreate(d *schema.ResourceData, meta interface{}) (err error) {

	session := meta.(*cfapi.Session)
	if session == nil {
		return fmt.Errorf("client is nil")
	}
	qm := session.QuotaManager()

	var id string
	if id, err = qm.CreateQuota(readQuotaResource(d)); err != nil {
		return
	}
	d.SetId(id)
	return
}

func resourceQuotaRead(d *schema.ResourceData, meta interface{}) (err error) {

	session := meta.(*cfapi.Session)
	if session == nil {
		return fmt.Errorf("client is nil")
	}
	qm := session.QuotaManager()

	var quota cfapi.CCQuota
	if quota, err = qm.ReadQuota(d.Id()); err != nil {
		return
	}
	d.Set("name", quota.Name)
	d.Set("org", quota.OrgGUID)
	d.Set("total_app_instances", quota.AppInstanceLimit)
	d.Set("instance_memory", quota.InstanceMemoryLimit)
	d.Set("total_memory", quota.MemoryLimit)
	d.Set("allow_paid_service_plans", quota.NonBasicServicesAllowed)
	d.Set("total_services", quota.TotalServices)
	d.Set("total_routes", quota.TotalRoutes)

	if len(quota.OrgGUID) == 0 {
		d.Set("total_route_ports", quota.TotalReserveredPorts)
		d.Set("total_private_domains", quota.TotalPrivateDomains)
	}
	return
}

func resourceQuotaUpdate(d *schema.ResourceData, meta interface{}) (err error) {

	session := meta.(*cfapi.Session)
	if session == nil {
		return fmt.Errorf("client is nil")
	}
	qm := session.QuotaManager()

	quota := readQuotaResource(d)
	quota.ID = d.Id()
	err = qm.UpdateQuota(quota)
	return
}

func resourceQuotaDelete(d *schema.ResourceData, meta interface{}) (err error) {

	session := meta.(*cfapi.Session)
	if session == nil {
		return fmt.Errorf("client is nil")
	}
	qm := session.QuotaManager()
	err = qm.DeleteQuota(d.Id(), d.Get("org").(string))
	return
}

func readQuotaResource(d *schema.ResourceData) cfapi.CCQuota {

	quota := cfapi.CCQuota{
		Name:                    d.Get("name").(string),
		AppInstanceLimit:        d.Get("total_app_instances").(int),
		AppTaskLimit:            -1,
		InstanceMemoryLimit:     int64(d.Get("instance_memory").(int)),
		MemoryLimit:             int64(d.Get("total_memory").(int)),
		NonBasicServicesAllowed: d.Get("allow_paid_service_plans").(bool),
		TotalServices:           d.Get("total_services").(int),
		TotalServiceKeys:        -1,
		TotalRoutes:             d.Get("total_routes").(int),
		TotalReserveredPorts:    d.Get("total_route_ports").(int),
		TotalPrivateDomains:     d.Get("total_private_domains").(int),
	}
	if v, ok := d.GetOk("org"); ok {
		quota.OrgGUID = v.(string)
	}
	return quota
}
