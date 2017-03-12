package circonus

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/circonus-labs/circonus-gometrics/api"
	"github.com/circonus-labs/circonus-gometrics/api/config"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

const (
	// circonus_rule_set.* resource attribute names
	ruleSetCheckAttr      = "check"
	ruleSetIfAttr         = "if"
	ruleSetLinkAttr       = "link"
	ruleSetMetricTypeAttr = "metric_type"
	ruleSetNotesAttr      = "notes"
	ruleSetParentAttr     = "parent"
	ruleSetMetricNameAttr = "metric_name"
	ruleSetTagsAttr       = "tags"

	// circonus_rule_set.if.* resource attribute names
	ruleSetThenAttr  = "then"
	ruleSetValueAttr = "value"

	// circonus_rule_set.if.then.* resource attribute names
	ruleSetAfterAttr    = "after"
	ruleSetNotifyAttr   = "notify"
	ruleSetSeverityAttr = "severity"

	// circonus_rule_set.if.value.* resource attribute names
	ruleSetAbsentAttr     = "absent"      // apiRuleSetAbsent
	ruleSetChangedAttr    = "changed"     // apiRuleSetChanged
	ruleSetContainsAttr   = "contains"    // apiRuleSetContains
	ruleSetMatchAttr      = "match"       // apiRuleSetMatch
	ruleSetMaxValueAttr   = "max_value"   // apiRuleSetMaxValue
	ruleSetMinValueAttr   = "min_value"   // apiRuleSetMinValue
	ruleSetNotContainAttr = "not_contain" // apiRuleSetNotContains
	ruleSetNotMatchAttr   = "not_match"   // apiRuleSetNotMatch
	ruleSetOverAttr       = "over"

	// circonus_rule_set.if.value.over.* resource attribute names
	ruleSetLastAttr  = "last"
	ruleSetUsingAttr = "using"
)

const (
	// Different criteria that an api.RuleSetRule can return
	apiRuleSetAbsent      = "on absence"       // ruleSetAbsentAttr
	apiRuleSetChanged     = "on change"        // ruleSetChangedAttr
	apiRuleSetContains    = "contains"         // ruleSetContainsAttr
	apiRuleSetMatch       = "match"            // ruleSetMatchAttr
	apiRuleSetMaxValue    = "max value"        // ruleSetMaxValueAttr
	apiRuleSetMinValue    = "min value"        // ruleSetMinValueAttr
	apiRuleSetNotContains = "does not contain" // ruleSetNotContainAttr
	apiRuleSetNotMatch    = "does not match"   // ruleSetNotMatchAttr
)

var ruleSetDescriptions = attrDescrs{
	// circonus_rule_set.* resource attribute names
	ruleSetCheckAttr:      "The CID of the check that contains the metric for this rule set",
	ruleSetIfAttr:         "A rule to execute for this rule set",
	ruleSetLinkAttr:       "URL to show users when this rule set is active (e.g. wiki)",
	ruleSetMetricTypeAttr: "The type of data flowing through the specified metric stream",
	ruleSetNotesAttr:      "Notes describing this rule set",
	ruleSetParentAttr:     "Parent CID that must be healthy for this rule set to be active",
	ruleSetMetricNameAttr: "The name of the metric stream within a check to register the rule set with",
	ruleSetTagsAttr:       "Tags associated with this rule set",
}

var ruleSetIfDescriptions = attrDescrs{
	// circonus_rule_set.if.* resource attribute names
	ruleSetThenAttr:  "Description of the action(s) to take when this rule set is active",
	ruleSetValueAttr: "Predicate that the rule set uses to evaluate a stream of metrics",
}

