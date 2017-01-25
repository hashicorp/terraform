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
	_TriggerCheckAttr      _SchemaAttr = "check"
	_TriggerIfAttr         _SchemaAttr = "if"
	_TriggerLinkAttr       _SchemaAttr = "link"
	_TriggerMetricTypeAttr _SchemaAttr = "metric_type"
	_TriggerNotesAttr      _SchemaAttr = "notes"
	_TriggerParentAttr     _SchemaAttr = "parent"
	_TriggerStreamNameAttr _SchemaAttr = "stream_name"
	_TriggerTagsAttr       _SchemaAttr = "tags"

	// circonus_trigger.if.* resource attribute names
	_TriggerThenAttr  _SchemaAttr = "then"
	_TriggerValueAttr _SchemaAttr = "value"

	// circonus_trigger.if.then.* resource attribute names
	_TriggerAfterAttr    _SchemaAttr = "after"
	_TriggerNotifyAttr   _SchemaAttr = "notify"
	_TriggerSeverityAttr _SchemaAttr = "severity"

	// circonus_trigger.if.value.* resource attribute names
	_TriggerAbsentAttr   _SchemaAttr = "absent"   // _APIRulesetAbsent
	_TriggerChangedAttr  _SchemaAttr = "changed"  // _APIRulesetChanged
	_TriggerContainsAttr _SchemaAttr = "contains" // _APIRulesetContains
	_TriggerEqualsAttr   _SchemaAttr = "equals"   // _APIRulesetMatch
	_TriggerExcludesAttr _SchemaAttr = "excludes" // _APIRulesetNotMatch
	_TriggerLessAttr     _SchemaAttr = "less"     // _APIRulesetMinValue
	_TriggerMissingAttr  _SchemaAttr = "missing"  // _APIRulesetNotContains
	_TriggerMoreAttr     _SchemaAttr = "more"     // _APIRulesetMaxValue
	_TriggerOverAttr     _SchemaAttr = "over"

	// circonus_trigger.if.value.over.* resource attribute names
	_TriggerLastAttr  _SchemaAttr = "last"
	_TriggerUsingAttr _SchemaAttr = "using"
)

const (
	// Different criteria that an api.RuleSetRule can return
	_APIRulesetAbsent      = "on absence"       // _TriggerAbsentAttr
	_APIRulesetChanged     = "on change"        // _TriggerChangedAttr
	_APIRulesetContains    = "contains"         // _TriggerContainsAttr
	_APIRulesetMatch       = "match"            // _TriggerEqualsAttr
	_APIRulesetMaxValue    = "max value"        // _TriggerMoreAttr
	_APIRulesetMinValue    = "min value"        // _TriggerLessAttr
	_APIRulesetNotContains = "does not contain" // _TriggerExcludesAttr
	_APIRulesetNotMatch    = "does not match"   // _TriggerMissingAttr
)

var _TriggerDescriptions = _AttrDescrs{
	// circonus_trigger.* resource attribute names
	_TriggerCheckAttr:      "The CID of the check that contains the stream for this trigger",
	_TriggerIfAttr:         "A rule to execute for this trigger",
	_TriggerLinkAttr:       "URL to show users when this trigger is active (e.g. wiki)",
	_TriggerMetricTypeAttr: "The type of data flowing through the specified stream",
	_TriggerNotesAttr:      "Notes describing this trigger",
	_TriggerParentAttr:     "Parent CID that must be healthy for this trigger to be active",
	_TriggerStreamNameAttr: "The name of the stream within a check to register the trigger with",
	_TriggerTagsAttr:       "Tags associated with this trigger",
}

var _TriggerIfDescriptions = _AttrDescrs{
	// circonus_trigger.if.* resource attribute names
	_TriggerThenAttr:  "Description of the action(s) to take when this trigger is active",
	_TriggerValueAttr: "Predicate that the trigger uses to evaluate a stream of metrics",
}

