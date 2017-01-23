package circonus

/*
 * Note to future readers: The `circonus_check` resource is actually a facade for
 * the check_bundle call.  check_bundle is an implementation detail that we mask
 * over and expose just a "check" even though the "check" is actually a
 * check_bundle.
 *
 * Style note: There are three directions that information flows:
 *
 * 1) Terraform Config file into API Objects.  *Attr named objects are Config or
 *    Schema attribute names.  In this file, all config constants should be
 *     named _Check*Attr.
 *
 * 2) API Objects into Statefile data.  _API*Attr named constants are parameters
 *    that originate from the API and need to be mapped into the provider's
 *    vernacular.
 */

import (
	"fmt"
	"strings"
	"time"

	"github.com/circonus-labs/circonus-gometrics/api"
	"github.com/circonus-labs/circonus-gometrics/api/config"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
)

const (
	// circonus_check.* "global" resource attribute names
	_CheckActiveAttr      _SchemaAttr = "active"
	_CheckCAQLAttr        _SchemaAttr = "caql"
	_CheckCloudWatchAttr  _SchemaAttr = "cloudwatch"
	_CheckCollectorAttr   _SchemaAttr = "collector"
	_CheckHTTPAttr        _SchemaAttr = "http"
	_CheckICMPPingAttr    _SchemaAttr = "icmp_ping"
	_CheckJSONAttr        _SchemaAttr = "json"
	_CheckMetricLimitAttr _SchemaAttr = "metric_limit"
	_CheckNameAttr        _SchemaAttr = "name"
	_CheckNotesAttr       _SchemaAttr = "notes"
	_CheckPeriodAttr      _SchemaAttr = "period"
	_CheckPostgreSQLAttr  _SchemaAttr = "postgresql"
	_CheckStreamAttr      _SchemaAttr = "stream"
	_CheckTagsAttr        _SchemaAttr = "tags"
	_CheckTargetAttr      _SchemaAttr = "target"
	_CheckTCPAttr         _SchemaAttr = "tcp"
	_CheckTimeoutAttr     _SchemaAttr = "timeout"
	_CheckTypeAttr        _SchemaAttr = "type"

	// circonus_check.collector.* resource attribute names
	_CheckCollectorIDAttr _SchemaAttr = "id"

	// circonus_check.stream.* resource attribute names are aliased to
	// circonus_metric.* resource attributes.

	// circonus_check.streams.* resource attribute names
	//_MetricIDAttr _SchemaAttr = "id"

	// Out parameters for circonus_check
	_CheckOutCheckUUIDsAttr         _SchemaAttr = "uuids"
	_CheckOutChecksAttr             _SchemaAttr = "checks"
	_CheckOutCreatedAttr            _SchemaAttr = "created"
	_CheckOutLastModifiedAttr       _SchemaAttr = "last_modified"
	_CheckOutLastModifiedByAttr     _SchemaAttr = "last_modified_by"
	_CheckOutReverseConnectURLsAttr _SchemaAttr = "reverse_connect_urls"
)

const (
	// Circonus API constants from their API endpoints
	_APICheckTypeCAQLAttr       _APICheckType = "caql"
	_APICheckTypeCloudWatchAttr _APICheckType = "cloudwatch"
	_APICheckTypeHTTPAttr       _APICheckType = "http"
	_APICheckTypeICMPPingAttr   _APICheckType = "ping_icmp"
	_APICheckTypeJSONAttr       _APICheckType = "json"
	_APICheckTypePostgreSQLAttr _APICheckType = "postgres"
	_APICheckTypeTCPAttr        _APICheckType = "tcp"
)

var _CheckDescriptions = _AttrDescrs{
	_CheckActiveAttr:      "If the check is activate or disabled",
	_CheckCAQLAttr:        "CAQL check configuration",
	_CheckCloudWatchAttr:  "CloudWatch check configuration",
	_CheckCollectorAttr:   "The collector(s) that are responsible for gathering the metrics",
	_CheckHTTPAttr:        "HTTP check configuration",
	_CheckICMPPingAttr:    "ICMP ping check configuration",
	_CheckJSONAttr:        "JSON check configuration",
	_CheckMetricLimitAttr: `Setting a metric_limit will enable all (-1), disable (0), or allow up to the specified limit of metrics for this check ("N+", where N is a positive integer)`,
	_CheckNameAttr:        "The name of the check bundle that will be displayed in the web interface",
	_CheckNotesAttr:       "Notes about this check bundle",
	_CheckPeriodAttr:      "The period between each time the check is made",
	_CheckPostgreSQLAttr:  "PostgreSQL check configuration",
	_CheckStreamAttr:      "Configuration for a stream of metrics",
	_CheckTagsAttr:        "A list of tags assigned to the check",
	_CheckTargetAttr:      "The target of the check (e.g. hostname, URL, IP, etc)",
	_CheckTCPAttr:         "TCP check configuration",
	_CheckTimeoutAttr:     "The length of time in seconds (and fractions of a second) before the check will timeout if no response is returned to the collector",
	_CheckTypeAttr:        "The check type",

	_CheckOutChecksAttr:             "",
	_CheckOutCheckUUIDsAttr:         "",
	_CheckOutCreatedAttr:            "",
	_CheckOutLastModifiedAttr:       "",
	_CheckOutLastModifiedByAttr:     "",
	_CheckOutReverseConnectURLsAttr: "",
}