var ruleSetIfValueDescriptions = attrDescrs{
	// circonus_rule_set.if.value.* resource attribute names
	ruleSetAbsentAttr:     "Fire the rule set if there has been no data for the given metric stream over the last duration",
	ruleSetChangedAttr:    "Boolean indicating the value has changed",
	ruleSetContainsAttr:   "Fire the rule set if the text metric contain the following string",
	ruleSetMatchAttr:      "Fire the rule set if the text metric exactly match the following string",
	ruleSetNotMatchAttr:   "Fire the rule set if the text metric not match the following string",
	ruleSetMinValueAttr:   "Fire the rule set if the numeric value less than the specified value",
	ruleSetNotContainAttr: "Fire the rule set if the text metric does not contain the following string",
	ruleSetMaxValueAttr:   "Fire the rule set if the numeric value is more than the specified value",
	ruleSetOverAttr:       "Use a derived value using a window",
	ruleSetThenAttr:       "Action to take when the rule set is active",
}

var ruleSetIfValueOverDescriptions = attrDescrs{
	// circonus_rule_set.if.value.over.* resource attribute names
	ruleSetLastAttr:  "Duration over which data from the last interval is examined",
	ruleSetUsingAttr: "Define the window funciton to use over the last duration",
}

var ruleSetIfThenDescriptions = attrDescrs{
	// circonus_rule_set.if.then.* resource attribute names
	ruleSetAfterAttr:    "The length of time we should wait before contacting the contact groups after this ruleset has faulted.",
	ruleSetNotifyAttr:   "List of contact groups to notify at the following appropriate severity if this rule set is active.",
	ruleSetSeverityAttr: "Send a notification at this severity level.",
}

