package nsone

import (
	"github.com/hashicorp/terraform/helper/schema"

	nsone "gopkg.in/ns1/ns1-go.v2/rest"
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

func permissionsToResourceData(d *schema.ResourceData, permissions nsone.PermissionsMap) {
	d.Set("dns_view_zones", permissions.Dns.ViewZones)
	d.Set("dns_manage_zones", permissions.Dns.ManageZones)
	d.Set("dns_zones_allow_by_default", permissions.Dns.ZonesAllowByDefault)
	d.Set("dns_zones_deny", permissions.Dns.ZonesDeny)
	d.Set("dns_zones_allow", permissions.Dns.ZonesAllow)
	d.Set("data_push_to_datafeeds", permissions.Data.PushToDatafeeds)
	d.Set("data_manage_datasources", permissions.Data.ManageDatasources)
	d.Set("data_manage_datafeeds", permissions.Data.ManageDatafeeds)
	d.Set("account_manage_users", permissions.Account.ManageUsers)
	d.Set("account_manage_payment_methods", permissions.Account.ManagePaymentMethods)
	d.Set("account_manage_plan", permissions.Account.ManagePlan)
	d.Set("account_manage_teams", permissions.Account.ManageTeams)
	d.Set("account_manage_apikeys", permissions.Account.ManageApikeys)
	d.Set("account_manage_account_settings", permissions.Account.ManageAccountSettings)
	d.Set("account_view_activity_log", permissions.Account.ViewActivityLog)
	d.Set("account_view_invoices", permissions.Account.ViewInvoices)
	d.Set("monitoring_manage_lists", permissions.Monitoring.ManageLists)
	d.Set("monitoring_manage_jobs", permissions.Monitoring.ManageJobs)
	d.Set("monitoring_view_jobs", permissions.Monitoring.ViewJobs)
}

func userToResourceData(d *schema.ResourceData, u *nsone.User) error {
	d.SetId(u.Username)
	d.Set("name", u.Name)
	d.Set("email", u.Email)
	d.Set("teams", u.Teams)
	notify := make(map[string]bool)
	notify["billing"] = u.Notify.Billing
	d.Set("notify", notify)
	permissionsToResourceData(d, u.Permissions)
	return nil
}

func resourceDataToUser(u *nsone.User, d *schema.ResourceData) error {
	u.Name = d.Get("name").(string)
	u.Username = d.Get("username").(string)
	u.Email = d.Get("email").(string)
	if v, ok := d.GetOk("teams"); ok {
		teamsRaw := v.([]interface{})
		u.Teams = make([]string, len(teamsRaw))
		for i, team := range teamsRaw {
			u.Teams[i] = team.(string)
		}
	} else {
		u.Teams = make([]string, 0)
	}
	if v, ok := d.GetOk("notify"); ok {
		notifyRaw := v.(map[string]interface{})
		u.Notify.Billing = notifyRaw["billing"].(bool)
	}
	u.Permissions = resourceDataToPermissions(d)
	return nil
}

// UserCreate creates the given user in ns1
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

// UserRead  reads the given users data from ns1
func UserRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	mj, err := client.GetUser(d.Id())
	if err != nil {
		return err
	}
	userToResourceData(d, &mj)
	return nil
}

// UserDelete deletes the given user from ns1
func UserDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	err := client.DeleteUser(d.Id())
	d.SetId("")
	return err
}

// UserUpdate updates the user with given parameters in ns1
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
