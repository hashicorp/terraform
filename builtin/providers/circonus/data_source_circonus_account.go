package circonus

import (
	"github.com/circonus-labs/circonus-gometrics/api"
	"github.com/circonus-labs/circonus-gometrics/api/config"
	"github.com/hashicorp/terraform/helper/schema"
)

const (
	accountAddress1Attr      = "address1"
	accountAddress2Attr      = "address2"
	accountCCEmailAttr       = "cc_email"
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
	accountUserIDAttr        = "id"
	accountUsersAttr         = "users"
)

func dataSourceCirconusAccount() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceCirconusAccountRead,

		Schema: map[string]*schema.Schema{
			accountAddress1Attr: &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: accountDescription[accountAddress1Attr],
			},
			accountAddress2Attr: &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: accountDescription[accountAddress2Attr],
			},
			accountCCEmailAttr: &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: accountDescription[accountCCEmailAttr],
			},
			accountIDAttr: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				// ConflictsWith: []string{accountCurrentAttr},
				ValidateFunc: _ValidateFuncs(
					_ValidateRegexp(accountIDAttr, config.AccountCIDRegex),
				),
				Description: accountDescription[accountIDAttr],
			},
			accountCityAttr: &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: accountDescription[accountCityAttr],
			},
			accountContactGroupsAttr: &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: accountDescription[accountContactGroupsAttr],
			},
			accountCountryAttr: &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: accountDescription[accountCountryAttr],
			},
			accountCurrentAttr: &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
				Description: accountDescription[accountCurrentAttr],
			},
			accountDescriptionAttr: &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: accountDescription[accountDescriptionAttr],
			},
			accountInvitesAttr: &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				Description: accountDescription[accountInvitesAttr],
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						accountEmailAttr: &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							Description: accountDescription[accountEmailAttr],
						},
						accountRoleAttr: &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							Description: accountDescription[accountRoleAttr],
						},
					},
				},
			},
			accountNameAttr: &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: accountDescription[accountNameAttr],
			},
			accountOwnerAttr: &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: accountDescription[accountOwnerAttr],
			},
			accountStateProvAttr: &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: accountDescription[accountStateProvAttr],
			},
			accountTimezoneAttr: &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: accountDescription[accountTimezoneAttr],
			},
			accountUIBaseURLAttr: &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: accountDescription[accountUIBaseURLAttr],
			},
			accountUsageAttr: &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				Description: accountDescription[accountUsageAttr],
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						accountLimitAttr: &schema.Schema{
							Type:        schema.TypeInt,
							Optional:    true,
							Computed:    true,
							Description: accountDescription[accountLimitAttr],
						},
						accountTypeAttr: &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							Description: accountDescription[accountTypeAttr],
						},
						accountUsedAttr: &schema.Schema{
							Type:        schema.TypeInt,
							Optional:    true,
							Computed:    true,
							Description: accountDescription[accountUsedAttr],
						},
					},
				},
			},
			accountUsersAttr: &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				Description: accountDescription[accountUsersAttr],
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						accountUserIDAttr: &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							Description: accountDescription[accountUserIDAttr],
						},
						accountRoleAttr: &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							Description: accountDescription[accountRoleAttr],
						},
					},
				},
			},
		},
	}
}

func dataSourceCirconusAccountRead(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*_ProviderContext)

	var cid string

	var a *api.Account
	var err error
	if v, ok := d.GetOk(accountIDAttr); ok {
		cid = v.(string)
	}

	if v, ok := d.GetOk(accountCurrentAttr); ok {
		if v.(bool) {
			cid = ""
		}
	}

	a, err = c.client.FetchAccount(api.CIDType(&cid))
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
			accountUserIDAttr: a.Users[i].UserCID,
			accountRoleAttr:   a.Users[i].Role,
		})
	}

	_StateSet(d, accountAddress1Attr, a.Address1)
	_StateSet(d, accountAddress2Attr, a.Address2)
	_StateSet(d, accountCCEmailAttr, a.CCEmail)
	_StateSet(d, accountIDAttr, a.CID)
	_StateSet(d, accountCityAttr, a.City)
	_StateSet(d, accountContactGroupsAttr, a.ContactGroups)
	_StateSet(d, accountCountryAttr, a.Country)
	_StateSet(d, accountDescriptionAttr, a.Description)
	_StateSet(d, accountInvitesAttr, invitesList)
	_StateSet(d, accountNameAttr, a.Name)
	_StateSet(d, accountOwnerAttr, a.OwnerCID)
	_StateSet(d, accountStateProvAttr, a.StateProv)
	_StateSet(d, accountTimezoneAttr, a.Timezone)
	_StateSet(d, accountUIBaseURLAttr, a.UIBaseURL)
	_StateSet(d, accountUsageAttr, usageList)
	_StateSet(d, accountUsersAttr, usersList)

	d.SetId(a.CID)

	return nil
}