func resourceRuleSet() *schema.Resource {
	makeConflictsWith := func(in ...schemaAttr) []string {
		out := make([]string, 0, len(in))
		for _, attr := range in {
			out = append(out, string(ruleSetIfAttr)+"."+string(ruleSetValueAttr)+"."+string(attr))
		}
		return out
	}

	return &schema.Resource{
		Create: ruleSetCreate,
		Read:   ruleSetRead,
		Update: ruleSetUpdate,
		Delete: ruleSetDelete,
		Exists: ruleSetExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: convertToHelperSchema(ruleSetDescriptions, map[schemaAttr]*schema.Schema{
			ruleSetCheckAttr: &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateRegexp(ruleSetCheckAttr, config.CheckCIDRegex),
			},
			ruleSetIfAttr: &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: convertToHelperSchema(ruleSetIfDescriptions, map[schemaAttr]*schema.Schema{
						ruleSetThenAttr: &schema.Schema{
							Type:     schema.TypeSet,
							MaxItems: 1,
							Optional: true,
							Elem: &schema.Resource{
								Schema: convertToHelperSchema(ruleSetIfThenDescriptions, map[schemaAttr]*schema.Schema{
									ruleSetAfterAttr: &schema.Schema{
										Type:             schema.TypeString,
										Optional:         true,
										DiffSuppressFunc: suppressEquivalentTimeDurations,
										StateFunc:        normalizeTimeDurationStringToSeconds,
										ValidateFunc: validateFuncs(
											validateDurationMin(ruleSetAfterAttr, "0s"),
										),
									},
									ruleSetNotifyAttr: &schema.Schema{
										Type:     schema.TypeList,
										Optional: true,
										MinItems: 1,
										Elem: &schema.Schema{
											Type:         schema.TypeString,
											ValidateFunc: validateContactGroupCID(ruleSetNotifyAttr),
										},
									},
									ruleSetSeverityAttr: &schema.Schema{
										Type:     schema.TypeInt,
										Optional: true,
										Default:  defaultAlertSeverity,
										ValidateFunc: validateFuncs(
											validateIntMax(ruleSetSeverityAttr, maxSeverity),
											validateIntMin(ruleSetSeverityAttr, minSeverity),
										),
									},
								}),
							},
						},
						ruleSetValueAttr: &schema.Schema{
							Type:     schema.TypeSet,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: convertToHelperSchema(ruleSetIfValueDescriptions, map[schemaAttr]*schema.Schema{
									ruleSetAbsentAttr: &schema.Schema{
										Type:             schema.TypeString, // Applies to text or numeric metrics
										Optional:         true,
										DiffSuppressFunc: suppressEquivalentTimeDurations,
										StateFunc:        normalizeTimeDurationStringToSeconds,
										ValidateFunc: validateFuncs(
											validateDurationMin(ruleSetAbsentAttr, ruleSetAbsentMin),
										),
										ConflictsWith: makeConflictsWith(ruleSetChangedAttr, ruleSetContainsAttr, ruleSetMatchAttr, ruleSetNotMatchAttr, ruleSetMinValueAttr, ruleSetNotContainAttr, ruleSetMaxValueAttr, ruleSetOverAttr),
									},
									ruleSetChangedAttr: &schema.Schema{
										Type:          schema.TypeBool, // Applies to text or numeric metrics
										Optional:      true,
										ConflictsWith: makeConflictsWith(ruleSetAbsentAttr, ruleSetContainsAttr, ruleSetMatchAttr, ruleSetNotMatchAttr, ruleSetMinValueAttr, ruleSetNotContainAttr, ruleSetMaxValueAttr, ruleSetOverAttr),
									},
									ruleSetContainsAttr: &schema.Schema{
										Type:          schema.TypeString, // Applies to text metrics only
										Optional:      true,
										ValidateFunc:  validateRegexp(ruleSetContainsAttr, `.+`),
										ConflictsWith: makeConflictsWith(ruleSetAbsentAttr, ruleSetChangedAttr, ruleSetMatchAttr, ruleSetNotMatchAttr, ruleSetMinValueAttr, ruleSetNotContainAttr, ruleSetMaxValueAttr, ruleSetOverAttr),
									},
									ruleSetMatchAttr: &schema.Schema{
										Type:          schema.TypeString, // Applies to text metrics only
										Optional:      true,
										ValidateFunc:  validateRegexp(ruleSetMatchAttr, `.+`),
										ConflictsWith: makeConflictsWith(ruleSetAbsentAttr, ruleSetChangedAttr, ruleSetContainsAttr, ruleSetNotMatchAttr, ruleSetMinValueAttr, ruleSetNotContainAttr, ruleSetMaxValueAttr, ruleSetOverAttr),
									},
									ruleSetNotMatchAttr: &schema.Schema{
										Type:          schema.TypeString, // Applies to text metrics only
										Optional:      true,
										ValidateFunc:  validateRegexp(ruleSetNotMatchAttr, `.+`),
										ConflictsWith: makeConflictsWith(ruleSetAbsentAttr, ruleSetChangedAttr, ruleSetContainsAttr, ruleSetMatchAttr, ruleSetMinValueAttr, ruleSetNotContainAttr, ruleSetMaxValueAttr, ruleSetOverAttr),
									},
									ruleSetMinValueAttr: &schema.Schema{
										Type:          schema.TypeString, // Applies to numeric metrics only
										Optional:      true,
										ValidateFunc:  validateRegexp(ruleSetMinValueAttr, `.+`), // TODO(sean): improve this regexp to match int and float
										ConflictsWith: makeConflictsWith(ruleSetAbsentAttr, ruleSetChangedAttr, ruleSetContainsAttr, ruleSetMatchAttr, ruleSetNotMatchAttr, ruleSetNotContainAttr, ruleSetMaxValueAttr),
									},
									ruleSetNotContainAttr: &schema.Schema{
										Type:          schema.TypeString, // Applies to text metrics only
										Optional:      true,
										ValidateFunc:  validateRegexp(ruleSetNotContainAttr, `.+`),
										ConflictsWith: makeConflictsWith(ruleSetAbsentAttr, ruleSetChangedAttr, ruleSetContainsAttr, ruleSetMatchAttr, ruleSetNotMatchAttr, ruleSetMinValueAttr, ruleSetMaxValueAttr, ruleSetOverAttr),
									},
									ruleSetMaxValueAttr: &schema.Schema{
										Type:          schema.TypeString, // Applies to numeric metrics only
										Optional:      true,
										ValidateFunc:  validateRegexp(ruleSetMaxValueAttr, `.+`), // TODO(sean): improve this regexp to match int and float
										ConflictsWith: makeConflictsWith(ruleSetAbsentAttr, ruleSetChangedAttr, ruleSetContainsAttr, ruleSetMatchAttr, ruleSetNotMatchAttr, ruleSetMinValueAttr, ruleSetNotContainAttr),
									},
									ruleSetOverAttr: &schema.Schema{
										Type:     schema.TypeSet,
										Optional: true,
										MaxItems: 1,
										// ruleSetOverAttr is only compatible with checks of
										// numeric type.  NOTE: It may be premature to conflict with
										// ruleSetChangedAttr.
										ConflictsWith: makeConflictsWith(ruleSetAbsentAttr, ruleSetChangedAttr, ruleSetContainsAttr, ruleSetMatchAttr, ruleSetNotMatchAttr, ruleSetNotContainAttr),
										Elem: &schema.Resource{
											Schema: convertToHelperSchema(ruleSetIfValueOverDescriptions, map[schemaAttr]*schema.Schema{
												ruleSetLastAttr: &schema.Schema{
													Type:             schema.TypeString,
													Optional:         true,
													Default:          defaultRuleSetLast,
													DiffSuppressFunc: suppressEquivalentTimeDurations,
													StateFunc:        normalizeTimeDurationStringToSeconds,
													ValidateFunc: validateFuncs(
														validateDurationMin(ruleSetLastAttr, "0s"),
													),
												},
												ruleSetUsingAttr: &schema.Schema{
													Type:         schema.TypeString,
													Optional:     true,
													Default:      defaultRuleSetWindowFunc,
													ValidateFunc: validateStringIn(ruleSetUsingAttr, validRuleSetWindowFuncs),
												},
											}),
										},
									},
								}),
							},
						},
					}),
				},
			},
			ruleSetLinkAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validateHTTPURL(ruleSetLinkAttr, urlIsAbs|urlOptional),
			},
			ruleSetMetricTypeAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      defaultRuleSetMetricType,
				ValidateFunc: validateStringIn(ruleSetMetricTypeAttr, validRuleSetMetricTypes),
			},
			ruleSetNotesAttr: &schema.Schema{
				Type:      schema.TypeString,
				Optional:  true,
				Computed:  true,
				StateFunc: suppressWhitespace,
			},
			ruleSetParentAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				StateFunc:    suppressWhitespace,
				ValidateFunc: validateRegexp(ruleSetParentAttr, `^[\d]+_[\d\w]+$`),
			},
			ruleSetMetricNameAttr: &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateRegexp(ruleSetMetricNameAttr, `^[\S]+$`),
			},
			ruleSetTagsAttr: tagMakeConfigSchema(ruleSetTagsAttr),
		}),
	}
}

