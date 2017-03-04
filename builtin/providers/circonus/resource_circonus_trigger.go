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
	// circonus_trigger.* resource attribute names
	triggerCheckAttr      = "check"
	triggerIfAttr         = "if"
	triggerLinkAttr       = "link"
	triggerMetricTypeAttr = "metric_type"
	triggerNotesAttr      = "notes"
	triggerParentAttr     = "parent"
	triggerStreamNameAttr = "stream_name"
	triggerTagsAttr       = "tags"

	// circonus_trigger.if.* resource attribute names
	triggerThenAttr  = "then"
	triggerValueAttr = "value"

	// circonus_trigger.if.then.* resource attribute names
	triggerAfterAttr    = "after"
	triggerNotifyAttr   = "notify"
	triggerSeverityAttr = "severity"

	// circonus_trigger.if.value.* resource attribute names
	triggerAbsentAttr   = "absent"   // apiRulesetAbsent
	triggerChangedAttr  = "changed"  // apiRulesetChanged
	triggerContainsAttr = "contains" // apiRulesetContains
	triggerEqualsAttr   = "equals"   // apiRulesetMatch
	triggerExcludesAttr = "excludes" // apiRulesetNotMatch
	triggerLessAttr     = "less"     // apiRulesetMinValue
	triggerMissingAttr  = "missing"  // apiRulesetNotContains
	triggerMoreAttr     = "more"     // apiRulesetMaxValue
	triggerOverAttr     = "over"

	// circonus_trigger.if.value.over.* resource attribute names
	triggerLastAttr  = "last"
	triggerUsingAttr = "using"
)

const (
	// Different criteria that an api.RuleSetRule can return
	apiRulesetAbsent      = "on absence"       // triggerAbsentAttr
	apiRulesetChanged     = "on change"        // triggerChangedAttr
	apiRulesetContains    = "contains"         // triggerContainsAttr
	apiRulesetMatch       = "match"            // triggerEqualsAttr
	apiRulesetMaxValue    = "max value"        // triggerMoreAttr
	apiRulesetMinValue    = "min value"        // triggerLessAttr
	apiRulesetNotContains = "does not contain" // triggerExcludesAttr
	apiRulesetNotMatch    = "does not match"   // triggerMissingAttr
)

var triggerDescriptions = attrDescrs{
	// circonus_trigger.* resource attribute names
	triggerCheckAttr:      "The CID of the check that contains the stream for this trigger",
	triggerIfAttr:         "A rule to execute for this trigger",
	triggerLinkAttr:       "URL to show users when this trigger is active (e.g. wiki)",
	triggerMetricTypeAttr: "The type of data flowing through the specified stream",
	triggerNotesAttr:      "Notes describing this trigger",
	triggerParentAttr:     "Parent CID that must be healthy for this trigger to be active",
	triggerStreamNameAttr: "The name of the stream within a check to register the trigger with",
	triggerTagsAttr:       "Tags associated with this trigger",
}

var triggerIfDescriptions = attrDescrs{
	// circonus_trigger.if.* resource attribute names
	triggerThenAttr:  "Description of the action(s) to take when this trigger is active",
	triggerValueAttr: "Predicate that the trigger uses to evaluate a stream of metrics",
}

var triggerIfValueDescriptions = attrDescrs{
	// circonus_trigger.if.value.* resource attribute names
	triggerAbsentAttr:   "Fire the trigger if there has been no data for the given stream over the last duration",
	triggerChangedAttr:  "Boolean indicating the value has changed",
	triggerContainsAttr: "Fire the trigger if the text metric contain the following string",
	triggerEqualsAttr:   "Fire the trigger if the text metric exactly match the following string",
	triggerExcludesAttr: "Fire the trigger if the text metric not match the following string",
	triggerLessAttr:     "Fire the trigger if the numeric value less than the specified value",
	triggerMissingAttr:  "Fire the trigger if the text metric does not contain the following string",
	triggerMoreAttr:     "Fire the trigger if the numeric value is more than the specified value",
	triggerOverAttr:     "Use a derived value using a window",
	triggerThenAttr:     "Action to take when the trigger is active",
}