var _CheckCollectorDescriptions = _AttrDescrs{
	_CheckCollectorIDAttr: "The ID of the collector",
}

var _CheckStreamDescriptions = _MetricDescriptions

func _NewCheckResource() *schema.Resource {
	return &schema.Resource{
		Create: _CheckCreate,
		Read:   _CheckRead,
		Update: _CheckUpdate,
		Delete: _CheckDelete,
		Exists: _CheckExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: _CastSchemaToTF(map[_SchemaAttr]*schema.Schema{
			_CheckActiveAttr: &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			_CheckCAQLAttr:       _SchemaCheckCAQL,
			_CheckCloudWatchAttr: _SchemaCheckCloudWatch,
			_CheckCollectorAttr: &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: _CastSchemaToTF(map[_SchemaAttr]*schema.Schema{
						_CheckCollectorIDAttr: &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: _ValidateRegexp(_CheckCollectorIDAttr, config.BrokerCIDRegex),
						},
					}, _CheckCollectorDescriptions),
				},
			},
			_CheckHTTPAttr:     _SchemaCheckHTTP,
			_CheckJSONAttr:     _SchemaCheckJSON,
			_CheckICMPPingAttr: _SchemaCheckICMPPing,
			_CheckMetricLimitAttr: &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
				ValidateFunc: _ValidateFuncs(
					_ValidateIntMin(_CheckMetricLimitAttr, -1),
				),
			},
			_CheckNameAttr: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			_CheckNotesAttr: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				StateFunc: func(v interface{}) string {
					return strings.TrimSpace(v.(string))
				},
			},
			_CheckPeriodAttr: &schema.Schema{
				Type:      schema.TypeString,
				Optional:  true,
				Computed:  true,
				StateFunc: normalizeTimeDurationStringToSeconds,
				ValidateFunc: _ValidateFuncs(
					_ValidateDurationMin(_CheckPeriodAttr, defaultCirconusCheckPeriodMin),
					_ValidateDurationMax(_CheckPeriodAttr, defaultCirconusCheckPeriodMax),
				),
			},
			_CheckPostgreSQLAttr: _SchemaCheckPostgreSQL,
			_CheckStreamAttr: &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Set:      _CheckStreamChecksum,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: _CastSchemaToTF(map[_SchemaAttr]*schema.Schema{
						_MetricActiveAttr: &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						_MetricNameAttr: &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: _ValidateRegexp(_MetricNameAttr, `[\S]+`),
						},
						_MetricTagsAttr: _TagMakeConfigSchema(_MetricTagsAttr),
						_MetricTypeAttr: &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: _ValidateMetricType,
						},
						_MetricUnitAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: _ValidateRegexp(_MetricUnitAttr, `.+`),
						},
					}, _CheckStreamDescriptions),
				},
			},
			_CheckTagsAttr: _TagMakeConfigSchema(_CheckTagsAttr),
			_CheckTargetAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: _ValidateRegexp(_CheckTagsAttr, `.+`),
			},
			_CheckTCPAttr: _SchemaCheckTCP,
			_CheckTimeoutAttr: &schema.Schema{
				Type:      schema.TypeString,
				Optional:  true,
				Computed:  true,
				StateFunc: normalizeTimeDurationStringToSeconds,
				ValidateFunc: _ValidateFuncs(
					_ValidateDurationMin(_CheckTimeoutAttr, defaultCirconusTimeoutMin),
					_ValidateDurationMax(_CheckTimeoutAttr, defaultCirconusTimeoutMax),
				),
			},
			_CheckTypeAttr: &schema.Schema{
				Type:         schema.TypeString,
				Computed:     true,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: _ValidateCheckType,
			},

			// Out parameters
			_CheckOutCheckUUIDsAttr: &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			_CheckOutChecksAttr: &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			_CheckOutCreatedAttr: &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},
			_CheckOutLastModifiedAttr: &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},
			_CheckOutLastModifiedByAttr: &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			_CheckOutReverseConnectURLsAttr: &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		}, _CheckDescriptions),
	}
}

