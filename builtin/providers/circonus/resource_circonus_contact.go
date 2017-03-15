package circonus

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/circonus-labs/circonus-gometrics/api"
	"github.com/circonus-labs/circonus-gometrics/api/config"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

const (
	// circonus_contact attributes
	contactAggregationWindowAttr = "aggregation_window"
	contactAlertOptionAttr       = "alert_option"
	contactEmailAttr             = "email"
	contactHTTPAttr              = "http"
	contactIRCAttr               = "irc"
	contactLongMessageAttr       = "long_message"
	contactLongSubjectAttr       = "long_subject"
	contactLongSummaryAttr       = "long_summary"
	contactNameAttr              = "name"
	contactPagerDutyAttr         = "pager_duty"
	contactSMSAttr               = "sms"
	contactShortMessageAttr      = "short_message"
	contactShortSummaryAttr      = "short_summary"
	contactSlackAttr             = "slack"
	contactTagsAttr              = "tags"
	contactVictorOpsAttr         = "victorops"
	contactXMPPAttr              = "xmpp"

	// circonus_contact.alert_option attributes
	contactEscalateAfterAttr = "escalate_after"
	contactEscalateToAttr    = "escalate_to"
	contactReminderAttr      = "reminder"
	contactSeverityAttr      = "severity"

	// circonus_contact.email attributes
	contactEmailAddressAttr = "address"
	//contactUserCIDAttr

	// circonus_contact.http attributes
	contactHTTPAddressAttr schemaAttr = "address"
	contactHTTPFormatAttr             = "format"
	contactHTTPMethodAttr             = "method"

	// circonus_contact.irc attributes
	//contactUserCIDAttr

	// circonus_contact.pager_duty attributes
	//contactContactGroupFallbackAttr
	contactPagerDutyServiceKeyAttr schemaAttr = "service_key"
	contactPagerDutyWebhookURLAttr schemaAttr = "webhook_url"

	// circonus_contact.slack attributes
	//contactContactGroupFallbackAttr
	contactSlackButtonsAttr  = "buttons"
	contactSlackChannelAttr  = "channel"
	contactSlackTeamAttr     = "team"
	contactSlackUsernameAttr = "username"

	// circonus_contact.sms attributes
	contactSMSAddressAttr = "address"
	//contactUserCIDAttr

	// circonus_contact.victorops attributes
	//contactContactGroupFallbackAttr
	contactVictorOpsAPIKeyAttr   = "api_key"
	contactVictorOpsCriticalAttr = "critical"
	contactVictorOpsInfoAttr     = "info"
	contactVictorOpsTeamAttr     = "team"
	contactVictorOpsWarningAttr  = "warning"

	// circonus_contact.victorops attributes
	//contactUserCIDAttr
	contactXMPPAddressAttr = "address"

	// circonus_contact read-only attributes
	contactLastModifiedAttr   = "last_modified"
	contactLastModifiedByAttr = "last_modified_by"

	// circonus_contact.* shared attributes
	contactContactGroupFallbackAttr = "contact_group_fallback"
	contactUserCIDAttr              = "user"
)

const (
	// Contact methods from Circonus
	circonusMethodEmail     = "email"
	circonusMethodHTTP      = "http"
	circonusMethodIRC       = "irc"
	circonusMethodPagerDuty = "pagerduty"
	circonusMethodSlack     = "slack"
	circonusMethodSMS       = "sms"
	circonusMethodVictorOps = "victorops"
	circonusMethodXMPP      = "xmpp"
)

type contactHTTPInfo struct {
	Address string `json:"url"`
	Format  string `json:"params"`
	Method  string `json:"method"`
}

type contactPagerDutyInfo struct {
	FallbackGroupCID int    `json:"failover_group,string"`
	ServiceKey       string `json:"service_key"`
	WebhookURL       string `json:"webhook_url"`
}

type contactSlackInfo struct {
	Buttons          int    `json:"buttons,string"`
	Channel          string `json:"channel"`
	FallbackGroupCID int    `json:"failover_group,string"`
	Team             string `json:"team"`
	Username         string `json:"username"`
}

type contactVictorOpsInfo struct {
	APIKey           string `json:"api_key"`
	Critical         int    `json:"critical,string"`
	FallbackGroupCID int    `json:"failover_group,string"`
	Info             int    `json:"info,string"`
	Team             string `json:"team"`
	Warning          int    `json:"warning,string"`
}

var contactGroupDescriptions = attrDescrs{
	contactAggregationWindowAttr:    "",
	contactAlertOptionAttr:          "",
	contactContactGroupFallbackAttr: "",
	contactEmailAttr:                "",
	contactHTTPAttr:                 "",
	contactIRCAttr:                  "",
	contactLastModifiedAttr:         "",
	contactLastModifiedByAttr:       "",
	contactLongMessageAttr:          "",
	contactLongSubjectAttr:          "",
	contactLongSummaryAttr:          "",
	contactNameAttr:                 "",
	contactPagerDutyAttr:            "",
	contactSMSAttr:                  "",
	contactShortMessageAttr:         "",
	contactShortSummaryAttr:         "",
	contactSlackAttr:                "",
	contactTagsAttr:                 "",
	contactVictorOpsAttr:            "",
	contactXMPPAttr:                 "",
}