var _TriggerIfValueDescriptions = _AttrDescrs{
	// circonus_trigger.if.value.* resource attribute names
	_TriggerAbsentAttr:   "Fire the trigger if there has been no data for the given stream over the last duration",
	_TriggerChangedAttr:  "Boolean indicating the value has changed",
	_TriggerContainsAttr: "Fire the trigger if the text metric contain the following string",
	_TriggerEqualsAttr:   "Fire the trigger if the text metric exactly match the following string",
	_TriggerExcludesAttr: "Fire the trigger if the text metric not match the following string",
	_TriggerLessAttr:     "Fire the trigger if the numeric value less than the specified value",
	_TriggerMissingAttr:  "Fire the trigger if the text metric does not contain the following string",
	_TriggerMoreAttr:     "Fire the trigger if the numeric value is more than the specified value",
	_TriggerOverAttr:     "Use a derived value using a window",
	_TriggerThenAttr:     "Action to take when the trigger is active",
}

var _TriggerIfValueOverDescriptions = _AttrDescrs{
	// circonus_trigger.if.value.over.* resource attribute names
	_TriggerLastAttr:  "Duration over which data from the last interval is examined",
	_TriggerUsingAttr: "Define the window funciton to use over the last duration",
}

var _TriggerIfThenDescriptions = _AttrDescrs{
	// circonus_trigger.if.then.* resource attribute names
	_TriggerAfterAttr:    "The length of time we should wait before contacting the contact groups after this ruleset has faulted.",
	_TriggerNotifyAttr:   "List of contact groups to notify at the following appropriate severity if this trigger is active.",
	_TriggerSeverityAttr: "Send a notification at this severity level.",
}