var triggerIfValueOverDescriptions = attrDescrs{
	// circonus_trigger.if.value.over.* resource attribute names
	triggerLastAttr:  "Duration over which data from the last interval is examined",
	triggerUsingAttr: "Define the window funciton to use over the last duration",
}

var triggerIfThenDescriptions = attrDescrs{
	// circonus_trigger.if.then.* resource attribute names
	triggerAfterAttr:    "The length of time we should wait before contacting the contact groups after this ruleset has faulted.",
	triggerNotifyAttr:   "List of contact groups to notify at the following appropriate severity if this trigger is active.",
	triggerSeverityAttr: "Send a notification at this severity level.",
}

func newTriggerResource() *schema.Resource {
	makeConflictsWith := func(in ...schemaAttr) []string {
		out := make([]string, 0, len(in))
		for _, attr := range in {
			out = append(out, string(triggerIfAttr)+"."+string(triggerValueAttr)+"."+string(attr))
		}
		return out
	}

	return &schema.Resource{
		Create: triggerCreate,
		Read:   triggerRead,
		Update: triggerUpdate,
		Delete: triggerDelete,
		Exists: triggerExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: convertToHelperSchema(triggerDescriptions, map[schemaAttr]*schema.Schema{
			triggerCheckAttr: &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateRegexp(triggerCheckAttr, config.CheckCIDRegex),
			},
			triggerIfAttr: &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: convertToHelperSchema(triggerIfDescriptions, map[schemaAttr]*schema.Schema{
						triggerThenAttr: &schema.Schema{
							Type:     schema.TypeSet,
							MaxItems: 1,
							Optional: true,
							Elem: &schema.Resource{
								Schema: convertToHelperSchema(triggerIfThenDescriptions, map[schemaAttr]*schema.Schema{
									triggerAfterAttr: &schema.Schema{
										Type:             schema.TypeString,
										Optional:         true,
										DiffSuppressFunc: suppressEquivalentTimeDurations,
										StateFunc:        normalizeTimeDurationStringToSeconds,
										ValidateFunc: validateFuncs(
											validateDurationMin(triggerAfterAttr, "0s"),
										),
									},
									triggerNotifyAttr: &schema.Schema{
										Type:     schema.TypeList,
										Optional: true,
										MinItems: 1,
										Elem: &schema.Schema{
											Type:         schema.TypeString,
											ValidateFunc: validateContactGroupCID(triggerNotifyAttr),
										},
									},
									triggerSeverityAttr: &schema.Schema{
										Type:     schema.TypeInt,
										Optional: true,
										Default:  defaultTriggerSeverity,
										ValidateFunc: validateFuncs(
											validateIntMax(triggerSeverityAttr, maxSeverity),
											validateIntMin(triggerSeverityAttr, minSeverity),
										),
									},
								}),
							},
						},
						triggerValueAttr: &schema.Schema{
							Type:     schema.TypeSet,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: convertToHelperSchema(triggerIfValueDescriptions, map[schemaAttr]*schema.Schema{
									triggerAbsentAttr: &schema.Schema{
										Type:             schema.TypeString, // Applies to text or numeric metrics
										Optional:         true,
										DiffSuppressFunc: suppressEquivalentTimeDurations,
										StateFunc:        normalizeTimeDurationStringToSeconds,
										ValidateFunc: validateFuncs(
											validateDurationMin(triggerAbsentAttr, triggerAbsentMin),
										),
										ConflictsWith: makeConflictsWith(triggerChangedAttr, triggerContainsAttr, triggerEqualsAttr, triggerExcludesAttr, triggerLessAttr, triggerMissingAttr, triggerMoreAttr, triggerOverAttr),
									},
									triggerChangedAttr: &schema.Schema{
										Type:          schema.TypeBool, // Applies to text or numeric metrics
										Optional:      true,
										ConflictsWith: makeConflictsWith(triggerAbsentAttr, triggerContainsAttr, triggerEqualsAttr, triggerExcludesAttr, triggerLessAttr, triggerMissingAttr, triggerMoreAttr, triggerOverAttr),
									},
									triggerContainsAttr: &schema.Schema{
										Type:          schema.TypeString, // Applies to text metrics only
										Optional:      true,
										ValidateFunc:  validateRegexp(triggerContainsAttr, `.+`),
										ConflictsWith: makeConflictsWith(triggerAbsentAttr, triggerChangedAttr, triggerEqualsAttr, triggerExcludesAttr, triggerLessAttr, triggerMissingAttr, triggerMoreAttr, triggerOverAttr),
									},
									triggerEqualsAttr: &schema.Schema{
										Type:          schema.TypeString, // Applies to text metrics only
										Optional:      true,
										ValidateFunc:  validateRegexp(triggerEqualsAttr, `.+`),
										ConflictsWith: makeConflictsWith(triggerAbsentAttr, triggerChangedAttr, triggerContainsAttr, triggerExcludesAttr, triggerLessAttr, triggerMissingAttr, triggerMoreAttr, triggerOverAttr),
									},
									triggerExcludesAttr: &schema.Schema{
										Type:          schema.TypeString, // Applies to text metrics only
										Optional:      true,
										ValidateFunc:  validateRegexp(triggerExcludesAttr, `.+`),
										ConflictsWith: makeConflictsWith(triggerAbsentAttr, triggerChangedAttr, triggerContainsAttr, triggerEqualsAttr, triggerLessAttr, triggerMissingAttr, triggerMoreAttr, triggerOverAttr),
									},
									triggerLessAttr: &schema.Schema{
										Type:          schema.TypeString, // Applies to numeric metrics only
										Optional:      true,
										ValidateFunc:  validateRegexp(triggerLessAttr, `.+`), // TODO(sean): improve this regexp to match int and float
										ConflictsWith: makeConflictsWith(triggerAbsentAttr, triggerChangedAttr, triggerContainsAttr, triggerEqualsAttr, triggerExcludesAttr, triggerMissingAttr, triggerMoreAttr),
									},
									triggerMissingAttr: &schema.Schema{
										Type:          schema.TypeString, // Applies to text metrics only
										Optional:      true,
										ValidateFunc:  validateRegexp(triggerMissingAttr, `.+`),
										ConflictsWith: makeConflictsWith(triggerAbsentAttr, triggerChangedAttr, triggerContainsAttr, triggerEqualsAttr, triggerExcludesAttr, triggerLessAttr, triggerMoreAttr, triggerOverAttr),
									},
									triggerMoreAttr: &schema.Schema{
										Type:          schema.TypeString, // Applies to numeric metrics only
										Optional:      true,
										ValidateFunc:  validateRegexp(triggerMoreAttr, `.+`), // TODO(sean): improve this regexp to match int and float
										ConflictsWith: makeConflictsWith(triggerAbsentAttr, triggerChangedAttr, triggerContainsAttr, triggerEqualsAttr, triggerExcludesAttr, triggerLessAttr, triggerMissingAttr),
									},
									triggerOverAttr: &schema.Schema{
										Type:     schema.TypeSet,
										Optional: true,
										MaxItems: 1,
										// triggerOverAttr is only compatible with checks of
										// numeric type.  NOTE: It may be premature to conflict with
										// triggerChangedAttr.
										ConflictsWith: makeConflictsWith(triggerAbsentAttr, triggerChangedAttr, triggerContainsAttr, triggerEqualsAttr, triggerExcludesAttr, triggerMissingAttr),
										Elem: &schema.Resource{
											Schema: convertToHelperSchema(triggerIfValueOverDescriptions, map[schemaAttr]*schema.Schema{
												triggerLastAttr: &schema.Schema{
													Type:             schema.TypeString,
													Optional:         true,
													Default:          defaultTriggerLast,
													DiffSuppressFunc: suppressEquivalentTimeDurations,
													StateFunc:        normalizeTimeDurationStringToSeconds,
													ValidateFunc: validateFuncs(
														validateDurationMin(triggerLastAttr, "0s"),
													),
												},
												triggerUsingAttr: &schema.Schema{
													Type:         schema.TypeString,
													Optional:     true,
													Default:      defaultTriggerWindowFunc,
													ValidateFunc: validateStringIn(triggerUsingAttr, validTriggerWindowFuncs),
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
			triggerLinkAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validateHTTPURL(triggerLinkAttr, urlIsAbs),
			},
			triggerMetricTypeAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      defaultTriggerMetricType,
				ValidateFunc: validateStringIn(triggerMetricTypeAttr, validTriggerMetricTypes),
			},
			triggerNotesAttr: &schema.Schema{
				Type:      schema.TypeString,
				Optional:  true,
				Computed:  true,
				StateFunc: suppressWhitespace,
			},
			triggerParentAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				StateFunc:    suppressWhitespace,
				ValidateFunc: validateRegexp(triggerParentAttr, `^[\d]+_[\d\w]+$`),
			},
			triggerStreamNameAttr: &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateRegexp(triggerStreamNameAttr, `^[\S]+$`),
			},
			triggerTagsAttr: tagMakeConfigSchema(triggerTagsAttr),
		}),
	}
}

func triggerCreate(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*providerContext)
	t := newTrigger()

	if err := t.ParseConfig(d); err != nil {
		return errwrap.Wrapf("error parsing trigger schema during create: {{err}}", err)
	}

	if err := t.Create(ctxt); err != nil {
		return errwrap.Wrapf("error creating trigger: {{err}}", err)
	}

	d.SetId(t.CID)

	return triggerRead(d, meta)
}

func triggerExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	ctxt := meta.(*providerContext)

	cid := d.Id()
	t, err := ctxt.client.FetchRuleSet(api.CIDType(&cid))
	if err != nil {
		return false, err
	}

	if t.CID == "" {
		return false, nil
	}

	return true, nil
}

// triggerRead pulls data out of the RuleSet object and stores it into the
// appropriate place in the statefile.
func triggerRead(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*providerContext)

	cid := d.Id()
	t, err := loadTrigger(ctxt, api.CIDType(&cid))
	if err != nil {
		return err
	}

	d.SetId(t.CID)

	ifRules := make([]interface{}, 0, defaultTriggerRuleLen)
	for _, rule := range t.Rules {
		ifAttrs := make(map[string]interface{}, 2)
		valueAttrs := make(map[string]interface{}, 2)
		valueOverAttrs := make(map[string]interface{}, 2)
		thenAttrs := make(map[string]interface{}, 3)

		switch rule.Criteria {
		case apiRulesetAbsent:
			d, _ := time.ParseDuration(fmt.Sprintf("%fs", rule.Value.(float64)))
			valueAttrs[string(triggerAbsentAttr)] = fmt.Sprintf("%ds", int(d.Seconds()))
		case apiRulesetChanged:
			valueAttrs[string(triggerChangedAttr)] = true
		case apiRulesetContains:
			valueAttrs[string(triggerContainsAttr)] = rule.Value
		case apiRulesetMatch:
			valueAttrs[string(triggerEqualsAttr)] = rule.Value
		case apiRulesetMaxValue:
			valueAttrs[string(triggerMoreAttr)] = rule.Value
		case apiRulesetMinValue:
			valueAttrs[string(triggerLessAttr)] = rule.Value
		case apiRulesetNotContains:
			valueAttrs[string(triggerExcludesAttr)] = rule.Value
		case apiRulesetNotMatch:
			valueAttrs[string(triggerMissingAttr)] = rule.Value
		default:
			return fmt.Errorf("PROVIDER BUG: Unsupported criteria %q", rule.Criteria)
		}

		if rule.Wait > 0 {
			thenAttrs[string(triggerAfterAttr)] = fmt.Sprintf("%ds", 60*rule.Wait)
		}
		thenAttrs[string(triggerSeverityAttr)] = int(rule.Severity)

		if rule.WindowingFunction != nil {
			valueOverAttrs[string(triggerUsingAttr)] = *rule.WindowingFunction

			// NOTE: Only save the window duration if a function was specified
			valueOverAttrs[string(triggerLastAttr)] = fmt.Sprintf("%ds", rule.WindowingDuration)
		}
		valueOverSet := schema.NewSet(triggerValueOverChecksum, nil)
		valueOverSet.Add(valueOverAttrs)
		valueAttrs[string(triggerOverAttr)] = valueOverSet

		if contactGroups, ok := t.ContactGroups[uint8(rule.Severity)]; ok {
			sort.Strings(contactGroups)
			thenAttrs[string(triggerNotifyAttr)] = contactGroups
		}
		thenSet := schema.NewSet(triggerThenChecksum, nil)
		thenSet.Add(thenAttrs)

		valueSet := schema.NewSet(triggerValueChecksum, nil)
		valueSet.Add(valueAttrs)
		ifAttrs[string(triggerThenAttr)] = thenSet
		ifAttrs[string(triggerValueAttr)] = valueSet

		ifRules = append(ifRules, ifAttrs)
	}

	d.Set(triggerCheckAttr, t.CheckCID)

	if err := d.Set(triggerIfAttr, ifRules); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Unable to store trigger %q attribute: {{err}}", triggerIfAttr), err)
	}

	d.Set(triggerLinkAttr, indirect(t.Link))
	d.Set(triggerStreamNameAttr, t.MetricName)
	d.Set(triggerMetricTypeAttr, t.MetricType)
	d.Set(triggerNotesAttr, indirect(t.Notes))
	d.Set(triggerParentAttr, indirect(t.Parent))

	if err := d.Set(triggerTagsAttr, tagsToState(apiToTags(t.Tags))); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Unable to store trigger %q attribute: {{err}}", triggerTagsAttr), err)
	}

	return nil
}