func ruleSetCreate(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*providerContext)
	rs := newRuleSet()

	if err := rs.ParseConfig(d); err != nil {
		return errwrap.Wrapf("error parsing rule set schema during create: {{err}}", err)
	}

	if err := rs.Create(ctxt); err != nil {
		return errwrap.Wrapf("error creating rule set: {{err}}", err)
	}

	d.SetId(rs.CID)

	return ruleSetRead(d, meta)
}

func ruleSetExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	ctxt := meta.(*providerContext)

	cid := d.Id()
	rs, err := ctxt.client.FetchRuleSet(api.CIDType(&cid))
	if err != nil {
		return false, err
	}

	if rs.CID == "" {
		return false, nil
	}

	return true, nil
}

// ruleSetRead pulls data out of the RuleSet object and stores it into the
// appropriate place in the statefile.
func ruleSetRead(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*providerContext)

	cid := d.Id()
	rs, err := loadRuleSet(ctxt, api.CIDType(&cid))
	if err != nil {
		return err
	}

	d.SetId(rs.CID)

	ifRules := make([]interface{}, 0, defaultRuleSetRuleLen)
	for _, rule := range rs.Rules {
		ifAttrs := make(map[string]interface{}, 2)
		valueAttrs := make(map[string]interface{}, 2)
		valueOverAttrs := make(map[string]interface{}, 2)
		thenAttrs := make(map[string]interface{}, 3)

		switch rule.Criteria {
		case apiRuleSetAbsent:
			d, _ := time.ParseDuration(fmt.Sprintf("%fs", rule.Value.(float64)))
			valueAttrs[string(ruleSetAbsentAttr)] = fmt.Sprintf("%ds", int(d.Seconds()))
		case apiRuleSetChanged:
			valueAttrs[string(ruleSetChangedAttr)] = true
		case apiRuleSetContains:
			valueAttrs[string(ruleSetContainsAttr)] = rule.Value
		case apiRuleSetMatch:
			valueAttrs[string(ruleSetMatchAttr)] = rule.Value
		case apiRuleSetMaxValue:
			valueAttrs[string(ruleSetMaxValueAttr)] = rule.Value
		case apiRuleSetMinValue:
			valueAttrs[string(ruleSetMinValueAttr)] = rule.Value
		case apiRuleSetNotContains:
			valueAttrs[string(ruleSetNotContainAttr)] = rule.Value
		case apiRuleSetNotMatch:
			valueAttrs[string(ruleSetNotMatchAttr)] = rule.Value
		default:
			return fmt.Errorf("PROVIDER BUG: Unsupported criteria %q", rule.Criteria)
		}

		if rule.Wait > 0 {
			thenAttrs[string(ruleSetAfterAttr)] = fmt.Sprintf("%ds", 60*rule.Wait)
		}
		thenAttrs[string(ruleSetSeverityAttr)] = int(rule.Severity)

		if rule.WindowingFunction != nil {
			valueOverAttrs[string(ruleSetUsingAttr)] = *rule.WindowingFunction

			// NOTE: Only save the window duration if a function was specified
			valueOverAttrs[string(ruleSetLastAttr)] = fmt.Sprintf("%ds", rule.WindowingDuration)
		}
		valueOverSet := schema.NewSet(ruleSetValueOverChecksum, nil)
		valueOverSet.Add(valueOverAttrs)
		valueAttrs[string(ruleSetOverAttr)] = valueOverSet

		if contactGroups, ok := rs.ContactGroups[uint8(rule.Severity)]; ok {
			sort.Strings(contactGroups)
			thenAttrs[string(ruleSetNotifyAttr)] = contactGroups
		}
		thenSet := schema.NewSet(ruleSetThenChecksum, nil)
		thenSet.Add(thenAttrs)

		valueSet := schema.NewSet(ruleSetValueChecksum, nil)
		valueSet.Add(valueAttrs)
		ifAttrs[string(ruleSetThenAttr)] = thenSet
		ifAttrs[string(ruleSetValueAttr)] = valueSet

		ifRules = append(ifRules, ifAttrs)
	}

	d.Set(ruleSetCheckAttr, rs.CheckCID)

	if err := d.Set(ruleSetIfAttr, ifRules); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Unable to store rule set %q attribute: {{err}}", ruleSetIfAttr), err)
	}

	d.Set(ruleSetLinkAttr, indirect(rs.Link))
	d.Set(ruleSetMetricNameAttr, rs.MetricName)
	d.Set(ruleSetMetricTypeAttr, rs.MetricType)
	d.Set(ruleSetNotesAttr, indirect(rs.Notes))
	d.Set(ruleSetParentAttr, indirect(rs.Parent))

	if err := d.Set(ruleSetTagsAttr, tagsToState(apiToTags(rs.Tags))); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Unable to store rule set %q attribute: {{err}}", ruleSetTagsAttr), err)
	}

	return nil
}