func _NewTriggerResource() *schema.Resource {
	makeConflictsWith := func(in ..._SchemaAttr) []string {
		out := make([]string, 0, len(in))
		for _, attr := range in {
			out = append(out, string(_TriggerIfAttr)+"."+string(_TriggerValueAttr)+"."+string(attr))
		}
		return out
	}

	return &schema.Resource{
		Create: _TriggerCreate,
		Read:   _TriggerRead,
		Update: _TriggerUpdate,
		Delete: _TriggerDelete,
		Exists: _TriggerExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: _CastSchemaToTF(map[_SchemaAttr]*schema.Schema{
			_TriggerCheckAttr: &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: _ValidateRegexp(_TriggerCheckAttr, config.CheckCIDRegex),
			},
			_TriggerIfAttr: &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: _CastSchemaToTF(map[_SchemaAttr]*schema.Schema{
						_TriggerThenAttr: &schema.Schema{
							Type:     schema.TypeSet,
							MaxItems: 1,
							Optional: true,
							Elem: &schema.Resource{
								Schema: _CastSchemaToTF(map[_SchemaAttr]*schema.Schema{
									_TriggerAfterAttr: &schema.Schema{
										Type:             schema.TypeString,
										Optional:         true,
										DiffSuppressFunc: suppressEquivalentTimeDurations,
										StateFunc:        normalizeTimeDurationStringToSeconds,
										ValidateFunc: _ValidateFuncs(
											_ValidateDurationMin(_TriggerAfterAttr, "0s"),
										),
									},
									_TriggerNotifyAttr: &schema.Schema{
										Type:     schema.TypeList,
										Optional: true,
										MinItems: 1,
										Elem: &schema.Schema{
											Type:         schema.TypeString,
											ValidateFunc: _ValidateContactGroupCID(_TriggerNotifyAttr),
										},
									},
									_TriggerSeverityAttr: &schema.Schema{
										Type:     schema.TypeInt,
										Optional: true,
										Default:  defaultTriggerSeverity,
										ValidateFunc: _ValidateFuncs(
											_ValidateIntMax(_TriggerSeverityAttr, maxSeverity),
											_ValidateIntMin(_TriggerSeverityAttr, minSeverity),
										),
									},
								}, _TriggerIfThenDescriptions),
							},
						},
						_TriggerValueAttr: &schema.Schema{
							Type:     schema.TypeSet,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: _CastSchemaToTF(map[_SchemaAttr]*schema.Schema{
									_TriggerAbsentAttr: &schema.Schema{
										Type:             schema.TypeString, // Applies to text or numeric metrics
										Optional:         true,
										DiffSuppressFunc: suppressEquivalentTimeDurations,
										StateFunc:        normalizeTimeDurationStringToSeconds,
										ValidateFunc: _ValidateFuncs(
											_ValidateDurationMin(_TriggerAbsentAttr, _TriggerAbsentMin),
										),
										ConflictsWith: makeConflictsWith(_TriggerChangedAttr, _TriggerContainsAttr, _TriggerEqualsAttr, _TriggerExcludesAttr, _TriggerLessAttr, _TriggerMissingAttr, _TriggerMoreAttr, _TriggerOverAttr),
									},
									_TriggerChangedAttr: &schema.Schema{
										Type:          schema.TypeBool, // Applies to text or numeric metrics
										Optional:      true,
										ConflictsWith: makeConflictsWith(_TriggerAbsentAttr, _TriggerContainsAttr, _TriggerEqualsAttr, _TriggerExcludesAttr, _TriggerLessAttr, _TriggerMissingAttr, _TriggerMoreAttr, _TriggerOverAttr),
									},
									_TriggerContainsAttr: &schema.Schema{
										Type:          schema.TypeString, // Applies to text metrics only
										Optional:      true,
										ValidateFunc:  _ValidateRegexp(_TriggerContainsAttr, `.+`),
										ConflictsWith: makeConflictsWith(_TriggerAbsentAttr, _TriggerChangedAttr, _TriggerEqualsAttr, _TriggerExcludesAttr, _TriggerLessAttr, _TriggerMissingAttr, _TriggerMoreAttr, _TriggerOverAttr),
									},
									_TriggerEqualsAttr: &schema.Schema{
										Type:          schema.TypeString, // Applies to text metrics only
										Optional:      true,
										ValidateFunc:  _ValidateRegexp(_TriggerEqualsAttr, `.+`),
										ConflictsWith: makeConflictsWith(_TriggerAbsentAttr, _TriggerChangedAttr, _TriggerContainsAttr, _TriggerExcludesAttr, _TriggerLessAttr, _TriggerMissingAttr, _TriggerMoreAttr, _TriggerOverAttr),
									},
									_TriggerExcludesAttr: &schema.Schema{
										Type:          schema.TypeString, // Applies to text metrics only
										Optional:      true,
										ValidateFunc:  _ValidateRegexp(_TriggerExcludesAttr, `.+`),
										ConflictsWith: makeConflictsWith(_TriggerAbsentAttr, _TriggerChangedAttr, _TriggerContainsAttr, _TriggerEqualsAttr, _TriggerLessAttr, _TriggerMissingAttr, _TriggerMoreAttr, _TriggerOverAttr),
									},
									_TriggerLessAttr: &schema.Schema{
										Type:          schema.TypeString, // Applies to numeric metrics only
										Optional:      true,
										ValidateFunc:  _ValidateRegexp(_TriggerLessAttr, `.+`), // TODO(sean): improve this regexp to match int and float
										ConflictsWith: makeConflictsWith(_TriggerAbsentAttr, _TriggerChangedAttr, _TriggerContainsAttr, _TriggerEqualsAttr, _TriggerExcludesAttr, _TriggerMissingAttr, _TriggerMoreAttr),
									},
									_TriggerMissingAttr: &schema.Schema{
										Type:          schema.TypeString, // Applies to text metrics only
										Optional:      true,
										ValidateFunc:  _ValidateRegexp(_TriggerMissingAttr, `.+`),
										ConflictsWith: makeConflictsWith(_TriggerAbsentAttr, _TriggerChangedAttr, _TriggerContainsAttr, _TriggerEqualsAttr, _TriggerExcludesAttr, _TriggerLessAttr, _TriggerMoreAttr, _TriggerOverAttr),
									},
									_TriggerMoreAttr: &schema.Schema{
										Type:          schema.TypeString, // Applies to numeric metrics only
										Optional:      true,
										ValidateFunc:  _ValidateRegexp(_TriggerMoreAttr, `.+`), // TODO(sean): improve this regexp to match int and float
										ConflictsWith: makeConflictsWith(_TriggerAbsentAttr, _TriggerChangedAttr, _TriggerContainsAttr, _TriggerEqualsAttr, _TriggerExcludesAttr, _TriggerLessAttr, _TriggerMissingAttr),
									},
									_TriggerOverAttr: &schema.Schema{
										Type:     schema.TypeSet,
										Optional: true,
										MaxItems: 1,
										// _TriggerOverAttr is only compatible with checks of
										// numeric type.  NOTE: It may be premature to conflict with
										// _TriggerChangedAttr.
										ConflictsWith: makeConflictsWith(_TriggerAbsentAttr, _TriggerChangedAttr, _TriggerContainsAttr, _TriggerEqualsAttr, _TriggerExcludesAttr, _TriggerMissingAttr),
										Elem: &schema.Resource{
											Schema: _CastSchemaToTF(map[_SchemaAttr]*schema.Schema{
												_TriggerLastAttr: &schema.Schema{
													Type:             schema.TypeString,
													Optional:         true,
													Default:          defaultTriggerLast,
													DiffSuppressFunc: suppressEquivalentTimeDurations,
													StateFunc:        normalizeTimeDurationStringToSeconds,
													ValidateFunc: _ValidateFuncs(
														_ValidateDurationMin(_TriggerLastAttr, "0s"),
													),
												},
												_TriggerUsingAttr: &schema.Schema{
													Type:         schema.TypeString,
													Optional:     true,
													Default:      defaultTriggerWindowFunc,
													ValidateFunc: _ValidateStringIn(_TriggerUsingAttr, _ValidTriggerWindowFuncs),
												},
											}, _TriggerIfValueOverDescriptions),
										},
									},
								}, _TriggerIfValueDescriptions),
							},
						},
					}, _TriggerIfDescriptions),
				},
			},
			_TriggerLinkAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: _ValidateHTTPURL(_TriggerLinkAttr, _URLIsAbs),
			},
			_TriggerMetricTypeAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      defaultTriggerMetricType,
				ValidateFunc: _ValidateStringIn(_TriggerMetricTypeAttr, _ValidTriggerMetricTypes),
			},
			_TriggerNotesAttr: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				StateFunc: func(v interface{}) string {
					return strings.TrimSpace(v.(string))
				},
			},
			_TriggerParentAttr: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				StateFunc: func(v interface{}) string {
					return strings.TrimSpace(v.(string))
				},
				ValidateFunc: _ValidateRegexp(_TriggerParentAttr, `^[\d]+_[\d\w]+$`),
			},
			_TriggerStreamNameAttr: &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: _ValidateRegexp(_TriggerStreamNameAttr, `^[\S]+$`),
			},
			_TriggerTagsAttr: _TagMakeConfigSchema(_TriggerTagsAttr),
		}, _TriggerDescriptions),
	}
}