func _CheckCreate(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*_ProviderContext)
	c := _NewCheck()
	cr := _NewConfigReader(ctxt, d)
	if err := c.ParseConfig(cr); err != nil {
		return errwrap.Wrapf("error parsing check schema during create: {{err}}", err)
	}

	if err := c.Create(ctxt); err != nil {
		return errwrap.Wrapf("error creating check: {{err}}", err)
	}

	d.SetId(c.CID)

	return _CheckRead(d, meta)
}

func _CheckExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	ctxt := meta.(*_ProviderContext)

	cid := d.Id()
	cb, err := ctxt.client.FetchCheckBundle(api.CIDType(&cid))
	if err != nil {
		return false, err
	}

	if cb.CID == "" {
		return false, nil
	}

	return true, nil
}

// _CheckRead pulls data out of the CheckBundle object and stores it into the
// appropriate place in the statefile.
func _CheckRead(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*_ProviderContext)

	cid := d.Id()
	c, err := _LoadCheck(ctxt, api.CIDType(&cid))
	if err != nil {
		return err
	}

	// Global circonus_check attributes are saved first, followed by the check
	// type specific attributes handled below in their respective _CheckRead*().

	streams := schema.NewSet(_CheckStreamChecksum, nil)
	for _, m := range c.Metrics {
		streamAttrs := map[string]interface{}{
			string(_MetricActiveAttr): _MetricAPIStatusToBool(m.Status),
			string(_MetricNameAttr):   m.Name,
			string(_MetricTagsAttr):   tagsToState(apiToTags(m.Tags)),
			string(_MetricTypeAttr):   m.Type,
			string(_MetricUnitAttr):   _Indirect(m.Units),
		}

		streams.Add(streamAttrs)
	}

	// Write the global circonus_check parameters followed by the check
	// type-specific parameters.

	_StateSet(d, _CheckActiveAttr, _CheckAPIStatusToBool(c.Status))
	_StateSet(d, _CheckCollectorAttr, stringListToSet(c.Brokers, _CheckCollectorIDAttr))
	_StateSet(d, _CheckMetricLimitAttr, c.MetricLimit)
	_StateSet(d, _CheckNameAttr, c.DisplayName)
	_StateSet(d, _CheckNotesAttr, c.Notes)
	_StateSet(d, _CheckPeriodAttr, fmt.Sprintf("%ds", c.Period))
	_StateSet(d, _CheckStreamAttr, streams)
	_StateSet(d, _CheckTagsAttr, c.Tags)
	_StateSet(d, _CheckTargetAttr, c.Target)
	{
		t, _ := time.ParseDuration(fmt.Sprintf("%fs", c.Timeout))
		_StateSet(d, _CheckTimeoutAttr, t.String())
	}
	_StateSet(d, _CheckTypeAttr, c.Type)

	// Last step: parse a check_bundle's config into the statefile.
	if err := _ParseCheckTypeConfig(&c, d); err != nil {
		return errwrap.Wrapf("Unable to parse check config: {{err}}", err)
	}

	// Out parameters
	_StateSet(d, _CheckOutCheckUUIDsAttr, c.CheckUUIDs)
	_StateSet(d, _CheckOutChecksAttr, c.Checks)
	_StateSet(d, _CheckOutCreatedAttr, c.Created)
	_StateSet(d, _CheckOutLastModifiedAttr, c.LastModified)
	_StateSet(d, _CheckOutLastModifiedByAttr, c.LastModifedBy)
	_StateSet(d, _CheckOutReverseConnectURLsAttr, c.ReverseConnectURLs)

	d.SetId(c.CID)

	return nil
}

func _CheckUpdate(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*_ProviderContext)
	c := _NewCheck()
	cr := _NewConfigReader(ctxt, d)
	if err := c.ParseConfig(cr); err != nil {
		return err
	}

	c.CID = d.Id()
	if err := c.Update(ctxt); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("unable to update check %q: {{err}}", d.Id()), err)
	}

	return _CheckRead(d, meta)
}

func _CheckDelete(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*_ProviderContext)

	if _, err := ctxt.client.Delete(d.Id()); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("unable to delete check %q: {{err}}", d.Id()), err)
	}

	d.SetId("")

	return nil
}

func _CheckStreamChecksum(v interface{}) int {
	m := v.(map[string]interface{})

	ar := _NewMapReader(nil, m)
	csum := _MetricChecksum(ar)
	return csum
}