func ruleSetUpdate(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*providerContext)
	rs := newRuleSet()

	if err := rs.ParseConfig(d); err != nil {
		return err
	}

	rs.CID = d.Id()
	if err := rs.Update(ctxt); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("unable to update rule set %q: {{err}}", d.Id()), err)
	}

	return ruleSetRead(d, meta)
}

func ruleSetDelete(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*providerContext)

	cid := d.Id()
	if _, err := ctxt.client.DeleteRuleSetByCID(api.CIDType(&cid)); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("unable to delete rule set %q: {{err}}", d.Id()), err)
	}

	d.SetId("")

	return nil
}

type circonusRuleSet struct {
	api.RuleSet
}

func newRuleSet() circonusRuleSet {
	rs := circonusRuleSet{
		RuleSet: *api.NewRuleSet(),
	}

	rs.ContactGroups = make(map[uint8][]string, config.NumSeverityLevels)
	for i := uint8(0); i < config.NumSeverityLevels; i++ {
		rs.ContactGroups[i+1] = make([]string, 0, 1)
	}

	rs.Rules = make([]api.RuleSetRule, 0, 1)

	return rs
}

func loadRuleSet(ctxt *providerContext, cid api.CIDType) (circonusRuleSet, error) {
	var rs circonusRuleSet
	crs, err := ctxt.client.FetchRuleSet(cid)
	if err != nil {
		return circonusRuleSet{}, err
	}
	rs.RuleSet = *crs

	return rs, nil
}

