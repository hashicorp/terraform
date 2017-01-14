package circonus

/*
 * Note to future readers: The `circonus_check` resource is actually a facade for
 * the check_bundle call.  check_bundle is an implementation detail that we mask
 * over and expose just a "check" even though the "check" is actually a
 * check_bundle.
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
	// circonus_check.* resource attribute names
	_CheckActiveAttr      _SchemaAttr = "active"
	_CheckCollectorAttr   _SchemaAttr = "collector"
	_CheckJSONAttr        _SchemaAttr = "json"
	_CheckMetricLimitAttr _SchemaAttr = "metric_limit"
	_CheckNameAttr        _SchemaAttr = "name"
	_CheckNotesAttr       _SchemaAttr = "notes"
	_CheckPeriodAttr      _SchemaAttr = "period"
	_CheckStreamAttr      _SchemaAttr = "stream"
	_CheckStreamsAttr     _SchemaAttr = "streams"
	_CheckTagsAttr        _SchemaAttr = "tags"
	_CheckTargetAttr      _SchemaAttr = "target"
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
	_APICheckTypeJSON _APICheckType = "json"
)

var _CheckDescriptions = _AttrDescrs{
	_CheckActiveAttr:      "If the check is activate or disabled",
	_CheckCollectorAttr:   "The collector(s) that are responsible for gathering the metrics",
	_CheckMetricLimitAttr: `Setting a metric_limit will enable all (-1), disable (0), or allow up to the specified limit of metrics for this check ("N+", where N is a positive integer)`,
	_CheckNameAttr:        "The name of the check bundle that will be displayed in the web interface",
	_CheckNotesAttr:       "Notes about this check bundle",
	_CheckPeriodAttr:      "The period between each time the check is made",
	_CheckTagsAttr:        "A list of tags assigned to the check",
	_CheckTargetAttr:      "The target of the check (e.g. hostname, URL, IP, etc)",
	_CheckTimeoutAttr:     "The length of time in seconds (and fractions of a second) before the check will timeout if no response is returned to the collector",
	_CheckTypeAttr:        "The check type",
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
			_CheckCollectorAttr: &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: _CastSchemaToTF(map[_SchemaAttr]*schema.Schema{
						_CheckCollectorIDAttr: &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateRegexp(_CheckCollectorIDAttr, config.BrokerCIDRegex),
						},
					}, _CheckCollectorDescriptions),
				},
			},
			_CheckJSONAttr: jsonAttr,
			_CheckMetricLimitAttr: &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
				ValidateFunc: validateFuncs(
					validateIntMin(_CheckMetricLimitAttr, -1),
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
				ValidateFunc: validateFuncs(
					validateDurationMin(_CheckPeriodAttr, defaultCirconusCheckPeriodMin),
					validateDurationMax(_CheckPeriodAttr, defaultCirconusCheckPeriodMax),
				),
			},
			_CheckStreamAttr: &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
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
							ValidateFunc: validateRegexp(_MetricNameAttr, `[\S]+`),
						},
						_MetricTagsAttr: _TagMakeConfigSchema(_MetricTagsAttr),
						_MetricTypeAttr: &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateMetricType,
						},
						_MetricUnitAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateRegexp(_MetricUnitAttr, `.+`),
						},
					}, _CheckStreamDescriptions),
				},
			},
			_CheckStreamsAttr: &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Set:      schema.HashString,
				MinItems: 1,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validateUUID(_MetricIDAttr),
				},
			},
			_CheckTagsAttr: _TagMakeConfigSchema(_CheckTagsAttr),
			_CheckTargetAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validateHTTPURL(_CheckTargetAttr, _URLWithoutSchema|_URLWithoutPort),
			},
			_CheckTimeoutAttr: &schema.Schema{
				Type:      schema.TypeString,
				Optional:  true,
				Computed:  true,
				StateFunc: normalizeTimeDurationStringToSeconds,
				ValidateFunc: validateFuncs(
					validateDurationMin(_CheckTimeoutAttr, defaultCirconusTimeoutMin),
					validateDurationMax(_CheckTimeoutAttr, defaultCirconusTimeoutMax),
				),
			},
			_CheckTypeAttr: &schema.Schema{
				Type:         schema.TypeString,
				Computed:     true,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validateCheckType,
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

func _CheckRead(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*_ProviderContext)

	cid := d.Id()
	c, err := _LoadCheck(ctxt, api.CIDType(&cid))
	if err != nil {
		return err
	}

	// Global circonus_check attributes are saved first, followed by the check
	// type specific attributes handled below in their respective _CheckRead*().

	var streams []interface{}
	{
		for _, m := range c.Metrics {
			metricActive := _MetricAPIStatusToBool(m.Status)
			var unit string
			if m.Units != nil {
				unit = *m.Units
			}

			metricMap := map[_SchemaAttr]interface{}{
				_MetricActiveAttr: metricActive,
				_MetricNameAttr:   m.Name,
				// TODO(sean@): FIXME: For some reason when I include the stream's tags
				// Set fails.
				//
				// _MetricTagsAttr:   tagsToState(apiToTags(m.Tags)),
				_MetricTypeAttr: m.Type,
				_MetricUnitAttr: unit,
			}

			streams = append(streams, metricMap)
		}
	}

	// Write the global circonus_check parameters followed by the check
	// type-specific parameters.

	_StateSet(d, _CheckActiveAttr, apiCheckStatusToBool(c.Status))
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

	switch _APICheckType(c.Type) {
	case _APICheckTypeJSON:
		if err := c.ReadJSON(d); err != nil {
			return errwrap.Wrapf("Unable to read JSON: {{err}}", err)
		}
	default:
		panic(fmt.Sprintf("PROVIDER BUG: unsupported check type %s", c.Type))
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