var contactAlertDescriptions = attrDescrs{
	contactEscalateAfterAttr: "",
	contactEscalateToAttr:    "",
	contactReminderAttr:      "",
	contactSeverityAttr:      "",
}

var contactEmailDescriptions = attrDescrs{
	contactEmailAddressAttr: "",
	contactUserCIDAttr:      "",
}

var contactHTTPDescriptions = attrDescrs{
	contactHTTPAddressAttr: "",
	contactHTTPFormatAttr:  "",
	contactHTTPMethodAttr:  "",
}

var contactPagerDutyDescriptions = attrDescrs{
	contactContactGroupFallbackAttr: "",
	contactPagerDutyServiceKeyAttr:  "",
	contactPagerDutyWebhookURLAttr:  "",
}

var contactSlackDescriptions = attrDescrs{
	contactContactGroupFallbackAttr: "",
	contactSlackButtonsAttr:         "",
	contactSlackChannelAttr:         "",
	contactSlackTeamAttr:            "",
	contactSlackUsernameAttr:        "Username Slackbot uses in Slack to deliver a notification",
}

var contactSMSDescriptions = attrDescrs{
	contactSMSAddressAttr: "",
	contactUserCIDAttr:    "",
}

var contactVictorOpsDescriptions = attrDescrs{
	contactContactGroupFallbackAttr: "",
	contactVictorOpsAPIKeyAttr:      "",
	contactVictorOpsCriticalAttr:    "",
	contactVictorOpsInfoAttr:        "",
	contactVictorOpsTeamAttr:        "",
	contactVictorOpsWarningAttr:     "",
}

var contactXMPPDescriptions = attrDescrs{
	contactUserCIDAttr:     "",
	contactXMPPAddressAttr: "",
}

