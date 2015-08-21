package nsone

import (
	"github.com/bobtfish/go-nsone-api"
	"github.com/hashicorp/terraform/helper/schema"
)

func addPermsSchema(s map[string]*schema.Schema) map[string]*schema.Schema {
	s["dns_view_zones"] = &schema.Schema{
		Type:     schema.TypeBool,
		Optional: true,
	}
	s["dns_manage_zones"] = &schema.Schema{
		Type:     schema.TypeBool,
		Optional: true,
	}
	s["dns_zones_allow_by_default"] = &schema.Schema{
		Type:     schema.TypeBool,
		Optional: true,
	}
	s["dns_zones_deny"] = &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem:     &schema.Schema{Type: schema.TypeString},
	}
	s["dns_zones_allow"] = &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem:     &schema.Schema{Type: schema.TypeString},
	}
	s["data_push_to_datafeeds"] = &schema.Schema{
		Type:     schema.TypeBool,
		Optional: true,
	}
	s["data_manage_datasources"] = &schema.Schema{
		Type:     schema.TypeBool,
		Optional: true,
	}
	s["data_manage_datafeeds"] = &schema.Schema{
		Type:     schema.TypeBool,
		Optional: true,
	}
	s["account_manage_users"] = &schema.Schema{
		Type:     schema.TypeBool,
		Optional: true,
	}
	s["account_manage_payment_methods"] = &schema.Schema{
		Type:     schema.TypeBool,
		Optional: true,
	}
	s["account_manage_plan"] = &schema.Schema{
		Type:     schema.TypeBool,
		Optional: true,
	}
	s["account_manage_teams"] = &schema.Schema{
		Type:     schema.TypeBool,
		Optional: true,
	}
	s["account_manage_apikeys"] = &schema.Schema{
		Type:     schema.TypeBool,
		Optional: true,
	}
	s["account_manage_account_settings"] = &schema.Schema{
		Type:     schema.TypeBool,
		Optional: true,
	}
	s["account_view_activity_log"] = &schema.Schema{
		Type:     schema.TypeBool,
		Optional: true,
	}
	s["account_view_invoices"] = &schema.Schema{
		Type:     schema.TypeBool,
		Optional: true,
	}
	s["monitoring_manage_lists"] = &schema.Schema{
		Type:     schema.TypeBool,
		Optional: true,
	}
	s["monitoring_manage_jobs"] = &schema.Schema{
		Type:     schema.TypeBool,
		Optional: true,
	}
	s["monitoring_view_jobs"] = &schema.Schema{
		Type:     schema.TypeBool,
		Optional: true,
	}
	return s
}

func userResource() *schema.Resource {
	s := map[string]*schema.Schema{
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
		"notify": &schema.Schema{
			Type:     schema.TypeMap,
			Optional: true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"billing": &schema.Schema{
						Type:     schema.TypeBool,
						Required: true,
					},
				},
			},
		},
		"teams": &schema.Schema{
			Type:     schema.TypeList,
			Optional: true,
			Elem:     &schema.Schema{Type: schema.TypeString},
		},
	}
	s = addPermsSchema(s)
	return &schema.Resource{
		Schema: s,
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
	notify := make(map[string]bool)
	notify["billing"] = u.Notify.Billing
	d.Set("notify", notify)
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

func resourceDataToUser(u *nsone.User, d *schema.ResourceData) error {
	u.Name = d.Get("name").(string)
	u.Username = d.Get("username").(string)
	u.Email = d.Get("email").(string)
	if v, ok := d.GetOk("teams"); ok {
		teams_raw := v.([]interface{})
		u.Teams = make([]string, len(teams_raw))
		for i, team := range teams_raw {
			u.Teams[i] = team.(string)
		}
	} else {
		u.Teams = make([]string, 0)
	}
	if v, ok := d.GetOk("notify"); ok {
		notify_raw := v.(map[string]interface{})
		u.Notify.Billing = notify_raw["billing"].(bool)
	}
	u.Permissions = resourceDataToPermissions(d)
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
