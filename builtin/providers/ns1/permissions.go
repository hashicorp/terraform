package ns1

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"gopkg.in/ns1/ns1-go.v2/rest/model/account"
)

func permissionsSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		MaxItems: 1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"dns":        dnsPermissionsSchema(),
				"data":       dataPermissionsSchema(),
				"account":    accountPermissionsSchema(),
				"monitoring": monitoringPermissionsSchema(),
			},
		},
	}
}

func dnsPermissionsSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		MaxItems: 1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"view_zones": &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
				},
				"manage_zones": &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
				},
				"zones_allow_by_default": &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
				},
				"zones_deny": &schema.Schema{
					Type:     schema.TypeList,
					Optional: true,
					Elem:     &schema.Schema{Type: schema.TypeString},
				},
				"zones_allow": &schema.Schema{
					Type:     schema.TypeList,
					Optional: true,
					Elem:     &schema.Schema{Type: schema.TypeString},
				},
			},
		},
	}
}

func dataPermissionsSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		MaxItems: 1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"push_to_datafeeds": &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
				},
				"manage_datasources": &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
				},
				"manage_datafeeds": &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
				},
			},
		},
	}
}

func accountPermissionsSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		MaxItems: 1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"manage_users": &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
				},
				"manage_payment_methods": &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
				},
				"manage_plan": &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
				},
				"manage_teams": &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
				},
				"manage_apikeys": &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
				},
				"manage_account_settings": &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
				},
				"view_activity_log": &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
				},
				"view_invoices": &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
				},
			},
		},
	}
}

func monitoringPermissionsSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		MaxItems: 1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"manage_lists": &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
				},
				"manage_jobs": &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
				},
				"view_jobs": &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
				},
			},
		},
	}
}

func flattenNS1Permissions(p account.PermissionsMap) []map[string]interface{} {
	m := setMap(make(map[string]interface{}))

	m.Set("dns", flattenNS1PermsDNS(p.DNS))
	m.Set("data", flattenNS1PermsData(p.Data))
	m.Set("account", flattenNS1PermsAccount(p.Account))
	m.Set("monitoring", flattenNS1PermsMonitoring(p.Monitoring))

	return m.MapList()
}

func flattenNS1PermsDNS(p account.PermissionsDNS) []map[string]interface{} {
	m := setMap(make(map[string]interface{}))

	m.Set("view_zones", p.ViewZones)
	m.Set("manage_zones", p.ManageZones)
	m.Set("zones_allow_by_default", p.ZonesAllowByDefault)
	m.Set("zones_deny", p.ZonesDeny)
	m.Set("zones_allow", p.ZonesAllow)

	return m.MapList()
}

func flattenNS1PermsData(p account.PermissionsData) []map[string]interface{} {
	m := setMap(make(map[string]interface{}))

	m.Set("push_to_datafeeds", p.PushToDatafeeds)
	m.Set("manage_datafeeds", p.ManageDatafeeds)
	m.Set("manage_datasources", p.ManageDatasources)

	return m.MapList()
}

func flattenNS1PermsAccount(p account.PermissionsAccount) []map[string]interface{} {
	m := setMap(make(map[string]interface{}))

	m.Set("manage_users", p.ManageUsers)
	m.Set("manage_payment_methods", p.ManagePaymentMethods)
	m.Set("manage_plan", p.ManagePlan)
	m.Set("manage_teams", p.ManageTeams)
	m.Set("manage_apikeys", p.ManageApikeys)
	m.Set("manage_account_settings", p.ManageAccountSettings)
	m.Set("view_activity_log", p.ViewActivityLog)
	m.Set("view_invoices", p.ViewInvoices)

	return m.MapList()
}

func flattenNS1PermsMonitoring(p account.PermissionsMonitoring) []map[string]interface{} {
	m := setMap(make(map[string]interface{}))

	m.Set("manage_jobs", p.ManageJobs)
	m.Set("manage_lists", p.ManageLists)
	m.Set("view_jobs", p.ViewJobs)

	return m.MapList()
}