func _TriggerCreate(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*_ProviderContext)
	t := _NewTrigger()
	cr := _NewConfigReader(ctxt, d)
	if err := t.ParseConfig(cr); err != nil {
		return errwrap.Wrapf("error parsing trigger schema during create: {{err}}", err)
	}

	if err := t.Create(ctxt); err != nil {
		return errwrap.Wrapf("error creating trigger: {{err}}", err)
	}

	d.SetId(t.CID)

	return _TriggerRead(d, meta)
}

func _TriggerExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	ctxt := meta.(*_ProviderContext)

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

// _TriggerRead pulls data out of the RuleSet object and stores it into the
// appropriate place in the statefile.
func _TriggerRead(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*_ProviderContext)

	cid := d.Id()
	t, err := _LoadTrigger(ctxt, api.CIDType(&cid))
	if err != nil {
		return err
	}

	ifRules := make([]interface{}, 0, defaultTriggerRuleLen)
	for _, rule := range t.Rules {
		ifAttrs := make(map[string]interface{}, 2)
		valueAttrs := make(map[string]interface{}, 2)
		valueOverAttrs := make(map[string]interface{}, 2)
		thenAttrs := make(map[string]interface{}, 3)

		switch rule.Criteria {
		case _APIRulesetAbsent:
			d, _ := time.ParseDuration(fmt.Sprintf("%fs", rule.Value.(float64)))
			valueAttrs[string(_TriggerAbsentAttr)] = fmt.Sprintf("%ds", int(d.Seconds()))
		case _APIRulesetChanged:
			valueAttrs[string(_TriggerChangedAttr)] = true
		case _APIRulesetContains:
			valueAttrs[string(_TriggerContainsAttr)] = rule.Value
		case _APIRulesetMatch:
			valueAttrs[string(_TriggerEqualsAttr)] = rule.Value
		case _APIRulesetMaxValue:
			valueAttrs[string(_TriggerMoreAttr)] = rule.Value
		case _APIRulesetMinValue:
			valueAttrs[string(_TriggerLessAttr)] = rule.Value
		case _APIRulesetNotContains:
			valueAttrs[string(_TriggerExcludesAttr)] = rule.Value
		case _APIRulesetNotMatch:
			valueAttrs[string(_TriggerMissingAttr)] = rule.Value
		default:
			panic(fmt.Sprintf("PROVIDER BUG: Unsupported criteria %q", rule.Criteria))
		}

		if rule.Wait > 0 {
			thenAttrs[string(_TriggerAfterAttr)] = fmt.Sprintf("%ds", 60*rule.Wait)
		}
		thenAttrs[string(_TriggerSeverityAttr)] = int(rule.Severity)

		if rule.WindowingFunction != nil {
			valueOverAttrs[string(_TriggerUsingAttr)] = *rule.WindowingFunction

			// NOTE: Only save the window duration if a function was specified
			valueOverAttrs[string(_TriggerLastAttr)] = fmt.Sprintf("%ds", rule.WindowingDuration)
		}
		valueOverSet := schema.NewSet(_TriggerValueOverChecksum, nil)
		valueOverSet.Add(valueOverAttrs)
		valueAttrs[string(_TriggerOverAttr)] = valueOverSet

		if contactGroups, ok := t.ContactGroups[uint8(rule.Severity)]; ok {
			sort.Strings(contactGroups)
			thenAttrs[string(_TriggerNotifyAttr)] = contactGroups
		}
		thenSet := schema.NewSet(_TriggerThenChecksum, nil)
		thenSet.Add(thenAttrs)

		valueSet := schema.NewSet(_TriggerValueChecksum, nil)
		valueSet.Add(valueAttrs)
		ifAttrs[string(_TriggerThenAttr)] = thenSet
		ifAttrs[string(_TriggerValueAttr)] = valueSet

		ifRules = append(ifRules, ifAttrs)
	}

	_StateSet(d, _TriggerCheckAttr, t.CheckCID)
	_StateSet(d, _TriggerIfAttr, ifRules)
	_StateSet(d, _TriggerLinkAttr, _Indirect(t.Link))
	_StateSet(d, _TriggerStreamNameAttr, t.MetricName)
	_StateSet(d, _TriggerMetricTypeAttr, t.MetricType)
	_StateSet(d, _TriggerNotesAttr, _Indirect(t.Notes))
	_StateSet(d, _TriggerParentAttr, _Indirect(t.Parent))
	_StateSet(d, _TriggerTagsAttr, tagsToState(apiToTags(t.Tags)))

	d.SetId(t.CID)

	return nil
}

