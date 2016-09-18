package nsone

import (
	"github.com/hashicorp/terraform/helper/schema"

	nsone "gopkg.in/ns1/ns1-go.v2/rest"
	"gopkg.in/ns1/ns1-go.v2/rest/model/account"
)

func apikeyResource() *schema.Resource {
	s := map[string]*schema.Schema{
		"id": &schema.Schema{
			Type:     schema.TypeString,
			Computed: true,
		},
		"name": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
		},
		"key": &schema.Schema{
			Type:     schema.TypeString,
			Computed: true,
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
		Create: ApikeyCreate,
		Read:   ApikeyRead,
		Update: ApikeyUpdate,
		Delete: ApikeyDelete,
	}
}

func apikeyToResourceData(d *schema.ResourceData, k *account.APIKey) error {
	d.SetId(k.ID)
	d.Set("name", k.Name)
	d.Set("key", k.Key)
	d.Set("teams", k.TeamIDs)
	permissionsToResourceData(d, k.Permissions)
	return nil
}

func resourceDataToPermissions(d *schema.ResourceData) account.PermissionsMap {
	var p account.PermissionsMap
	if v, ok := d.GetOk("dns_view_zones"); ok {
		p.DNS.ViewZones = v.(bool)
	}
	if v, ok := d.GetOk("dns_manage_zones"); ok {
		p.DNS.ManageZones = v.(bool)
	}
	if v, ok := d.GetOk("dns_zones_allow_by_default"); ok {
		p.DNS.ZonesAllowByDefault = v.(bool)
	}
	if v, ok := d.GetOk("dns_zones_deny"); ok {
		denyRaw := v.([]interface{})
		p.DNS.ZonesDeny = make([]string, len(denyRaw))
		for i, deny := range denyRaw {
			p.DNS.ZonesDeny[i] = deny.(string)
		}
	} else {
		p.DNS.ZonesDeny = make([]string, 0)
	}
	if v, ok := d.GetOk("dns_zones_allow"); ok {
		allowRaw := v.([]interface{})
		p.DNS.ZonesAllow = make([]string, len(allowRaw))
		for i, allow := range allowRaw {
			p.DNS.ZonesAllow[i] = allow.(string)
		}
	} else {
		p.DNS.ZonesAllow = make([]string, 0)
	}
	if v, ok := d.GetOk("data_push_to_datafeeds"); ok {
		p.Data.PushToDatafeeds = v.(bool)
	}
	if v, ok := d.GetOk("data_manage_datasources"); ok {
		p.Data.ManageDatasources = v.(bool)
	}
	if v, ok := d.GetOk("data_manage_datafeeds"); ok {
		p.Data.ManageDatafeeds = v.(bool)
	}
	if v, ok := d.GetOk("account_manage_users"); ok {
		p.Account.ManageUsers = v.(bool)
	}
	if v, ok := d.GetOk("account_manage_payment_methods"); ok {
		p.Account.ManagePaymentMethods = v.(bool)
	}
	if v, ok := d.GetOk("account_manage_plan"); ok {
		p.Account.ManagePlan = v.(bool)
	}
	if v, ok := d.GetOk("account_manage_teams"); ok {
		p.Account.ManageTeams = v.(bool)
	}
	if v, ok := d.GetOk("account_manage_apikeys"); ok {
		p.Account.ManageApikeys = v.(bool)
	}
	if v, ok := d.GetOk("account_manage_account_settings"); ok {
		p.Account.ManageAccountSettings = v.(bool)
	}
	if v, ok := d.GetOk("account_view_activity_log"); ok {
		p.Account.ViewActivityLog = v.(bool)
	}
	if v, ok := d.GetOk("account_view_invoices"); ok {
		p.Account.ViewInvoices = v.(bool)
	}
	if v, ok := d.GetOk("monitoring_manage_lists"); ok {
		p.Monitoring.ManageLists = v.(bool)
	}
	if v, ok := d.GetOk("monitoring_manage_jobs"); ok {
		p.Monitoring.ManageJobs = v.(bool)
	}
	if v, ok := d.GetOk("monitoring_view_jobs"); ok {
		p.Monitoring.ViewJobs = v.(bool)
	}
	return p
}

func resourceDataToApikey(k *account.APIKey, d *schema.ResourceData) error {
	k.ID = d.Id()
	k.Name = d.Get("name").(string)
	if v, ok := d.GetOk("teams"); ok {
		teamsRaw := v.([]interface{})
		k.TeamIDs = make([]string, len(teamsRaw))
		for i, team := range teamsRaw {
			k.TeamIDs[i] = team.(string)
		}
	} else {
		k.TeamIDs = make([]string, 0)
	}
	k.Permissions = resourceDataToPermissions(d)
	return nil
}

// ApikeyCreate creates ns1 API key
func ApikeyCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.Client)
	k := account.APIKey{}
	if err := resourceDataToApikey(&k, d); err != nil {
		return err
	}
	if _, err := client.APIKeys.Create(&k); err != nil {
		return err
	}
	return apikeyToResourceData(d, &k)
}

// ApikeyRead reads API key from ns1
func ApikeyRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.Client)
	k, _, err := client.APIKeys.Get(d.Id())
	if err != nil {
		return err
	}
	return apikeyToResourceData(d, k)
}

//ApikeyDelete deletes the given ns1 api key
func ApikeyDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.Client)
	_, err := client.APIKeys.Delete(d.Id())
	d.SetId("")
	return err
}

//ApikeyUpdate updates the given api key in ns1
func ApikeyUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.Client)
	k := account.APIKey{
		ID: d.Id(),
	}
	if err := resourceDataToApikey(&k, d); err != nil {
		return err
	}
	if _, err := client.APIKeys.Update(&k); err != nil {
		return err
	}
	return apikeyToResourceData(d, &k)
}