func triggerUpdate(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*providerContext)
	t := newTrigger()

	if err := t.ParseConfig(d); err != nil {
		return err
	}

	t.CID = d.Id()
	if err := t.Update(ctxt); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("unable to update trigger %q: {{err}}", d.Id()), err)
	}

	return triggerRead(d, meta)
}

func triggerDelete(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*providerContext)

	cid := d.Id()
	if _, err := ctxt.client.DeleteRuleSetByCID(api.CIDType(&cid)); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("unable to delete trigger %q: {{err}}", d.Id()), err)
	}

	d.SetId("")

	return nil
}

type circonusTrigger struct {
	api.RuleSet
}

func newTrigger() circonusTrigger {
	t := circonusTrigger{
		RuleSet: *api.NewRuleSet(),
	}

	t.ContactGroups = make(map[uint8][]string, config.NumSeverityLevels)
	for i := uint8(0); i < config.NumSeverityLevels; i++ {
		t.ContactGroups[i+1] = make([]string, 0, 1)
	}

	t.Rules = make([]api.RuleSetRule, 0, 1)

	return t
}

func loadTrigger(ctxt *providerContext, cid api.CIDType) (circonusTrigger, error) {
	var t circonusTrigger
	rs, err := ctxt.client.FetchRuleSet(cid)
	if err != nil {
		return circonusTrigger{}, err
	}
	t.RuleSet = *rs

	return t, nil
}