func resourceContactGroup() *schema.Resource {
	return &schema.Resource{
		Create: contactGroupCreate,
		Read:   contactGroupRead,
		Update: contactGroupUpdate,
		Delete: contactGroupDelete,
		Exists: contactGroupExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: convertToHelperSchema(contactGroupDescriptions, map[schemaAttr]*schema.Schema{
			contactAggregationWindowAttr: &schema.Schema{
				Type:             schema.TypeString,
				Optional:         true,
				Default:          defaultCirconusAggregationWindow,
				DiffSuppressFunc: suppressEquivalentTimeDurations,
				StateFunc:        normalizeTimeDurationStringToSeconds,
				ValidateFunc: validateFuncs(
					validateDurationMin(contactAggregationWindowAttr, "0s"),
				),
			},
			contactAlertOptionAttr: &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Set:      contactGroupAlertOptionsChecksum,
				Elem: &schema.Resource{
					Schema: convertToHelperSchema(contactAlertDescriptions, map[schemaAttr]*schema.Schema{
						contactEscalateAfterAttr: &schema.Schema{
							Type:             schema.TypeString,
							Optional:         true,
							DiffSuppressFunc: suppressEquivalentTimeDurations,
							StateFunc:        normalizeTimeDurationStringToSeconds,
							ValidateFunc: validateFuncs(
								validateDurationMin(contactEscalateAfterAttr, defaultCirconusAlertMinEscalateAfter),
							),
						},
						contactEscalateToAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateContactGroupCID(contactEscalateToAttr),
						},
						contactReminderAttr: &schema.Schema{
							Type:             schema.TypeString,
							Optional:         true,
							DiffSuppressFunc: suppressEquivalentTimeDurations,
							StateFunc:        normalizeTimeDurationStringToSeconds,
							ValidateFunc: validateFuncs(
								validateDurationMin(contactReminderAttr, "0s"),
							),
						},
						contactSeverityAttr: &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
							ValidateFunc: validateFuncs(
								validateIntMin(contactSeverityAttr, minSeverity),
								validateIntMax(contactSeverityAttr, maxSeverity),
							),
						},
					}),
				},
			},
			contactEmailAttr: &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: convertToHelperSchema(contactEmailDescriptions, map[schemaAttr]*schema.Schema{
						contactEmailAddressAttr: &schema.Schema{
							Type:          schema.TypeString,
							Optional:      true,
							ConflictsWith: []string{contactEmailAttr + "." + contactUserCIDAttr},
						},
						contactUserCIDAttr: &schema.Schema{
							Type:          schema.TypeString,
							Optional:      true,
							ValidateFunc:  validateUserCID(contactUserCIDAttr),
							ConflictsWith: []string{contactEmailAttr + "." + contactEmailAddressAttr},
						},
					}),
				},
			},
			contactHTTPAttr: &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: convertToHelperSchema(contactHTTPDescriptions, map[schemaAttr]*schema.Schema{
						contactHTTPAddressAttr: &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateHTTPURL(contactHTTPAddressAttr, urlBasicCheck),
						},
						contactHTTPFormatAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							Default:      defaultCirconusHTTPFormat,
							ValidateFunc: validateStringIn(contactHTTPFormatAttr, validContactHTTPFormats),
						},
						contactHTTPMethodAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							Default:      defaultCirconusHTTPMethod,
							ValidateFunc: validateStringIn(contactHTTPMethodAttr, validContactHTTPMethods),
						},
					}),
				},
			},
			contactIRCAttr: &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						contactUserCIDAttr: &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateUserCID(contactUserCIDAttr),
						},
					},
				},
			},
			contactLongMessageAttr: &schema.Schema{
				Type:      schema.TypeString,
				Optional:  true,
				StateFunc: suppressWhitespace,
			},
			contactLongSubjectAttr: &schema.Schema{
				Type:      schema.TypeString,
				Optional:  true,
				StateFunc: suppressWhitespace,
			},
			contactLongSummaryAttr: &schema.Schema{
				Type:      schema.TypeString,
				Optional:  true,
				StateFunc: suppressWhitespace,
			},
			contactNameAttr: &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			contactPagerDutyAttr: &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: convertToHelperSchema(contactPagerDutyDescriptions, map[schemaAttr]*schema.Schema{
						contactContactGroupFallbackAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateContactGroupCID(contactContactGroupFallbackAttr),
						},
						contactPagerDutyServiceKeyAttr: &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							Sensitive:    true,
							ValidateFunc: validateRegexp(contactPagerDutyServiceKeyAttr, `^[a-zA-Z0-9]{32}$`),
						},
						contactPagerDutyWebhookURLAttr: &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateHTTPURL(contactPagerDutyWebhookURLAttr, urlIsAbs),
						},
					}),
				},
			},
			contactShortMessageAttr: &schema.Schema{
				Type:      schema.TypeString,
				Optional:  true,
				StateFunc: suppressWhitespace,
			},
			contactShortSummaryAttr: &schema.Schema{
				Type:      schema.TypeString,
				Optional:  true,
				StateFunc: suppressWhitespace,
			},
			contactSlackAttr: &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: convertToHelperSchema(contactSlackDescriptions, map[schemaAttr]*schema.Schema{
						contactContactGroupFallbackAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateContactGroupCID(contactContactGroupFallbackAttr),
						},
						contactSlackButtonsAttr: &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						contactSlackChannelAttr: &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: validateFuncs(
								validateRegexp(contactSlackChannelAttr, `^#[\S]+$`),
							),
						},
						contactSlackTeamAttr: &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						contactSlackUsernameAttr: &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  defaultCirconusSlackUsername,
							ValidateFunc: validateFuncs(
								validateRegexp(contactSlackChannelAttr, `^[\S]+$`),
							),
						},
					}),
				},
			},
			contactSMSAttr: &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: convertToHelperSchema(contactSMSDescriptions, map[schemaAttr]*schema.Schema{
						contactSMSAddressAttr: &schema.Schema{
							Type:          schema.TypeString,
							Optional:      true,
							ConflictsWith: []string{contactSMSAttr + "." + contactUserCIDAttr},
						},
						contactUserCIDAttr: &schema.Schema{
							Type:          schema.TypeString,
							Optional:      true,
							ValidateFunc:  validateUserCID(contactUserCIDAttr),
							ConflictsWith: []string{contactSMSAttr + "." + contactSMSAddressAttr},
						},
					}),
				},
			},
			contactTagsAttr: tagMakeConfigSchema(contactTagsAttr),
			contactVictorOpsAttr: &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: convertToHelperSchema(contactVictorOpsDescriptions, map[schemaAttr]*schema.Schema{
						contactContactGroupFallbackAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateContactGroupCID(contactContactGroupFallbackAttr),
						},
						contactVictorOpsAPIKeyAttr: &schema.Schema{
							Type:      schema.TypeString,
							Required:  true,
							Sensitive: true,
						},
						contactVictorOpsCriticalAttr: &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
							ValidateFunc: validateFuncs(
								validateIntMin(contactVictorOpsCriticalAttr, 1),
								validateIntMax(contactVictorOpsCriticalAttr, 5),
							),
						},
						contactVictorOpsInfoAttr: &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
							ValidateFunc: validateFuncs(
								validateIntMin(contactVictorOpsInfoAttr, 1),
								validateIntMax(contactVictorOpsInfoAttr, 5),
							),
						},
						contactVictorOpsTeamAttr: &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						contactVictorOpsWarningAttr: &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
							ValidateFunc: validateFuncs(
								validateIntMin(contactVictorOpsWarningAttr, 1),
								validateIntMax(contactVictorOpsWarningAttr, 5),
							),
						},
					}),
				},
			},
			contactXMPPAttr: &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: convertToHelperSchema(contactXMPPDescriptions, map[schemaAttr]*schema.Schema{
						contactXMPPAddressAttr: &schema.Schema{
							Type:          schema.TypeString,
							Optional:      true,
							ConflictsWith: []string{contactXMPPAttr + "." + contactUserCIDAttr},
						},
						contactUserCIDAttr: &schema.Schema{
							Type:          schema.TypeString,
							Optional:      true,
							ValidateFunc:  validateUserCID(contactUserCIDAttr),
							ConflictsWith: []string{contactXMPPAttr + "." + contactXMPPAddressAttr},
						},
					}),
				},
			},

			// OUT parameters
			contactLastModifiedAttr: &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},
			contactLastModifiedByAttr: &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		}),
	}
}

func contactGroupCreate(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*providerContext)

	in, err := getContactGroupInput(d)
	if err != nil {
		return err
	}

	cg, err := ctxt.client.CreateContactGroup(in)
	if err != nil {
		return err
	}

	d.SetId(cg.CID)

	return contactGroupRead(d, meta)
}

func contactGroupExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	c := meta.(*providerContext)

	cid := d.Id()
	cg, err := c.client.FetchContactGroup(api.CIDType(&cid))
	if err != nil {
		if strings.Contains(err.Error(), defaultCirconus404ErrorString) {
			return false, nil
		}

		return false, err
	}

	if cg.CID == "" {
		return false, nil
	}

	return true, nil
}

func contactGroupRead(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*providerContext)

	cid := d.Id()

	cg, err := c.client.FetchContactGroup(api.CIDType(&cid))
	if err != nil {
		return err
	}

	if cg.CID == "" {
		return nil
	}

	d.SetId(cg.CID)

	httpState, err := contactGroupHTTPToState(cg)
	if err != nil {
		return err
	}

	pagerDutyState, err := contactGroupPagerDutyToState(cg)
	if err != nil {
		return err
	}

	slackState, err := contactGroupSlackToState(cg)
	if err != nil {
		return err
	}

	smsState, err := contactGroupSMSToState(cg)
	if err != nil {
		return err
	}

	victorOpsState, err := contactGroupVictorOpsToState(cg)
	if err != nil {
		return err
	}

	xmppState, err := contactGroupXMPPToState(cg)
	if err != nil {
		return err
	}

	d.Set(contactAggregationWindowAttr, fmt.Sprintf("%ds", cg.AggregationWindow))

	if err := d.Set(contactAlertOptionAttr, contactGroupAlertOptionsToState(cg)); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Unable to store contact %q attribute: {{err}}", contactAlertOptionAttr), err)
	}

	if err := d.Set(contactEmailAttr, contactGroupEmailToState(cg)); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Unable to store contact %q attribute: {{err}}", contactEmailAttr), err)
	}

	if err := d.Set(contactHTTPAttr, httpState); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Unable to store contact %q attribute: {{err}}", contactHTTPAttr), err)
	}

	if err := d.Set(contactIRCAttr, contactGroupIRCToState(cg)); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Unable to store contact %q attribute: {{err}}", contactIRCAttr), err)
	}

	d.Set(contactLongMessageAttr, cg.AlertFormats.LongMessage)
	d.Set(contactLongSubjectAttr, cg.AlertFormats.LongSubject)
	d.Set(contactLongSummaryAttr, cg.AlertFormats.LongSummary)
	d.Set(contactNameAttr, cg.Name)

	if err := d.Set(contactPagerDutyAttr, pagerDutyState); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Unable to store contact %q attribute: {{err}}", contactPagerDutyAttr), err)
	}

	d.Set(contactShortMessageAttr, cg.AlertFormats.ShortMessage)
	d.Set(contactShortSummaryAttr, cg.AlertFormats.ShortSummary)

	if err := d.Set(contactSlackAttr, slackState); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Unable to store contact %q attribute: {{err}}", contactSlackAttr), err)
	}

	if err := d.Set(contactSMSAttr, smsState); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Unable to store contact %q attribute: {{err}}", contactSMSAttr), err)
	}

	if err := d.Set(contactTagsAttr, cg.Tags); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Unable to store contact %q attribute: {{err}}", contactTagsAttr), err)
	}

	if err := d.Set(contactVictorOpsAttr, victorOpsState); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Unable to store contact %q attribute: {{err}}", contactVictorOpsAttr), err)
	}

	if err := d.Set(contactXMPPAttr, xmppState); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Unable to store contact %q attribute: {{err}}", contactXMPPAttr), err)
	}

	// Out parameters
	d.Set(contactLastModifiedAttr, cg.LastModified)
	d.Set(contactLastModifiedByAttr, cg.LastModifiedBy)

	return nil
}

func contactGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*providerContext)

	in, err := getContactGroupInput(d)
	if err != nil {
		return err
	}

	in.CID = d.Id()

	if _, err := c.client.UpdateContactGroup(in); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("unable to update contact group %q: {{err}}", d.Id()), err)
	}

	return contactGroupRead(d, meta)
}

func contactGroupDelete(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*providerContext)

	cid := d.Id()
	if _, err := c.client.DeleteContactGroupByCID(api.CIDType(&cid)); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("unable to delete contact group %q: {{err}}", d.Id()), err)
	}

	d.SetId("")

	return nil
}

func contactGroupAlertOptionsToState(cg *api.ContactGroup) []interface{} {
	if config.NumSeverityLevels != len(cg.Reminders) {
		log.Printf("[FATAL] PROVIDER BUG: Need to update constants in contactGroupAlertOptionsToState re: reminders")
		return nil
	}
	if config.NumSeverityLevels != len(cg.Escalations) {
		log.Printf("[FATAL] PROVIDER BUG: Need to update constants in contactGroupAlertOptionsToState re: escalations")
		return nil
	}

	// Populate all alert options for every severity level.  We'll prune empty
	// values at the end of this function.
	const defaultNumAlertOptions = 4
	alertOptions := make([]*map[string]interface{}, config.NumSeverityLevels)
	for severityIndex := 0; severityIndex < config.NumSeverityLevels; severityIndex++ {
		sevAction := make(map[string]interface{}, defaultNumAlertOptions)
		sevAction[string(contactSeverityAttr)] = severityIndex + 1
		alertOptions[severityIndex] = &sevAction
	}

	for severityIndex, reminder := range cg.Reminders {
		if reminder != 0 {
			(*alertOptions[severityIndex])[string(contactReminderAttr)] = fmt.Sprintf("%ds", reminder)
		}
	}

	for severityIndex, escalate := range cg.Escalations {
		if escalate == nil {
			continue
		}

		(*alertOptions[severityIndex])[string(contactEscalateAfterAttr)] = fmt.Sprintf("%ds", escalate.After)
		(*alertOptions[severityIndex])[string(contactEscalateToAttr)] = escalate.ContactGroupCID
	}

	alertOptionsList := make([]interface{}, 0, config.NumSeverityLevels)
	for i := 0; i < config.NumSeverityLevels; i++ {
		// NOTE: the 1 is from the always-populated contactSeverityAttr which is
		// always set.
		if len(*alertOptions[i]) > 1 {
			alertOptionsList = append(alertOptionsList, *alertOptions[i])
		}
	}

	return alertOptionsList
}