func _TriggerUpdate(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*_ProviderContext)
	t := _NewTrigger()
	cr := _NewConfigReader(ctxt, d)
	if err := t.ParseConfig(cr); err != nil {
		return err
	}

	t.CID = d.Id()
	if err := t.Update(ctxt); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("unable to update trigger %q: {{err}}", d.Id()), err)
	}

	return _TriggerRead(d, meta)
}

func _TriggerDelete(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*_ProviderContext)

	cid := d.Id()
	if _, err := ctxt.client.DeleteRuleSetByCID(api.CIDType(&cid)); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("unable to delete trigger %q: {{err}}", d.Id()), err)
	}

	d.SetId("")

	return nil
}

func _TriggerGroup(v interface{}) int {
	m := v.(map[string]interface{})
	ar := _NewMapReader(nil, m)

	b := &bytes.Buffer{}
	b.Grow(defaultHashBufSize)

	// Order writes to the buffer using lexically sorted list for easy visual
	// reconciliation with other lists.
	fmt.Fprint(b, ar.GetString(_TriggerCheckAttr))
	if p := ar.GetStringPtr(_TriggerLinkAttr); p != nil {
		fmt.Fprint(b, _Indirect(p))
	}
	fmt.Fprint(b, ar.GetString(_TriggerStreamNameAttr))
	fmt.Fprint(b, ar.GetString(_TriggerMetricTypeAttr))
	if p := ar.GetStringPtr(_TriggerNotesAttr); p != nil {
		fmt.Fprint(b, _Indirect(p))
	}
	{
		tags := ar.GetTags(_TriggerTagsAttr)
		for _, tag := range tags {
			fmt.Fprint(b, tag)
		}
	}

	s := b.String()
	return hashcode.String(s)
}

