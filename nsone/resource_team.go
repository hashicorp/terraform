package nsone

import (
	"github.com/bobtfish/go-nsone-api"
	"github.com/hashicorp/terraform/helper/schema"
)

func teamResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"dns_view_zones": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"dns_manage_zones": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"dns_zones_allow_by_default": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"dns_zones_deny": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"dns_zones_allow": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"data_push_to_datafeeds": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"data_manage_datasources": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"data_manage_datafeeds": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"account_manage_users": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"account_manage_payment_methods": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"account_manage_plan": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"account_manage_teams": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"account_manage_apikeys": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"account_manage_account_settings": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"account_view_activity_log": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"account_view_invoices": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"monitoring_manage_lists": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"monitoring_manage_jobs": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"monitoring_view_jobs": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
		},
		Create: TeamCreate,
		Read:   TeamRead,
		Update: TeamUpdate,
		Delete: TeamDelete,
	}
}

func teamToResourceData(d *schema.ResourceData, u *nsone.Team) error {
	d.SetId(u.Id)
	d.Set("name", u.Name)
	d.Set("dns_view_zones", u.Permissions.Dns.ViewZones)
	d.Set("dns_manage_zones", u.Permissions.Dns.ManageZones)
	d.Set("dns_zones_allow_by_default", u.Permissions.Dns.ZonesAllowByDefault)
	d.Set("dns_zones_deny", u.Permissions.Dns.ZonesDeny)
	d.Set("dns_zones_allow", u.Permissions.Dns.ZonesAllow)
	d.Set("data_push_to_datafeeds", u.Permissions.Data.PushToDatafeeds)
	d.Set("data_manage_datasources", u.Permissions.Data.ManageDatasources)
	d.Set("data_manage_datafeeds", u.Permissions.Data.ManageDatafeeds)
	d.Set("account_manage_users", u.Permissions.Account.ManageUsers)
	d.Set("account_manage_payment_methods", u.Permissions.Account.ManagePaymentMethods)
	d.Set("account_manage_plan", u.Permissions.Account.ManagePlan)
	d.Set("account_manage_teams", u.Permissions.Account.ManageTeams)
	d.Set("account_manage_apikeys", u.Permissions.Account.ManageApikeys)
	d.Set("account_manage_account_settings", u.Permissions.Account.ManageAccountSettings)
	d.Set("account_view_activity_log", u.Permissions.Account.ViewActivityLog)
	d.Set("account_view_invoices", u.Permissions.Account.ViewInvoices)
	d.Set("monitoring_manage_lists", u.Permissions.Monitoring.ManageLists)
	d.Set("monitoring_manage_jobs", u.Permissions.Monitoring.ManageJobs)
	d.Set("monitoring_view_jobs", u.Permissions.Monitoring.ViewJobs)
	return nil
}

func resourceDataToTeam(u *nsone.Team, d *schema.ResourceData) error {
	u.Id = d.Id()
	u.Name = d.Get("name").(string)
	u.Permissions = resourceDataToPermissions(d)
	return nil
}

func TeamCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	mj := nsone.Team{}
	if err := resourceDataToTeam(&mj, d); err != nil {
		return err
	}
	if err := client.CreateTeam(&mj); err != nil {
		return err
	}
	return teamToResourceData(d, &mj)
}

func TeamRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	mj, err := client.GetTeam(d.Id())
	if err != nil {
		return err
	}
	teamToResourceData(d, &mj)
	return nil
}

func TeamDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	err := client.DeleteTeam(d.Id())
	d.SetId("")
	return err
}

func TeamUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	mj := nsone.Team{
		Id: d.Id(),
	}
	if err := resourceDataToTeam(&mj, d); err != nil {
		return err
	}
	if err := client.UpdateTeam(&mj); err != nil {
		return err
	}
	teamToResourceData(d, &mj)
	return nil
}