func triggerThenChecksum(v interface{}) int {
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

	writeString(m, triggerAfterAttr)
	writeStringArray(m, triggerNotifyAttr)
	writeInt(m, triggerSeverityAttr)

	s := b.String()
	return hashcode.String(s)
}

func triggerValueChecksum(v interface{}) int {
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

	if v, found := m[triggerValueAttr]; found {
		valueMap := v.(map[string]interface{})
		if valueMap != nil {
			writeDuration(valueMap, triggerAbsentAttr)
			writeBool(valueMap, triggerChangedAttr)
			writeString(valueMap, triggerContainsAttr)
			writeString(valueMap, triggerEqualsAttr)
			writeString(valueMap, triggerExcludesAttr)
			writeString(valueMap, triggerLessAttr)
			writeString(valueMap, triggerMissingAttr)
			writeString(valueMap, triggerMoreAttr)

			if v, found := valueMap[triggerOverAttr]; found {
				overMap := v.(map[string]interface{})
				writeDuration(overMap, triggerLastAttr)
				writeString(overMap, triggerUsingAttr)
			}
		}
	}

	s := b.String()
	return hashcode.String(s)
}

func triggerValueOverChecksum(v interface{}) int {
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

	writeString(m, triggerLastAttr)
	writeString(m, triggerUsingAttr)

	s := b.String()
	return hashcode.String(s)
}