func ruleSetThenChecksum(v interface{}) int {
	b := &bytes.Buffer{}
	b.Grow(defaultHashBufSize)

	writeInt := func(m map[string]interface{}, attrName string) {
		if v, found := m[attrName]; found {
			i := v.(int)
			if i != 0 {
				fmt.Fprintf(b, "%x", i)
			}
		}
	}

	writeString := func(m map[string]interface{}, attrName string) {
		if v, found := m[attrName]; found {
			s := strings.TrimSpace(v.(string))
			if s != "" {
				fmt.Fprint(b, s)
			}
		}
	}

	writeStringArray := func(m map[string]interface{}, attrName string) {
		if v, found := m[attrName]; found {
			a := v.([]string)
			if a != nil {
				sort.Strings(a)
				for _, s := range a {
					fmt.Fprint(b, strings.TrimSpace(s))
				}
			}
		}
	}

	m := v.(map[string]interface{})

	writeString(m, ruleSetAfterAttr)
	writeStringArray(m, ruleSetNotifyAttr)
	writeInt(m, ruleSetSeverityAttr)

	s := b.String()
	return hashcode.String(s)
}

func ruleSetValueChecksum(v interface{}) int {
	b := &bytes.Buffer{}
	b.Grow(defaultHashBufSize)

	writeBool := func(m map[string]interface{}, attrName string) {
		if v, found := m[attrName]; found {
			fmt.Fprintf(b, "%t", v.(bool))
		}
	}

	writeDuration := func(m map[string]interface{}, attrName string) {
		if v, found := m[attrName]; found {
			s := v.(string)
			if s != "" {
				d, _ := time.ParseDuration(s)
				fmt.Fprint(b, d.String())
			}
		}
	}

	writeString := func(m map[string]interface{}, attrName string) {
		if v, found := m[attrName]; found {
			s := strings.TrimSpace(v.(string))
			if s != "" {
				fmt.Fprint(b, s)
			}
		}
	}

	m := v.(map[string]interface{})

	if v, found := m[ruleSetValueAttr]; found {
		valueMap := v.(map[string]interface{})
		if valueMap != nil {
			writeDuration(valueMap, ruleSetAbsentAttr)
			writeBool(valueMap, ruleSetChangedAttr)
			writeString(valueMap, ruleSetContainsAttr)
			writeString(valueMap, ruleSetMatchAttr)
			writeString(valueMap, ruleSetNotMatchAttr)
			writeString(valueMap, ruleSetMinValueAttr)
			writeString(valueMap, ruleSetNotContainAttr)
			writeString(valueMap, ruleSetMaxValueAttr)

			if v, found := valueMap[ruleSetOverAttr]; found {
				overMap := v.(map[string]interface{})
				writeDuration(overMap, ruleSetLastAttr)
				writeString(overMap, ruleSetUsingAttr)
			}
		}
	}

	s := b.String()
	return hashcode.String(s)
}

func ruleSetValueOverChecksum(v interface{}) int {
	b := &bytes.Buffer{}
	b.Grow(defaultHashBufSize)

	writeString := func(m map[string]interface{}, attrName string) {
		if v, found := m[attrName]; found {
			s := strings.TrimSpace(v.(string))
			if s != "" {
				fmt.Fprint(b, s)
			}
		}
	}

	m := v.(map[string]interface{})

	writeString(m, ruleSetLastAttr)
	writeString(m, ruleSetUsingAttr)

	s := b.String()
	return hashcode.String(s)
}