type _Trigger struct {
	api.RuleSet
}

func _NewTrigger() _Trigger {
	t := _Trigger{
		RuleSet: *api.NewRuleSet(),
	}

	t.ContactGroups = make(map[uint8][]string, config.NumSeverityLevels)
	for i := uint8(0); i < config.NumSeverityLevels; i++ {
		t.ContactGroups[i+1] = make([]string, 0, 1)
	}

	t.Rules = make([]api.RuleSetRule, 0, 1)

	return t
}

func _LoadTrigger(ctxt *_ProviderContext, cid api.CIDType) (_Trigger, error) {
	var t _Trigger
	rs, err := ctxt.client.FetchRuleSet(cid)
	if err != nil {
		return _Trigger{}, err
	}
	t.RuleSet = *rs

	return t, nil
}

func _TriggerThenChecksum(v interface{}) int {
	b := &bytes.Buffer{}
	b.Grow(defaultHashBufSize)

	writeInt := func(ar _AttrReader, attrName _SchemaAttr) {
		if i, ok := ar.GetIntOK(attrName); ok && i != 0 {
			fmt.Fprintf(b, "%x", i)
		}
	}

	writeString := func(ar _AttrReader, attrName _SchemaAttr) {
		if s, ok := ar.GetStringOK(attrName); ok && s != "" {
			fmt.Fprint(b, strings.TrimSpace(s))
		}
	}

	writeStringArray := func(ar _AttrReader, attrName _SchemaAttr) {
		if a := ar.GetStringSlice(attrName); a != nil {
			sort.Strings(a)
			for _, s := range a {
				fmt.Fprint(b, strings.TrimSpace(s))
			}
		}
	}

	m := v.(map[string]interface{})
	thenReader := _NewMapReader(nil, m)

	writeString(thenReader, _TriggerAfterAttr)
	writeStringArray(thenReader, _TriggerNotifyAttr)
	writeInt(thenReader, _TriggerSeverityAttr)

	s := b.String()
	return hashcode.String(s)
}

func _TriggerValueChecksum(v interface{}) int {
	b := &bytes.Buffer{}
	b.Grow(defaultHashBufSize)

	writeBool := func(ar _AttrReader, attrName _SchemaAttr) {
		if v, ok := ar.GetBoolOK(attrName); ok {
			fmt.Fprintf(b, "%t", v)
		}
	}

	writeDuration := func(ar _AttrReader, attrName _SchemaAttr) {
		if s, ok := ar.GetStringOK(attrName); ok && s != "" {
			d, _ := time.ParseDuration(s)
			fmt.Fprint(b, d.String())
		}
	}

	// writeFloat64 := func(ar _AttrReader, attrName _SchemaAttr) {
	// 	if f, ok := ar.GetFloat64OK(attrName); ok {
	// 		fmt.Fprintf(b, "%f", f)
	// 	}
	// }

	writeString := func(ar _AttrReader, attrName _SchemaAttr) {
		if s, ok := ar.GetStringOK(attrName); ok && s != "" {
			fmt.Fprint(b, strings.TrimSpace(s))
		}
	}

	m := v.(map[string]interface{})
	ifReader := _NewMapReader(nil, m)

	if valueReader := _NewMapReader(nil, ifReader.GetMap(_TriggerValueAttr)); valueReader != nil {
		// writeFloat64(valueReader, _TriggerAbsentAttr)
		writeDuration(valueReader, _TriggerAbsentAttr)
		writeBool(valueReader, _TriggerChangedAttr)
		writeString(valueReader, _TriggerContainsAttr)
		writeString(valueReader, _TriggerEqualsAttr)
		writeString(valueReader, _TriggerExcludesAttr)
		writeString(valueReader, _TriggerLessAttr)
		writeString(valueReader, _TriggerMissingAttr)
		writeString(valueReader, _TriggerMoreAttr)

		if overReader := _NewMapReader(nil, valueReader.GetMap(_TriggerOverAttr)); overReader != nil {
			writeDuration(overReader, _TriggerLastAttr)
			writeString(overReader, _TriggerUsingAttr)
		}
	}

	s := b.String()
	return hashcode.String(s)
}