func contactGroupEmailToState(cg *api.ContactGroup) []interface{} {
	emailContacts := make([]interface{}, 0, len(cg.Contacts.Users)+len(cg.Contacts.External))

	for _, ext := range cg.Contacts.External {
		switch ext.Method {
		case circonusMethodEmail:
			emailContacts = append(emailContacts, map[string]interface{}{
				contactEmailAddressAttr: ext.Info,
			})
		}
	}

	for _, user := range cg.Contacts.Users {
		switch user.Method {
		case circonusMethodEmail:
			emailContacts = append(emailContacts, map[string]interface{}{
				contactUserCIDAttr: user.UserCID,
			})
		}
	}

	return emailContacts
}

func contactGroupHTTPToState(cg *api.ContactGroup) ([]interface{}, error) {
	httpContacts := make([]interface{}, 0, len(cg.Contacts.External))

	for _, ext := range cg.Contacts.External {
		switch ext.Method {
		case circonusMethodHTTP:
			url := contactHTTPInfo{}
			if err := json.Unmarshal([]byte(ext.Info), &url); err != nil {
				return nil, errwrap.Wrapf(fmt.Sprintf("unable to decode external %s JSON (%q): {{err}}", contactHTTPAttr, ext.Info), err)
			}

			httpContacts = append(httpContacts, map[string]interface{}{
				string(contactHTTPAddressAttr): url.Address,
				string(contactHTTPFormatAttr):  url.Format,
				string(contactHTTPMethodAttr):  url.Method,
			})
		}
	}

	return httpContacts, nil
}

