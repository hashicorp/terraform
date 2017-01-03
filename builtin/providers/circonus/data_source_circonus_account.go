package circonus

import (
	"github.com/circonus-labs/circonus-gometrics/api"
	"github.com/hashicorp/terraform/helper/schema"
)

const (
	accountAddress1Attr      = "address1"
	accountAddress2Attr      = "address2"
	accountCCEmailAttr       = "cc_email"
	accountCIDAttr           = "cid"
	accountCityAttr          = "city"
	accountContactGroupsAttr = "contact_groups"
	accountCountryAttr       = "country"
	accountCurrentAttr       = "current"
	accountDescriptionAttr   = "description"
	accountEmailAttr         = "email"
	accountIDAttr            = "id"
	accountInvitesAttr       = "invites"
	accountLimitAttr         = "limit"
	accountNameAttr          = "name"
	accountOwnerAttr         = "owner"
	accountRoleAttr          = "role"
	accountStateProvAttr     = "state"
	accountTimezoneAttr      = "timezone"
	accountTypeAttr          = "type"
	accountUIBaseURLAttr     = "ui_base_url"
	accountUsageAttr         = "usage"
	accountUsedAttr          = "used"
	accountUsersAttr         = "users"
)

func dataSourceCirconusAccount() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceCirconusAccountRead,

		Schema: map[string]*schema.Schema{
			accountAddress1Attr: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			accountAddress2Attr: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			accountCCEmailAttr: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			accountCIDAttr: &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ConflictsWith: []string{accountCurrentAttr},
			},
			accountCityAttr: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			accountContactGroupsAttr: &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				Description: "Contact Groups in this account",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			accountCountryAttr: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			accountCurrentAttr: &schema.Schema{
				Type:          schema.TypeBool,
				Optional:      true,
				Computed:      true,
				ConflictsWith: []string{accountCIDAttr},
			},
			accountDescriptionAttr: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			accountInvitesAttr: &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				Description: "Outstanding invites attached to the account",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						accountEmailAttr: &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
						accountRoleAttr: &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
					},
				},
			},
			accountNameAttr: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			accountOwnerAttr: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			accountStateProvAttr: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			accountTimezoneAttr: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			accountUIBaseURLAttr: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			accountUsageAttr: &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				Description: "Account's usage limits",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						accountLimitAttr: &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
						},
						accountTypeAttr: &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
						accountUsedAttr: &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
						},
					},
				},
			},
			accountUsersAttr: &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				Description: "Users attached to this account",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						accountIDAttr: &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
						accountRoleAttr: &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceCirconusAccountRead(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*api.API)

	var cid string

	var a *api.Account
	var err error
	if v, ok := d.GetOk(accountCIDAttr); ok {
		cid = v.(string)
	}

	if v, ok := d.GetOk(accountCurrentAttr); ok {
		if v.(bool) {
			cid = ""
		}
	}

	a, err = c.FetchAccount(api.CIDType(&cid))
	if err != nil {
		return err
	}

	invitesList := make([]interface{}, 0, len(a.Invites))
	for i := range a.Invites {
		invitesList = append(invitesList, map[string]interface{}{
			accountEmailAttr: a.Invites[i].Email,
			accountRoleAttr:  a.Invites[i].Role,
		})
	}

	usageList := make([]interface{}, 0, len(a.Usage))
	for i := range a.Usage {
		usageList = append(usageList, map[string]interface{}{
			accountLimitAttr: a.Usage[i].Limit,
			accountTypeAttr:  a.Usage[i].Type,
			accountUsedAttr:  a.Usage[i].Used,
		})
	}

	usersList := make([]interface{}, 0, len(a.Users))
	for i := range a.Users {
		usersList = append(usersList, map[string]interface{}{
			accountIDAttr:   a.Users[i].UserCID,
			accountRoleAttr: a.Users[i].Role,
		})
	}

	d.Set(accountAddress1Attr, a.Address1)
	d.Set(accountAddress2Attr, a.Address2)
	d.Set(accountCCEmailAttr, a.CCEmail)
	d.Set(accountCIDAttr, a.CID)
	d.Set(accountCityAttr, a.City)
	d.Set(accountContactGroupsAttr, a.ContactGroups)
	d.Set(accountCountryAttr, a.Country)
	d.Set(accountDescriptionAttr, a.Description)
	d.Set(accountInvitesAttr, invitesList)
	d.Set(accountNameAttr, a.Name)
	d.Set(accountOwnerAttr, a.OwnerCID)
	d.Set(accountStateProvAttr, a.StateProv)
	d.Set(accountTimezoneAttr, a.Timezone)
	d.Set(accountUIBaseURLAttr, a.UIBaseURL)
	d.Set(accountUsageAttr, usageList)
	d.Set(accountUsersAttr, usersList)

	d.SetId(a.CID)

	return nil
}
