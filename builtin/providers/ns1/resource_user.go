package ns1

import (
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	ns1 "gopkg.in/ns1/ns1-go.v2/rest"
	"gopkg.in/ns1/ns1-go.v2/rest/model/account"
)

func userResource() *schema.Resource {
	return &schema.Resource{
		Create: resourceNS1UserCreate,
		Read:   resourceNS1UserRead,
		Update: resourceNS1UserUpdate,
		Delete: resourceNS1UserDelete,
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
			"permissions": permissionsSchema(),
		},
	}
}

func resourceNS1UserCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)

	u := buildNS1User(d)

	log.Printf("[INFO] Creating NS1 user: %s \n", u.Name)

	if _, err := client.Users.Create(u); err != nil {
		return err
	}

	d.SetId(u.Username)

	return resourceNS1UserRead(d, meta)
}

func resourceNS1UserRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)

	log.Printf("[INFO] Reading NS1 user: %s \n", d.Id())

	u, _, err := client.Users.Get(d.Id())
	if err != nil {
		return err
	}

	d.Set("name", u.Name)
	d.Set("email", u.Email)
	d.Set("teams", u.TeamIDs)

	notify := make(map[string]bool)
	notify["billing"] = u.Notify.Billing
	d.Set("notify", notify)

	return nil
}

func resourceNS1UserUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)

	u := buildNS1User(d)

	log.Printf("[INFO] Updating NS1 user: %s \n", u.Name)

	if _, err := client.Users.Update(u); err != nil {
		return err
	}

	return nil
}

func resourceNS1UserDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)

	log.Printf("[INFO] Deleting NS1 user: %s \n", d.Id())

	if _, err := client.Users.Delete(d.Id()); err != nil {
		return err
	}

	d.SetId("")

	return nil
}

func buildNS1User(d *schema.ResourceData) *account.User {
	u := &account.User{
		Name:     d.Get("name").(string),
		Username: d.Get("username").(string),
		Email:    d.Get("email").(string),
	}

	if v, ok := d.GetOk("teams"); ok {
		teamsRaw := v.([]interface{})
		u.TeamIDs = make([]string, len(teamsRaw))
		for i, team := range teamsRaw {
			u.TeamIDs[i] = team.(string)
		}
	} else {
		u.TeamIDs = make([]string, 0)
	}

	if v, ok := d.GetOk("notify"); ok {
		notifyRaw := v.(map[string]interface{})
		u.Notify.Billing = notifyRaw["billing"].(bool)
	}
	// u.Permissions = resourceDataToPermissions(d)
	return u
}

// func permissionsToResourceData(d *schema.ResourceData, permissions account.PermissionsMap) {
// 	d.Set("dns_view_zones", permissions.DNS.ViewZones)
// 	d.Set("dns_manage_zones", permissions.DNS.ManageZones)
// 	d.Set("dns_zones_allow_by_default", permissions.DNS.ZonesAllowByDefault)
// 	d.Set("dns_zones_deny", permissions.DNS.ZonesDeny)
// 	d.Set("dns_zones_allow", permissions.DNS.ZonesAllow)
// 	d.Set("data_push_to_datafeeds", permissions.Data.PushToDatafeeds)
// 	d.Set("data_manage_datasources", permissions.Data.ManageDatasources)
// 	d.Set("data_manage_datafeeds", permissions.Data.ManageDatafeeds)
// 	d.Set("account_manage_users", permissions.Account.ManageUsers)
// 	d.Set("account_manage_payment_methods", permissions.Account.ManagePaymentMethods)
// 	d.Set("account_manage_plan", permissions.Account.ManagePlan)
// 	d.Set("account_manage_teams", permissions.Account.ManageTeams)
// 	d.Set("account_manage_apikeys", permissions.Account.ManageApikeys)
// 	d.Set("account_manage_account_settings", permissions.Account.ManageAccountSettings)
// 	d.Set("account_view_activity_log", permissions.Account.ViewActivityLog)
// 	d.Set("account_view_invoices", permissions.Account.ViewInvoices)
// 	d.Set("monitoring_manage_lists", permissions.Monitoring.ManageLists)
// 	d.Set("monitoring_manage_jobs", permissions.Monitoring.ManageJobs)
// 	d.Set("monitoring_view_jobs", permissions.Monitoring.ViewJobs)
// }