func expandNS1Permissions(d *schema.ResourceData) account.PermissionsMap {
	var p account.PermissionsMap

	expandNS1PermsDNS(d, &p.DNS)
	expandNS1PermsData(d, &p.Data)
	expandNS1PermsAccount(d, &p.Account)
	expandNS1PermsMonitoring(d, &p.Monitoring)

	return p
}

func expandNS1PermsDNS(d *schema.ResourceData, p *account.PermissionsDNS) {
	prefix := "permissions.0.dns.0"

	if v, ok := d.GetOk(prefix + ".view_zones"); ok {
		p.ViewZones = v.(bool)
	}
	if v, ok := d.GetOk(prefix + ".manage_zones"); ok {
		p.ManageZones = v.(bool)
	}
	if v, ok := d.GetOk(prefix + ".zones_allow_by_default"); ok {
		p.ZonesAllowByDefault = v.(bool)
	}

	if l, ok := d.GetOk(prefix + ".zones_deny.#"); ok {
		p.ZonesDeny = make([]string, l.(int))
		for i := 0; i < l.(int); i++ {
			key := fmt.Sprintf("%s.zones_deny.%d", prefix, i)
			p.ZonesDeny[i] = d.Get(key).(string)
		}
	} else {
		p.ZonesDeny = make([]string, 0)
	}
	if l, ok := d.GetOk(prefix + ".zones_allow.#"); ok {
		p.ZonesAllow = make([]string, l.(int))
		for i := 0; i < l.(int); i++ {
			key := fmt.Sprintf("%s.zones_allow.%d", prefix, i)
			p.ZonesAllow[i] = d.Get(key).(string)
		}
	} else {
		p.ZonesAllow = make([]string, 0)
	}
}

func expandNS1PermsData(d *schema.ResourceData, p *account.PermissionsData) {
	prefix := "permissions.0.data.0"

	if v, ok := d.GetOk(prefix + ".push_to_datafeeds"); ok {
		p.PushToDatafeeds = v.(bool)
	}
	if v, ok := d.GetOk(prefix + ".manage_datasources"); ok {
		p.ManageDatasources = v.(bool)
	}
	if v, ok := d.GetOk(prefix + ".manage_datafeeds"); ok {
		p.ManageDatafeeds = v.(bool)
	}
}

func expandNS1PermsAccount(d *schema.ResourceData, p *account.PermissionsAccount) {
	prefix := "permissions.0.account.0"

	if v, ok := d.GetOk(prefix + ".manage_users"); ok {
		p.ManageUsers = v.(bool)
	}
	if v, ok := d.GetOk(prefix + ".manage_payment_methods"); ok {
		p.ManagePaymentMethods = v.(bool)
	}
	if v, ok := d.GetOk(prefix + ".manage_plan"); ok {
		p.ManagePlan = v.(bool)
	}
	if v, ok := d.GetOk(prefix + ".manage_teams"); ok {
		p.ManageTeams = v.(bool)
	}
	if v, ok := d.GetOk(prefix + ".manage_apikeys"); ok {
		p.ManageApikeys = v.(bool)
	}
	if v, ok := d.GetOk(prefix + ".manage_account_settings"); ok {
		p.ManageAccountSettings = v.(bool)
	}
	if v, ok := d.GetOk(prefix + ".view_activity_log"); ok {
		p.ViewActivityLog = v.(bool)
	}
	if v, ok := d.GetOk(prefix + ".view_invoices"); ok {
		p.ViewInvoices = v.(bool)
	}
}

func expandNS1PermsMonitoring(d *schema.ResourceData, p *account.PermissionsMonitoring) {
	prefix := "permissions.0.monitoring.0"

	if v, ok := d.GetOk(prefix + ".manage_lists"); ok {
		p.ManageLists = v.(bool)
	}
	if v, ok := d.GetOk(prefix + ".manage_jobs"); ok {
		p.ManageJobs = v.(bool)
	}
	if v, ok := d.GetOk(prefix + ".view_jobs"); ok {
		p.ViewJobs = v.(bool)
	}
}