func _TriggerValueOverChecksum(v interface{}) int {
	b := &bytes.Buffer{}
	b.Grow(defaultHashBufSize)

	writeString := func(ar _AttrReader, attrName _SchemaAttr) {
		if s, ok := ar.GetStringOK(attrName); ok && s != "" {
			fmt.Fprint(b, strings.TrimSpace(s))
		}
	}

	m := v.(map[string]interface{})
	overReader := _NewMapReader(nil, m)

	writeString(overReader, _TriggerLastAttr)
	writeString(overReader, _TriggerUsingAttr)

	s := b.String()
	return hashcode.String(s)
}

// ParseConfig reads Terraform config data and stores the information into a
// Circonus RuleSet object.  ParseConfig, _TriggerRead(), and _TriggerChecksum
// must be kept in sync.
func (t *_Trigger) ParseConfig(ar _AttrReader) error {
	if s, ok := ar.GetStringOK(_TriggerCheckAttr); ok {
		t.CheckCID = s
	}

	t.Link = ar.GetStringPtr(_TriggerLinkAttr)

	if s, ok := ar.GetStringOK(_TriggerMetricTypeAttr); ok {
		t.MetricType = s
	}

	t.Notes = ar.GetStringPtr(_TriggerNotesAttr)
	t.Parent = ar.GetStringPtr(_TriggerParentAttr)
	if s, ok := ar.GetStringOK(_TriggerStreamNameAttr); ok {
		t.MetricName = s
	}

	t.Rules = make([]api.RuleSetRule, 0, defaultTriggerRuleLen)
	if ifList, ok := ar.GetListOK(_TriggerIfAttr); ok {
		for _, ifListRaw := range ifList {
			for _, ifListElem := range ifListRaw.([]interface{}) {
				ifAttrs := _NewInterfaceMap(ifListElem.(map[string]interface{}))
				ifReader := _NewMapReader(ar.Context(), ifAttrs)
				rule := api.RuleSetRule{}

				if thenList, ok := ifReader.GetSetAsListOK(_TriggerThenAttr); ok {
					for _, thenListRaw := range thenList {
						thenAttrs := _NewInterfaceMap(thenListRaw)
						thenReader := _NewMapReader(ar.Context(), thenAttrs)

						if s, ok := thenReader.GetStringOK(_TriggerAfterAttr); ok {
							d, _ := time.ParseDuration(s)
							rule.Wait = uint(d.Minutes())
						}

						// NOTE: break from convention of alpha sorting attributes and handle Notify after Severity

						if i, ok := thenReader.GetIntOK(_TriggerSeverityAttr); ok {
							rule.Severity = uint(i)
						}

						if notifyList, ok := thenReader.GetListOK(_TriggerNotifyAttr); ok {
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

				if valueList, ok := ifReader.GetSetAsListOK(_TriggerValueAttr); ok {
					for _, valueListRaw := range valueList {
						valueAttrs := _NewInterfaceMap(valueListRaw)
						valueReader := _NewMapReader(ar.Context(), valueAttrs)

					METRIC_TYPE:
						switch t.MetricType {
						case _TriggerMetricTypeNumeric:
							if s, ok := valueReader.GetStringOK(_TriggerAbsentAttr); ok && s != "" {
								d, _ := time.ParseDuration(s)
								rule.Criteria = _APIRulesetAbsent
								rule.Value = float64(d.Seconds())
								break METRIC_TYPE
							}

							if b, ok := valueReader.GetBoolOK(_TriggerChangedAttr); ok && b {
								rule.Criteria = _APIRulesetChanged
								break METRIC_TYPE
							}

							if s, ok := valueReader.GetStringOK(_TriggerLessAttr); ok && s != "" {
								rule.Criteria = _APIRulesetMinValue
								rule.Value = s
								break METRIC_TYPE
							}

							if s, ok := valueReader.GetStringOK(_TriggerMoreAttr); ok && s != "" {
								rule.Criteria = _APIRulesetMaxValue
								rule.Value = s
								break METRIC_TYPE
							}
						case _TriggerMetricTypeText:
							if s, ok := valueReader.GetStringOK(_TriggerAbsentAttr); ok && s != "" {
								d, _ := time.ParseDuration(s)
								rule.Criteria = _APIRulesetAbsent
								rule.Value = float64(d.Seconds())
								break METRIC_TYPE
							}

							if b, ok := valueReader.GetBoolOK(_TriggerChangedAttr); ok && b {
								rule.Criteria = _APIRulesetChanged
								break METRIC_TYPE
							}

							if s, ok := valueReader.GetStringOK(_TriggerContainsAttr); ok && s != "" {
								rule.Criteria = _APIRulesetContains
								rule.Value = s
								break METRIC_TYPE
							}

							if s, ok := valueReader.GetStringOK(_TriggerEqualsAttr); ok && s != "" {
								rule.Criteria = _APIRulesetMatch
								rule.Value = s
								break METRIC_TYPE
							}

							if s, ok := valueReader.GetStringOK(_TriggerExcludesAttr); ok && s != "" {
								rule.Criteria = _APIRulesetNotMatch
								rule.Value = s
								break METRIC_TYPE
							}

							if s, ok := valueReader.GetStringOK(_TriggerMissingAttr); ok && s != "" {
								rule.Criteria = _APIRulesetNotContains
								rule.Value = s
								break METRIC_TYPE
							}
						default:
							panic(fmt.Sprintf("PROVIDER BUG: unsupported trigger metric type: %q", t.MetricType))
						}

						if overList, ok := valueReader.GetSetAsListOK(_TriggerOverAttr); ok {
							for _, overListRaw := range overList {
								overAttrs := _NewInterfaceMap(overListRaw)
								overReader := _NewMapReader(ar.Context(), overAttrs)

								if s, ok := overReader.GetStringOK(_TriggerLastAttr); ok {
									last, _ := time.ParseDuration(s)
									rule.WindowingDuration = uint(last.Seconds())
								}

								if s, ok := overReader.GetStringOK(_TriggerUsingAttr); ok {
									rule.WindowingFunction = &s
								}
							}
						}
					}
				}
				t.Rules = append(t.Rules, rule)
			}
		}
	}

	t.Tags = tagsToAPI(ar.GetTags(_TriggerTagsAttr))

	if err := t.Validate(); err != nil {
		return err
	}

	return nil
}

func (t *_Trigger) Create(ctxt *_ProviderContext) error {
	rs, err := ctxt.client.CreateRuleSet(&t.RuleSet)
	if err != nil {
		return err
	}

	t.CID = rs.CID

	return nil
}

func (t *_Trigger) Update(ctxt *_ProviderContext) error {
	_, err := ctxt.client.UpdateRuleSet(&t.RuleSet)
	if err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Unable to update trigger %s: {{err}}", t.CID), err)
	}

	return nil
}

func (t *_Trigger) Validate() error {
	// TODO(sean@): From https://login.circonus.com/resources/api/calls/rule_set
	// under `value`:
	//
	// For an 'on absence' rule this is the number of seconds the metric must not
	// have been collected for, and should not be lower than either the period or
	// timeout of the metric being collected.
	return nil
}