// ParseConfig reads Terraform config data and stores the information into a
// Circonus RuleSet object.  ParseConfig, triggerRead(), and triggerChecksum
// must be kept in sync.
func (t *circonusTrigger) ParseConfig(d *schema.ResourceData) error {
	if v, found := d.GetOk(triggerCheckAttr); found {
		t.CheckCID = v.(string)
	}

	if v, found := d.GetOk(triggerLinkAttr); found {
		s := v.(string)
		t.Link = &s
	}

	if v, found := d.GetOk(triggerMetricTypeAttr); found {
		t.MetricType = v.(string)
	}

	if v, found := d.GetOk(triggerNotesAttr); found {
		s := v.(string)
		t.Notes = &s
	}

	if v, found := d.GetOk(triggerParentAttr); found {
		s := v.(string)
		t.Parent = &s
	}

	if v, found := d.GetOk(triggerStreamNameAttr); found {
		t.MetricName = v.(string)
	}

	t.Rules = make([]api.RuleSetRule, 0, defaultTriggerRuleLen)
	if ifListRaw, found := d.GetOk(triggerIfAttr); found {
		ifList := ifListRaw.([]interface{})
		for _, ifListElem := range ifList {
			ifAttrs := newInterfaceMap(ifListElem.(map[string]interface{}))

			rule := api.RuleSetRule{}

			if thenListRaw, found := ifAttrs[triggerThenAttr]; found {
				thenList := thenListRaw.(*schema.Set).List()

				for _, thenListRaw := range thenList {
					thenAttrs := newInterfaceMap(thenListRaw)

					if v, found := thenAttrs[triggerAfterAttr]; found {
						s := v.(string)
						if s != "" {
							d, err := time.ParseDuration(v.(string))
							if err != nil {
								return errwrap.Wrapf(fmt.Sprintf("unable to parse %q duration %q: {{err}}", triggerAfterAttr, v.(string)), err)
							}
							rule.Wait = uint(d.Minutes())
						}
					}

					// NOTE: break from convention of alpha sorting attributes and handle Notify after Severity

					if i, found := thenAttrs[triggerSeverityAttr]; found {
						rule.Severity = uint(i.(int))
					}

					if notifyListRaw, found := thenAttrs[triggerNotifyAttr]; found {
						notifyList := interfaceList(notifyListRaw.([]interface{}))

						sev := uint8(rule.Severity)
						for _, contactGroupCID := range notifyList.List() {
							var found bool
							if contactGroups, ok := t.ContactGroups[sev]; ok {
								for _, contactGroup := range contactGroups {
									if contactGroup == contactGroupCID {
										found = true
										break
									}
								}
							}
							if !found {
								t.ContactGroups[sev] = append(t.ContactGroups[sev], contactGroupCID)
							}
						}
					}
				}
			}

			if triggerValueListRaw, found := ifAttrs[triggerValueAttr]; found {
				triggerValueList := triggerValueListRaw.(*schema.Set).List()

				for _, valueListRaw := range triggerValueList {
					valueAttrs := newInterfaceMap(valueListRaw)

				METRIC_TYPE:
					switch t.MetricType {
					case triggerMetricTypeNumeric:
						if v, found := valueAttrs[triggerAbsentAttr]; found {
							s := v.(string)
							if s != "" {
								d, _ := time.ParseDuration(s)
								rule.Criteria = apiRulesetAbsent
								rule.Value = float64(d.Seconds())
								break METRIC_TYPE
							}
						}

						if v, found := valueAttrs[triggerChangedAttr]; found {
							b := v.(bool)
							if b {
								rule.Criteria = apiRulesetChanged
								break METRIC_TYPE
							}
						}

						if v, found := valueAttrs[triggerLessAttr]; found {
							s := v.(string)
							if s != "" {
								rule.Criteria = apiRulesetMinValue
								rule.Value = s
								break METRIC_TYPE
							}
						}

						if v, found := valueAttrs[triggerMoreAttr]; found {
							s := v.(string)
							if s != "" {
								rule.Criteria = apiRulesetMaxValue
								rule.Value = s
								break METRIC_TYPE
							}
						}
					case triggerMetricTypeText:
						if v, found := valueAttrs[triggerAbsentAttr]; found {
							s := v.(string)
							if s != "" {
								d, _ := time.ParseDuration(s)
								rule.Criteria = apiRulesetAbsent
								rule.Value = float64(d.Seconds())
								break METRIC_TYPE
							}
						}

						if v, found := valueAttrs[triggerChangedAttr]; found {
							b := v.(bool)
							if b {
								rule.Criteria = apiRulesetChanged
								break METRIC_TYPE
							}
						}

						if v, found := valueAttrs[triggerContainsAttr]; found {
							s := v.(string)
							if s != "" {
								rule.Criteria = apiRulesetContains
								rule.Value = s
								break METRIC_TYPE
							}
						}

						if v, found := valueAttrs[triggerEqualsAttr]; found {
							s := v.(string)
							if s != "" {
								rule.Criteria = apiRulesetMatch
								rule.Value = s
								break METRIC_TYPE
							}
						}

						if v, found := valueAttrs[triggerExcludesAttr]; found {
							s := v.(string)
							if s != "" {
								rule.Criteria = apiRulesetNotMatch
								rule.Value = s
								break METRIC_TYPE
							}
						}

						if v, found := valueAttrs[triggerMissingAttr]; found {
							s := v.(string)
							if s != "" {
								rule.Criteria = apiRulesetNotContains
								rule.Value = s
								break METRIC_TYPE
							}
						}
					default:
						return fmt.Errorf("PROVIDER BUG: unsupported trigger metric type: %q", t.MetricType)
					}

					if triggerOverListRaw, found := valueAttrs[triggerOverAttr]; found {
						overList := triggerOverListRaw.(*schema.Set).List()

						for _, overListRaw := range overList {
							overAttrs := newInterfaceMap(overListRaw)

							if v, found := overAttrs[triggerLastAttr]; found {
								last, err := time.ParseDuration(v.(string))
								if err != nil {
									return errwrap.Wrapf(fmt.Sprintf("unable to parse duration %s attribute", triggerLastAttr), err)
								}
								rule.WindowingDuration = uint(last.Seconds())
							}

							if v, found := overAttrs[triggerUsingAttr]; found {
								s := v.(string)
								rule.WindowingFunction = &s
							}
						}
					}
				}
			}
			t.Rules = append(t.Rules, rule)
		}
	}

	if v, found := d.GetOk(triggerTagsAttr); found {
		t.Tags = derefStringList(flattenSet(v.(*schema.Set)))
	}

	if err := t.Validate(); err != nil {
		return err
	}

	return nil
}

func (t *circonusTrigger) Create(ctxt *providerContext) error {
	rs, err := ctxt.client.CreateRuleSet(&t.RuleSet)
	if err != nil {
		return err
	}

	t.CID = rs.CID

	return nil
}

func (t *circonusTrigger) Update(ctxt *providerContext) error {
	_, err := ctxt.client.UpdateRuleSet(&t.RuleSet)
	if err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Unable to update trigger %s: {{err}}", t.CID), err)
	}

	return nil
}

func (t *circonusTrigger) Validate() error {
	// TODO(sean@): From https://login.circonus.com/resources/api/calls/rule_set
	// under `value`:
	//
	// For an 'on absence' rule this is the number of seconds the metric must not
	// have been collected for, and should not be lower than either the period or
	// timeout of the metric being collected.
	return nil
}