// ParseConfig reads Terraform config data and stores the information into a
// Circonus CheckBundle object.
func (c *_Check) ParseConfig(ar _AttrReader) error {
	if status, ok := ar.GetBoolOK(_CheckActiveAttr); ok {
		c.Status = _CheckActiveToAPIStatus(status)
	}

	if collectorsList, ok := ar.GetSetAsListOK(_CheckCollectorAttr); ok {
		c.Brokers = collectorsList.CollectList(_CheckCollectorIDAttr)
	}

	if i, ok := ar.GetIntOK(_CheckMetricLimitAttr); ok {
		c.MetricLimit = i
	}

	if name, ok := ar.GetStringOK(_CheckNameAttr); ok {
		c.DisplayName = name
	}

	c.Notes = ar.GetStringPtr(_CheckNotesAttr)

	if d, ok := ar.GetDurationOK(_CheckPeriodAttr); ok {
		c.Period = uint(d.Seconds())
	}

	if streamList, ok := ar.GetSetAsListOK(_CheckStreamAttr); ok {
		c.Metrics = make([]api.CheckBundleMetric, 0, len(streamList))

		for _, metricListRaw := range streamList {
			metricAttrs := _NewInterfaceMap(metricListRaw)
			mr := _NewMapReader(ar.Context(), metricAttrs)

			var id string
			if v, ok := mr.GetStringOK(_MetricIDAttr); ok {
				id = v
			} else {
				var err error
				id, err = _NewMetricID()
				if err != nil {
					return errwrap.Wrapf("unable to create a new metric ID: {{err}}", err)
				}
			}

			m := _NewMetric()
			if err := m.ParseConfig(id, mr); err != nil {
				return errwrap.Wrapf("unable to parse config: {{err}}", err)
			}

			c.Metrics = append(c.Metrics, m.CheckBundleMetric)
		}
	}

	c.Tags = tagsToAPI(ar.GetTags(_CheckTagsAttr))

	if s, ok := ar.GetStringOK(_CheckTargetAttr); ok {
		c.Target = s
	}

	if d, ok := ar.GetDurationOK(_CheckTimeoutAttr); ok {
		var t float32 = float32(d.Seconds())
		c.Timeout = t
	}

	// Last step: parse the individual check types
	if err := parsePerCheckTypeConfig(c, ar); err != nil {
		return errwrap.Wrapf("unable to parse check type: {{err}}", err)
	}

	if err := c.Fixup(); err != nil {
		return err
	}

	if err := c.Validate(); err != nil {
		return err
	}

	return nil
}

// parsePerCheckTypeConfig parses the Terraform config into the respective
// per-check type api.Config attributes.
func parsePerCheckTypeConfig(c *_Check, ar _AttrReader) error {
	checkTypeParseMap := map[_SchemaAttr]func(*_Check, *_ProviderContext, _InterfaceList) error{
		_CheckCAQLAttr:       _CheckConfigToAPICAQL,
		_CheckCloudWatchAttr: _CheckConfigToAPICloudWatch,
		_CheckHTTPAttr:       _CheckConfigToAPIHTTP,
		_CheckICMPPingAttr:   _CheckConfigToAPIICMPPing,
		_CheckJSONAttr:       _CheckConfigToAPIJSON,
		_CheckPostgreSQLAttr: _CheckConfigToAPIPostgreSQL,
		_CheckTCPAttr:        _CheckConfigToAPITCP,
	}

	for checkType, fn := range checkTypeParseMap {
		if l, ok := ar.GetSetAsListOK(checkType); ok {
			if err := fn(c, ar.Context(), l); err != nil {
				return errwrap.Wrapf(fmt.Sprintf("Unable to parse type %q: {{err}}", string(checkType)), err)
			}
		}
	}

	return nil
}

// _ParseCheckTypeConfig parses an API Config object and stores the result in the
// statefile.
func _ParseCheckTypeConfig(c *_Check, d *schema.ResourceData) error {
	checkTypeConfigHandlers := map[_APICheckType]func(*_Check, *schema.ResourceData) error{
		_APICheckTypeCAQLAttr:       _CheckAPIToStateCAQL,
		_APICheckTypeCloudWatchAttr: _CheckAPIToStateCloudWatch,
		_APICheckTypeHTTPAttr:       _CheckAPIToStateHTTP,
		_APICheckTypeICMPPingAttr:   _CheckAPIToStateICMPPing,
		_APICheckTypeJSONAttr:       _CheckAPIToStateJSON,
		_APICheckTypePostgreSQLAttr: _CheckAPIToStatePostgreSQL,
		_APICheckTypeTCPAttr:        _CheckAPIToStateTCP,
	}

	var checkType _APICheckType = _APICheckType(c.Type)
	fn, ok := checkTypeConfigHandlers[checkType]
	if !ok {
		return fmt.Errorf("check type %q not supported", c.Type)
	}

	if err := fn(c, d); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("unable to parse the API config for %q: {{err}}", c.Type), err)
	}

	return nil
}
