package nsone

import (
	"github.com/bobtfish/go-nsone-api"
	"github.com/hashicorp/terraform/helper/schema"
)

func userResource() *schema.Resource {
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
			"username": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"email": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"teams": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"dns_viewzones": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"dns_managezones": &schema.Schema{
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
		Create: UserCreate,
		Read:   UserRead,
		Update: UserUpdate,
		Delete: UserDelete,
	}
}

func userToResourceData(d *schema.ResourceData, u *nsone.User) error {
	d.SetId(u.Username)
	d.Set("name", u.Name)
	d.Set("email", u.Email)
	d.Set("teams", u.Teams)
	d.Set("dns_viewzones", u.Permissions.Dns.ViewZones)
	d.Set("dns_managezones", u.Permissions.Dns.ManageZones)
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

func resourceDataToUser(u *nsone.User, d *schema.ResourceData) error {
	u.Name = d.Get("name").(string)
	u.Username = d.Get("username").(string)
	u.Email = d.Get("email").(string)
	if v, ok := f.GetOk("teams"); ok {
		teams_raw := v.([]interface{})
		u.Teams = make([]string, len(teams_raw))
		for i, team := range teams_raw {
			u.Teams[i] = team.(string)
		}
	} else {
		u.Teams = make([]string, 0)
	}
	if v, ok := d.GetOk("dns_viewzones"); ok {
		u.Permissions.Dns.ViewZones = v.(bool)
	}
	if v, ok := d.GetOk("dns_managezones"); ok {
		u.Permissions.Dns.ManageZones = v.(bool)
	}
	if v, ok := d.GetOk("dns_zones_allow_by_default"); ok {
		u.Permissions.Dns.ZonesAllowByDefault = v.(bool)
	}
	/*	if v, ok := d.GetOk("dns_zones_deny"); ok {
			d.Set("dns_zones_deny", u.Permissions.Dns.ZonesDeny)
		}
		if v, ok := d.GetOk("dns_zones_allow"); ok {
			d.Set("dns_zones_allow", u.Permissions.Dns.ZonesAllow)
		} */
	if v, ok := d.GetOk("data_push_to_datafeeds"); ok {
		u.Permissions.Data.PushToDatafeeds = v.(bool)
	}
	if v, ok := d.GetOk("data_manage_datasources"); ok {
		u.Permissions.Data.ManageDatasources = v.(bool)
	}
	if v, ok := d.GetOk("data_manage_datafeeds"); ok {
		u.Permissions.Data.ManageDatafeeds = v.(bool)
	}
	if v, ok := d.GetOk("account_manage_users"); ok {
		u.Permissions.Account.ManageUsers = v.(bool)
	}
	if v, ok := d.GetOk("account_manage_payment_methods"); ok {
		u.Permissions.Account.ManagePaymentMethods = v.(bool)
	}
	if v, ok := d.GetOk("account_manage_plan"); ok {
		u.Permissions.Account.ManagePlan = v.(bool)
	}
	if v, ok := d.GetOk("account_manage_teams"); ok {
		u.Permissions.Account.ManageTeams = v.(bool)
	}
	if v, ok := d.GetOk("account_manage_apikeys"); ok {
		u.Permissions.Account.ManageApikeys = v.(bool)
	}
	if v, ok := d.GetOk("account_manage_account_settings"); ok {
		u.Permissions.Account.ManageAccountSettings = v.(bool)
	}
	if v, ok := d.GetOk("account_view_activity_log"); ok {
		u.Permissions.Account.ViewActivityLog = v.(bool)
	}
	if v, ok := d.GetOk("account_view_invoices"); ok {
		u.Permissions.Account.ViewInvoices = v.(bool)
	}
	if v, ok := d.GetOk("monitoring_manage_lists"); ok {
		u.Permissions.Monitoring.ManageLists = v.(bool)
	}
	if v, ok := d.GetOk("monitoring_manage_jobs"); ok {
		u.Permissions.Monitoring.ManageJobs = v.(bool)
	}
	if v, ok := d.GetOk("monitoring_view_jobs"); ok {
		u.Permissions.Monitoring.ViewJobs = v.(bool)
	}
	return nil
}

func UserCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	mj := nsone.User{}
	if err := resourceDataToUser(&mj, d); err != nil {
		return err
	}
	if err := client.CreateUser(&mj); err != nil {
		return err
	}
	return userToResourceData(d, &mj)
}

func UserRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	mj, err := client.GetUser(d.Id())
	if err != nil {
		return err
	}
	userToResourceData(d, &mj)
	return nil
}

func UserDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	err := client.DeleteUser(d.Id())
	d.SetId("")
	return err
}

func UserUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	mj := nsone.User{
		Username: d.Id(),
	}
	if err := resourceDataToUser(&mj, d); err != nil {
		return err
	}
	if err := client.UpdateUser(&mj); err != nil {
		return err
	}
	userToResourceData(d, &mj)
	return nil
}