// ParseConfig reads Terraform config data and stores the information into a
// Circonus RuleSet object.  ParseConfig, ruleSetRead(), and ruleSetChecksum
// must be kept in sync.
func (rs *circonusRuleSet) ParseConfig(d *schema.ResourceData) error {
	if v, found := d.GetOk(ruleSetCheckAttr); found {
		rs.CheckCID = v.(string)
	}

	if v, found := d.GetOk(ruleSetLinkAttr); found {
		s := v.(string)
		rs.Link = &s
	}

	if v, found := d.GetOk(ruleSetMetricTypeAttr); found {
		rs.MetricType = v.(string)
	}

	if v, found := d.GetOk(ruleSetNotesAttr); found {
		s := v.(string)
		rs.Notes = &s
	}

	if v, found := d.GetOk(ruleSetParentAttr); found {
		s := v.(string)
		rs.Parent = &s
	}

	if v, found := d.GetOk(ruleSetMetricNameAttr); found {
		rs.MetricName = v.(string)
	}

	rs.Rules = make([]api.RuleSetRule, 0, defaultRuleSetRuleLen)
	if ifListRaw, found := d.GetOk(ruleSetIfAttr); found {
		ifList := ifListRaw.([]interface{})
		for _, ifListElem := range ifList {
			ifAttrs := newInterfaceMap(ifListElem.(map[string]interface{}))

			rule := api.RuleSetRule{}

			if thenListRaw, found := ifAttrs[ruleSetThenAttr]; found {
				thenList := thenListRaw.(*schema.Set).List()

				for _, thenListRaw := range thenList {
					thenAttrs := newInterfaceMap(thenListRaw)

					if v, found := thenAttrs[ruleSetAfterAttr]; found {
						s := v.(string)
						if s != "" {
							d, err := time.ParseDuration(v.(string))
							if err != nil {
								return errwrap.Wrapf(fmt.Sprintf("unable to parse %q duration %q: {{err}}", ruleSetAfterAttr, v.(string)), err)
							}
							rule.Wait = uint(d.Minutes())
						}
					}

					// NOTE: break from convention of alpha sorting attributes and handle Notify after Severity

					if i, found := thenAttrs[ruleSetSeverityAttr]; found {
						rule.Severity = uint(i.(int))
					}

					if notifyListRaw, found := thenAttrs[ruleSetNotifyAttr]; found {
						notifyList := interfaceList(notifyListRaw.([]interface{}))

						sev := uint8(rule.Severity)
						for _, contactGroupCID := range notifyList.List() {
							var found bool
							if contactGroups, ok := rs.ContactGroups[sev]; ok {
								for _, contactGroup := range contactGroups {
									if contactGroup == contactGroupCID {
										found = true
										break
									}
								}
							}
							if !found {
								rs.ContactGroups[sev] = append(rs.ContactGroups[sev], contactGroupCID)
							}
						}
					}
				}
			}

			if ruleSetValueListRaw, found := ifAttrs[ruleSetValueAttr]; found {
				ruleSetValueList := ruleSetValueListRaw.(*schema.Set).List()

				for _, valueListRaw := range ruleSetValueList {
					valueAttrs := newInterfaceMap(valueListRaw)

				METRIC_TYPE:
					switch rs.MetricType {
					case ruleSetMetricTypeNumeric:
						if v, found := valueAttrs[ruleSetAbsentAttr]; found {
							s := v.(string)
							if s != "" {
								d, _ := time.ParseDuration(s)
								rule.Criteria = apiRuleSetAbsent
								rule.Value = float64(d.Seconds())
								break METRIC_TYPE
							}
						}

						if v, found := valueAttrs[ruleSetChangedAttr]; found {
							b := v.(bool)
							if b {
								rule.Criteria = apiRuleSetChanged
								break METRIC_TYPE
							}
						}

						if v, found := valueAttrs[ruleSetMinValueAttr]; found {
							s := v.(string)
							if s != "" {
								rule.Criteria = apiRuleSetMinValue
								rule.Value = s
								break METRIC_TYPE
							}
						}

						if v, found := valueAttrs[ruleSetMaxValueAttr]; found {
							s := v.(string)
							if s != "" {
								rule.Criteria = apiRuleSetMaxValue
								rule.Value = s
								break METRIC_TYPE
							}
						}
					case ruleSetMetricTypeText:
						if v, found := valueAttrs[ruleSetAbsentAttr]; found {
							s := v.(string)
							if s != "" {
								d, _ := time.ParseDuration(s)
								rule.Criteria = apiRuleSetAbsent
								rule.Value = float64(d.Seconds())
								break METRIC_TYPE
							}
						}

						if v, found := valueAttrs[ruleSetChangedAttr]; found {
							b := v.(bool)
							if b {
								rule.Criteria = apiRuleSetChanged
								break METRIC_TYPE
							}
						}

						if v, found := valueAttrs[ruleSetContainsAttr]; found {
							s := v.(string)
							if s != "" {
								rule.Criteria = apiRuleSetContains
								rule.Value = s
								break METRIC_TYPE
							}
						}

						if v, found := valueAttrs[ruleSetMatchAttr]; found {
							s := v.(string)
							if s != "" {
								rule.Criteria = apiRuleSetMatch
								rule.Value = s
								break METRIC_TYPE
							}
						}

						if v, found := valueAttrs[ruleSetNotMatchAttr]; found {
							s := v.(string)
							if s != "" {
								rule.Criteria = apiRuleSetNotMatch
								rule.Value = s
								break METRIC_TYPE
							}
						}

						if v, found := valueAttrs[ruleSetNotContainAttr]; found {
							s := v.(string)
							if s != "" {
								rule.Criteria = apiRuleSetNotContains
								rule.Value = s
								break METRIC_TYPE
							}
						}
					default:
						return fmt.Errorf("PROVIDER BUG: unsupported rule set metric type: %q", rs.MetricType)
					}

					if ruleSetOverListRaw, found := valueAttrs[ruleSetOverAttr]; found {
						overList := ruleSetOverListRaw.(*schema.Set).List()

						for _, overListRaw := range overList {
							overAttrs := newInterfaceMap(overListRaw)

							if v, found := overAttrs[ruleSetLastAttr]; found {
								last, err := time.ParseDuration(v.(string))
								if err != nil {
									return errwrap.Wrapf(fmt.Sprintf("unable to parse duration %s attribute", ruleSetLastAttr), err)
								}
								rule.WindowingDuration = uint(last.Seconds())
							}

							if v, found := overAttrs[ruleSetUsingAttr]; found {
								s := v.(string)
								rule.WindowingFunction = &s
							}
						}
					}
				}
			}
			rs.Rules = append(rs.Rules, rule)
		}
	}

	if v, found := d.GetOk(ruleSetTagsAttr); found {
		rs.Tags = derefStringList(flattenSet(v.(*schema.Set)))
	}

	if err := rs.Validate(); err != nil {
		return err
	}

	return nil
}

func (rs *circonusRuleSet) Create(ctxt *providerContext) error {
	crs, err := ctxt.client.CreateRuleSet(&rs.RuleSet)
	if err != nil {
		return err
	}

	rs.CID = crs.CID

	return nil
}

func (rs *circonusRuleSet) Update(ctxt *providerContext) error {
	_, err := ctxt.client.UpdateRuleSet(&rs.RuleSet)
	if err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Unable to update rule set %s: {{err}}", rs.CID), err)
	}

	return nil
}

func (rs *circonusRuleSet) Validate() error {
	// TODO(sean@): From https://login.circonus.com/resources/api/calls/rule_set
	// under `value`:
	//
	// For an 'on absence' rule this is the number of seconds the metric must not
	// have been collected for, and should not be lower than either the period or
	// timeout of the metric being collected.

	for i, rule := range rs.Rules {
		if rule.Criteria == "" {
			return fmt.Errorf("rule %d for check ID %s has an empty criteria", i, rs.CheckCID)
		}
	}

	return nil
}