func getContactGroupInput(d *schema.ResourceData) (*api.ContactGroup, error) {
	cg := api.NewContactGroup()
	if v, ok := d.GetOk(contactAggregationWindowAttr); ok {
		aggWindow, _ := time.ParseDuration(v.(string))
		cg.AggregationWindow = uint(aggWindow.Seconds())
	}

	if v, ok := d.GetOk(contactAlertOptionAttr); ok {
		alertOptionsRaw := v.(*schema.Set).List()

		ensureEscalationSeverity := func(severity int) {
			if cg.Escalations[severity] == nil {
				cg.Escalations[severity] = &api.ContactGroupEscalation{}
			}
		}

		for _, alertOptionRaw := range alertOptionsRaw {
			alertOptionsMap := alertOptionRaw.(map[string]interface{})

			severityIndex := -1

			if optRaw, ok := alertOptionsMap[contactSeverityAttr]; ok {
				severityIndex = optRaw.(int) - 1
			}

			if optRaw, ok := alertOptionsMap[contactEscalateAfterAttr]; ok {
				if optRaw.(string) != "" {
					d, _ := time.ParseDuration(optRaw.(string))
					if d != 0 {
						ensureEscalationSeverity(severityIndex)
						cg.Escalations[severityIndex].After = uint(d.Seconds())
					}
				}
			}

			if optRaw, ok := alertOptionsMap[contactEscalateToAttr]; ok && optRaw.(string) != "" {
				ensureEscalationSeverity(severityIndex)
				cg.Escalations[severityIndex].ContactGroupCID = optRaw.(string)
			}

			if optRaw, ok := alertOptionsMap[contactReminderAttr]; ok {
				if optRaw.(string) == "" {
					optRaw = "0s"
				}

				d, _ := time.ParseDuration(optRaw.(string))
				cg.Reminders[severityIndex] = uint(d.Seconds())
			}
		}
	}

	if v, ok := d.GetOk(contactNameAttr); ok {
		cg.Name = v.(string)
	}

	if v, ok := d.GetOk(contactEmailAttr); ok {
		emailListRaw := v.(*schema.Set).List()
		for _, emailMapRaw := range emailListRaw {
			emailMap := emailMapRaw.(map[string]interface{})

			var requiredAttrFound bool
			if v, ok := emailMap[contactEmailAddressAttr]; ok && v.(string) != "" {
				requiredAttrFound = true
				cg.Contacts.External = append(cg.Contacts.External, api.ContactGroupContactsExternal{
					Info:   v.(string),
					Method: circonusMethodEmail,
				})
			}

			if v, ok := emailMap[contactUserCIDAttr]; ok && v.(string) != "" {
				requiredAttrFound = true
				cg.Contacts.Users = append(cg.Contacts.Users, api.ContactGroupContactsUser{
					Method:  circonusMethodEmail,
					UserCID: v.(string),
				})
			}

			// Can't mark two attributes that are conflicting as required so we do our
			// own validation check here.
			if !requiredAttrFound {
				return nil, fmt.Errorf("In type %s, either %s or %s must be specified", contactEmailAttr, contactEmailAddressAttr, contactUserCIDAttr)
			}
		}
	}

	if v, ok := d.GetOk(contactHTTPAttr); ok {
		httpListRaw := v.(*schema.Set).List()
		for _, httpMapRaw := range httpListRaw {
			httpMap := httpMapRaw.(map[string]interface{})

			httpInfo := contactHTTPInfo{}

			if v, ok := httpMap[string(contactHTTPAddressAttr)]; ok {
				httpInfo.Address = v.(string)
			}

			if v, ok := httpMap[string(contactHTTPFormatAttr)]; ok {
				httpInfo.Format = v.(string)
			}

			if v, ok := httpMap[string(contactHTTPMethodAttr)]; ok {
				httpInfo.Method = v.(string)
			}

			js, err := json.Marshal(httpInfo)
			if err != nil {
				return nil, errwrap.Wrapf(fmt.Sprintf("error marshalling %s JSON config string: {{err}}", contactHTTPAttr), err)
			}

			cg.Contacts.External = append(cg.Contacts.External, api.ContactGroupContactsExternal{
				Info:   string(js),
				Method: circonusMethodHTTP,
			})
		}
	}

	if v, ok := d.GetOk(contactIRCAttr); ok {
		ircListRaw := v.(*schema.Set).List()
		for _, ircMapRaw := range ircListRaw {
			ircMap := ircMapRaw.(map[string]interface{})

			if v, ok := ircMap[contactUserCIDAttr]; ok && v.(string) != "" {
				cg.Contacts.Users = append(cg.Contacts.Users, api.ContactGroupContactsUser{
					Method:  circonusMethodIRC,
					UserCID: v.(string),
				})
			}
		}
	}

	if v, ok := d.GetOk(contactPagerDutyAttr); ok {
		pagerDutyListRaw := v.(*schema.Set).List()
		for _, pagerDutyMapRaw := range pagerDutyListRaw {
			pagerDutyMap := pagerDutyMapRaw.(map[string]interface{})

			pagerDutyInfo := contactPagerDutyInfo{}

			if v, ok := pagerDutyMap[contactContactGroupFallbackAttr]; ok && v.(string) != "" {
				cid := v.(string)
				contactGroupID, err := failoverGroupCIDToID(api.CIDType(&cid))
				if err != nil {
					return nil, errwrap.Wrapf("error reading contact group CID: {{err}}", err)
				}
				pagerDutyInfo.FallbackGroupCID = contactGroupID
			}

			if v, ok := pagerDutyMap[string(contactPagerDutyServiceKeyAttr)]; ok {
				pagerDutyInfo.ServiceKey = v.(string)
			}

			if v, ok := pagerDutyMap[string(contactPagerDutyWebhookURLAttr)]; ok {
				pagerDutyInfo.WebhookURL = v.(string)
			}

			js, err := json.Marshal(pagerDutyInfo)
			if err != nil {
				return nil, errwrap.Wrapf(fmt.Sprintf("error marshalling %s JSON config string: {{err}}", contactPagerDutyAttr), err)
			}

			cg.Contacts.External = append(cg.Contacts.External, api.ContactGroupContactsExternal{
				Info:   string(js),
				Method: circonusMethodPagerDuty,
			})
		}
	}

	if v, ok := d.GetOk(contactSlackAttr); ok {
		slackListRaw := v.(*schema.Set).List()
		for _, slackMapRaw := range slackListRaw {
			slackMap := slackMapRaw.(map[string]interface{})

			slackInfo := contactSlackInfo{}

			var buttons int
			if v, ok := slackMap[contactSlackButtonsAttr]; ok {
				if v.(bool) {
					buttons = 1
				}
				slackInfo.Buttons = buttons
			}

			if v, ok := slackMap[contactSlackChannelAttr]; ok {
				slackInfo.Channel = v.(string)
			}

			if v, ok := slackMap[contactContactGroupFallbackAttr]; ok && v.(string) != "" {
				cid := v.(string)
				contactGroupID, err := failoverGroupCIDToID(api.CIDType(&cid))
				if err != nil {
					return nil, errwrap.Wrapf("error reading contact group CID: {{err}}", err)
				}
				slackInfo.FallbackGroupCID = contactGroupID
			}

			if v, ok := slackMap[contactSlackTeamAttr]; ok {
				slackInfo.Team = v.(string)
			}

			if v, ok := slackMap[contactSlackUsernameAttr]; ok {
				slackInfo.Username = v.(string)
			}

			js, err := json.Marshal(slackInfo)
			if err != nil {
				return nil, errwrap.Wrapf(fmt.Sprintf("error marshalling %s JSON config string: {{err}}", contactSlackAttr), err)
			}

			cg.Contacts.External = append(cg.Contacts.External, api.ContactGroupContactsExternal{
				Info:   string(js),
				Method: circonusMethodSlack,
			})
		}
	}

	if v, ok := d.GetOk(contactSMSAttr); ok {
		smsListRaw := v.(*schema.Set).List()
		for _, smsMapRaw := range smsListRaw {
			smsMap := smsMapRaw.(map[string]interface{})

			var requiredAttrFound bool
			if v, ok := smsMap[contactSMSAddressAttr]; ok && v.(string) != "" {
				requiredAttrFound = true
				cg.Contacts.External = append(cg.Contacts.External, api.ContactGroupContactsExternal{
					Info:   v.(string),
					Method: circonusMethodSMS,
				})
			}

			if v, ok := smsMap[contactUserCIDAttr]; ok && v.(string) != "" {
				requiredAttrFound = true
				cg.Contacts.Users = append(cg.Contacts.Users, api.ContactGroupContactsUser{
					Method:  circonusMethodSMS,
					UserCID: v.(string),
				})
			}

			// Can't mark two attributes that are conflicting as required so we do our
			// own validation check here.
			if !requiredAttrFound {
				return nil, fmt.Errorf("In type %s, either %s or %s must be specified", contactEmailAttr, contactEmailAddressAttr, contactUserCIDAttr)
			}
		}
	}

	if v, ok := d.GetOk(contactVictorOpsAttr); ok {
		victorOpsListRaw := v.(*schema.Set).List()
		for _, victorOpsMapRaw := range victorOpsListRaw {
			victorOpsMap := victorOpsMapRaw.(map[string]interface{})

			victorOpsInfo := contactVictorOpsInfo{}

			if v, ok := victorOpsMap[contactContactGroupFallbackAttr]; ok && v.(string) != "" {
				cid := v.(string)
				contactGroupID, err := failoverGroupCIDToID(api.CIDType(&cid))
				if err != nil {
					return nil, errwrap.Wrapf("error reading contact group CID: {{err}}", err)
				}
				victorOpsInfo.FallbackGroupCID = contactGroupID
			}

			if v, ok := victorOpsMap[contactVictorOpsAPIKeyAttr]; ok {
				victorOpsInfo.APIKey = v.(string)
			}

			if v, ok := victorOpsMap[contactVictorOpsCriticalAttr]; ok {
				victorOpsInfo.Critical = v.(int)
			}

			if v, ok := victorOpsMap[contactVictorOpsInfoAttr]; ok {
				victorOpsInfo.Info = v.(int)
			}

			if v, ok := victorOpsMap[contactVictorOpsTeamAttr]; ok {
				victorOpsInfo.Team = v.(string)
			}

			if v, ok := victorOpsMap[contactVictorOpsWarningAttr]; ok {
				victorOpsInfo.Warning = v.(int)
			}

			js, err := json.Marshal(victorOpsInfo)
			if err != nil {
				return nil, errwrap.Wrapf(fmt.Sprintf("error marshalling %s JSON config string: {{err}}", contactVictorOpsAttr), err)
			}

			cg.Contacts.External = append(cg.Contacts.External, api.ContactGroupContactsExternal{
				Info:   string(js),
				Method: circonusMethodVictorOps,
			})
		}
	}

	if v, ok := d.GetOk(contactXMPPAttr); ok {
		xmppListRaw := v.(*schema.Set).List()
		for _, xmppMapRaw := range xmppListRaw {
			xmppMap := xmppMapRaw.(map[string]interface{})

			if v, ok := xmppMap[contactXMPPAddressAttr]; ok && v.(string) != "" {
				cg.Contacts.External = append(cg.Contacts.External, api.ContactGroupContactsExternal{
					Info:   v.(string),
					Method: circonusMethodXMPP,
				})
			}

			if v, ok := xmppMap[contactUserCIDAttr]; ok && v.(string) != "" {
				cg.Contacts.Users = append(cg.Contacts.Users, api.ContactGroupContactsUser{
					Method:  circonusMethodXMPP,
					UserCID: v.(string),
				})
			}
		}
	}

	if v, ok := d.GetOk(contactLongMessageAttr); ok {
		msg := v.(string)
		cg.AlertFormats.LongMessage = &msg
	}

	if v, ok := d.GetOk(contactLongSubjectAttr); ok {
		msg := v.(string)
		cg.AlertFormats.LongSubject = &msg
	}

	if v, ok := d.GetOk(contactLongSummaryAttr); ok {
		msg := v.(string)
		cg.AlertFormats.LongSummary = &msg
	}

	if v, ok := d.GetOk(contactShortMessageAttr); ok {
		msg := v.(string)
		cg.AlertFormats.ShortMessage = &msg
	}

	if v, ok := d.GetOk(contactShortSummaryAttr); ok {
		msg := v.(string)
		cg.AlertFormats.ShortSummary = &msg
	}

	if v, ok := d.GetOk(contactShortMessageAttr); ok {
		msg := v.(string)
		cg.AlertFormats.ShortMessage = &msg
	}

	if v, found := d.GetOk(checkTagsAttr); found {
		cg.Tags = derefStringList(flattenSet(v.(*schema.Set)))
	}

	if err := validateContactGroup(cg); err != nil {
		return nil, err
	}

	return cg, nil
}

func contactGroupIRCToState(cg *api.ContactGroup) []interface{} {
	ircContacts := make([]interface{}, 0, len(cg.Contacts.Users))

	for _, user := range cg.Contacts.Users {
		switch user.Method {
		case circonusMethodIRC:
			ircContacts = append(ircContacts, map[string]interface{}{
				contactUserCIDAttr: user.UserCID,
			})
		}
	}

	return ircContacts
}

func contactGroupPagerDutyToState(cg *api.ContactGroup) ([]interface{}, error) {
	pdContacts := make([]interface{}, 0, len(cg.Contacts.External))

	for _, ext := range cg.Contacts.External {
		switch ext.Method {
		case circonusMethodPagerDuty:
			pdInfo := contactPagerDutyInfo{}
			if err := json.Unmarshal([]byte(ext.Info), &pdInfo); err != nil {
				return nil, errwrap.Wrapf(fmt.Sprintf("unable to decode external %s JSON (%q): {{err}}", contactPagerDutyAttr, ext.Info), err)
			}

			pdContacts = append(pdContacts, map[string]interface{}{
				string(contactContactGroupFallbackAttr): failoverGroupIDToCID(pdInfo.FallbackGroupCID),
				string(contactPagerDutyServiceKeyAttr):  pdInfo.ServiceKey,
				string(contactPagerDutyWebhookURLAttr):  pdInfo.WebhookURL,
			})
		}
	}

	return pdContacts, nil
}

func contactGroupSlackToState(cg *api.ContactGroup) ([]interface{}, error) {
	slackContacts := make([]interface{}, 0, len(cg.Contacts.External))

	for _, ext := range cg.Contacts.External {
		switch ext.Method {
		case circonusMethodSlack:
			slackInfo := contactSlackInfo{}
			if err := json.Unmarshal([]byte(ext.Info), &slackInfo); err != nil {
				return nil, errwrap.Wrapf(fmt.Sprintf("unable to decode external %s JSON (%q): {{err}}", contactSlackAttr, ext.Info), err)
			}

			slackContacts = append(slackContacts, map[string]interface{}{
				contactContactGroupFallbackAttr: failoverGroupIDToCID(slackInfo.FallbackGroupCID),
				contactSlackButtonsAttr:         int(slackInfo.Buttons) == int(1),
				contactSlackChannelAttr:         slackInfo.Channel,
				contactSlackTeamAttr:            slackInfo.Team,
				contactSlackUsernameAttr:        slackInfo.Username,
			})
		}
	}

	return slackContacts, nil
}

func contactGroupSMSToState(cg *api.ContactGroup) ([]interface{}, error) {
	smsContacts := make([]interface{}, 0, len(cg.Contacts.Users)+len(cg.Contacts.External))

	for _, ext := range cg.Contacts.External {
		switch ext.Method {
		case circonusMethodSMS:
			smsContacts = append(smsContacts, map[string]interface{}{
				contactSMSAddressAttr: ext.Info,
			})
		}
	}

	for _, user := range cg.Contacts.Users {
		switch user.Method {
		case circonusMethodSMS:
			smsContacts = append(smsContacts, map[string]interface{}{
				contactUserCIDAttr: user.UserCID,
			})
		}
	}

	return smsContacts, nil
}

func contactGroupVictorOpsToState(cg *api.ContactGroup) ([]interface{}, error) {
	victorOpsContacts := make([]interface{}, 0, len(cg.Contacts.External))

	for _, ext := range cg.Contacts.External {
		switch ext.Method {
		case circonusMethodVictorOps:
			victorOpsInfo := contactVictorOpsInfo{}
			if err := json.Unmarshal([]byte(ext.Info), &victorOpsInfo); err != nil {
				return nil, errwrap.Wrapf(fmt.Sprintf("unable to decode external %s JSON (%q): {{err}}", contactVictorOpsInfoAttr, ext.Info), err)
			}

			victorOpsContacts = append(victorOpsContacts, map[string]interface{}{
				contactContactGroupFallbackAttr: failoverGroupIDToCID(victorOpsInfo.FallbackGroupCID),
				contactVictorOpsAPIKeyAttr:      victorOpsInfo.APIKey,
				contactVictorOpsCriticalAttr:    victorOpsInfo.Critical,
				contactVictorOpsInfoAttr:        victorOpsInfo.Info,
				contactVictorOpsTeamAttr:        victorOpsInfo.Team,
				contactVictorOpsWarningAttr:     victorOpsInfo.Warning,
			})
		}
	}

	return victorOpsContacts, nil
}

func contactGroupXMPPToState(cg *api.ContactGroup) ([]interface{}, error) {
	xmppContacts := make([]interface{}, 0, len(cg.Contacts.Users)+len(cg.Contacts.External))

	for _, ext := range cg.Contacts.External {
		switch ext.Method {
		case circonusMethodXMPP:
			xmppContacts = append(xmppContacts, map[string]interface{}{
				contactXMPPAddressAttr: ext.Info,
			})
		}
	}

	for _, user := range cg.Contacts.Users {
		switch user.Method {
		case circonusMethodXMPP:
			xmppContacts = append(xmppContacts, map[string]interface{}{
				contactUserCIDAttr: user.UserCID,
			})
		}
	}

	return xmppContacts, nil
}

// contactGroupAlertOptionsChecksum creates a stable hash of the normalized values
func contactGroupAlertOptionsChecksum(v interface{}) int {
	m := v.(map[string]interface{})
	b := &bytes.Buffer{}
	b.Grow(defaultHashBufSize)
	fmt.Fprintf(b, "%x", m[contactSeverityAttr].(int))
	fmt.Fprint(b, normalizeTimeDurationStringToSeconds(m[contactEscalateAfterAttr]))
	fmt.Fprint(b, m[contactEscalateToAttr])
	fmt.Fprint(b, normalizeTimeDurationStringToSeconds(m[contactReminderAttr]))
	return hashcode.String(b.String())
}
