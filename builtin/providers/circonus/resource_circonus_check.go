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

	"github.com/circonus-labs/circonus-gometrics/api"
	"github.com/circonus-labs/circonus-gometrics/api/config"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
)

const (
	// circonus_check.* resource attribute names
	_CheckActiveAttr    _SchemaAttr = "active"
	_CheckCollectorAttr _SchemaAttr = "collector"
	_CheckJSONAttr      _SchemaAttr = "json"
	_CheckStreamAttr    _SchemaAttr = "stream"
	_CheckStreamsAttr   _SchemaAttr = "streams"

	// circonus_check.collector.* resource attribute names
	_CheckCollectorIDAttr _SchemaAttr = "id"

	// circonus_check.json.* resource attribute names
	_CheckJSONHTTPHeadersAttr _SchemaAttr = "http_headers"
	_CheckJSONHTTPVersionAttr _SchemaAttr = "http_version"
	_CheckJSONMethodAttr      _SchemaAttr = "method"
	_CheckJSONPortAttr        _SchemaAttr = "port"
	_CheckJSONReadLimitAttr   _SchemaAttr = "read_limit"
	_CheckJSONRedirectsAttr   _SchemaAttr = "redirects"
	_CheckJSONURLAttr         _SchemaAttr = "url"

	// circonus_check.stream.* resource attribute names
	//_MetricIDAttr   _SchemaAttr = "id"
	//_MetricNameAttr _SchemaAttr = "name"
	//_MetricTagsAttr _SchemaAttr = "tags"
	//_MetricUnitAttr _SchemaAttr = "unit"

	// circonus_check.streams.* resource attribute names
	//_MetricIDAttr _SchemaAttr = "id"

)

var _CheckDescriptions = _AttrDescrs{
	_CheckActiveAttr:               "If the check is activate or disabled",
	_CheckCollectorAttr:            "The collector(s) that are responsible for gathering the metrics",
	checkConfigAuthMethodAttr:      "The HTTP Authentication method",
	checkConfigAuthPasswordAttr:    "The HTTP Authentication user password",
	checkConfigAuthUserAttr:        "The HTTP Authentication user name",
	checkConfigCAChainAttr:         "A path to a file containing all the certificate authorities that should be loaded to validate the remote certificate (for SSL checks)",
	checkConfigCertificateFileAttr: "A path to a file containing the client certificate that will be presented to the remote server (for SSL checks)",
	checkConfigCiphersAttr:         "A list of ciphers to be used in the SSL protocol (for SSL checks)",
	checkConfigHTTPHeadersAttr:     "Map of HTTP Headers to send along with HTTP Requests",
	checkConfigHTTPVersionAttr:     "Sets the HTTP version for the check to use",
	checkConfigKeyFileAttr:         "A path to a file containing key to be used in conjunction with the cilent certificate (for SSL checks)",
	checkConfigMethodAttr:          "The HTTP method to use",
	checkConfigPayloadAttr:         "The information transferred as the payload of an HTTP request",
	checkConfigPortAttr:            "Specifies the port on which the management interface can be reached",
	checkConfigReadLimitAttr:       "Sets an approximate limit on the data read (0 means no limit)",
	checkConfigRedirectsAttr:       `The maximum number of Location header redirects to follow (0 means no limit)`,
	checkConfigURLAttr:             "The URL including schema and hostname (as you would type into a browser's location bar)",
	checkMetricLimitAttr:           `Setting a metric_limit will enable all (-1), disable (0), or allow up to the specified limit of metrics for this check ("N+", where N is a positive integer)`,
	checkMetricNamesAttr:           "A list of metric names found within this check",
	checkNameAttr:                  "The name of the check bundle that will be displayed in the web interface",
	checkNotesAttr:                 "Notes about this check bundle",
	checkPeriodAttr:                "The period between each time the check is made",
	checkTagsAttr:                  "A list of tags assigned to the check",
	checkTargetAttr:                "The target of the check (e.g. hostname, URL, IP, etc)",
	checkTimeoutAttr:               "The length of time in seconds (and fractions of a second) before the check will timeout if no response is returned to the collector",
	checkTypeAttr:                  "The check type",
}

var _CheckCollectorDescriptions = _AttrDescrs{
	_CheckCollectorIDAttr: "The ID of the collector",
}

// var _CheckStreamsDescriptions = _AttrDescrs{
// 	_MetricIDAttr: "The circonus_metric.id being used in a stream",
// }

var _CheckJSONDescriptions = _AttrDescrs{
	_CheckJSONURLAttr: "The URL to use as the target of the check",
}

var _CheckStreamDescriptions = _MetricDescriptions

func _NewCirconusCheckResource() *schema.Resource {
	return &schema.Resource{
		Create: _CheckCreate,
		Read:   checkBundleRead,
		Update: checkBundleUpdate,
		Delete: checkBundleDelete,
		Exists: checkBundleExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: castSchemaToTF(map[_SchemaAttr]*schema.Schema{
			_CheckActiveAttr: &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},
			_CheckCollectorAttr: &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true, // validated further below
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: castSchemaToTF(map[_SchemaAttr]*schema.Schema{
						_CheckCollectorIDAttr: &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateRegexp(_CheckCollectorIDAttr, config.BrokerCIDRegex),
						},
					}, _CheckCollectorDescriptions),
				},
			},
			_CheckStreamAttr: &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: castSchemaToTF(map[_SchemaAttr]*schema.Schema{
						_MetricNameAttr: &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateRegexp(_MetricNameAttr, `[\S]+`),
						},
						_MetricTagsAttr: &schema.Schema{
							Type:         schema.TypeMap,
							Optional:     true,
							ValidateFunc: validateTags,
						},
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
			_CheckJSONAttr: jsonAttr,
			// checkConfigAttr: &schema.Schema{
			// 	Type:     schema.TypeList,
			// 	Optional: true,
			// 	MaxItems: 1,
			// 	Elem: &schema.Resource{
			// 		Schema: map[string]*schema.Schema{
			// 			checkConfigAuthMethodAttr: &schema.Schema{
			// 				Type:         schema.TypeString,
			// 				Optional:     true,
			// 				Computed:     true,
			// 				ValidateFunc: validateRegexp(checkConfigAuthMethodAttr, `^(?:Basic|Digest|Auto)$`),
			// 			},
			// 			checkConfigAuthPasswordAttr: &schema.Schema{
			// 				Type:         schema.TypeString,
			// 				Optional:     true,
			// 				ValidateFunc: validateRegexp(checkConfigAuthPasswordAttr, `^.*`),
			// 			},
			// 			checkConfigAuthUserAttr: &schema.Schema{
			// 				Type:         schema.TypeString,
			// 				Optional:     true,
			// 				ValidateFunc: validateRegexp(checkConfigAuthUserAttr, `[^:]*`),
			// 			},
			// 			checkConfigCAChainAttr: &schema.Schema{
			// 				Type:         schema.TypeString,
			// 				Optional:     true,
			// 				ValidateFunc: validateRegexp(checkConfigCAChainAttr, `.+`),
			// 			},
			// 			checkConfigCertificateFileAttr: &schema.Schema{
			// 				Type:         schema.TypeString,
			// 				Optional:     true,
			// 				ValidateFunc: validateRegexp(checkConfigCertificateFileAttr, `.+`),
			// 			},
			// 			checkConfigCiphersAttr: &schema.Schema{
			// 				Type:         schema.TypeString,
			// 				Optional:     true,
			// 				Computed:     true,
			// 				ValidateFunc: validateRegexp(checkConfigCiphersAttr, `.+`),
			// 			},
			// 			checkConfigHTTPHeadersAttr: &schema.Schema{
			// 				Type:         schema.TypeMap,
			// 				Optional:     true,
			// 				ValidateFunc: validateHTTPHeaders,
			// 			},
			// 			checkConfigHTTPVersionAttr: &schema.Schema{
			// 				Type:         schema.TypeString,
			// 				Optional:     true,
			// 				Computed:     true,
			// 				ValidateFunc: validateHTTPVersion,
			// 			},
			// 			checkConfigKeyFileAttr: &schema.Schema{
			// 				Type:         schema.TypeString,
			// 				Optional:     true,
			// 				ValidateFunc: validateRegexp(checkConfigKeyFileAttr, `.+`),
			// 			},
			// 			checkConfigMethodAttr: &schema.Schema{
			// 				Type:         schema.TypeString,
			// 				Optional:     true,
			// 				Computed:     true,
			// 				ValidateFunc: validateRegexp(checkConfigMethodAttr, `\S+`),
			// 			},
			// 			checkConfigPayloadAttr: &schema.Schema{
			// 				Type:     schema.TypeString,
			// 				Optional: true,
			// 				Computed: true,
			// 			},
			// 			checkConfigPortAttr: &schema.Schema{
			// 				Type:     schema.TypeString, // NOTE(sean@): Why isn't this an Int on Circonus's side?  Are they doing an /etc/services lookup?  TODO: convert this to a TypeInt and force users in TF to do a map lookup.
			// 				Optional: true,
			// 				Computed: true,
			// 			},
			// 			checkConfigReadLimitAttr: &schema.Schema{
			// 				Type:     schema.TypeInt,
			// 				Optional: true,
			// 				Computed: true,
			// 				ValidateFunc: validateFuncs(
			// 					validateIntMin(checkConfigReadLimitAttr, 0),
			// 				),
			// 			},
			// 			checkConfigRedirectsAttr: &schema.Schema{
			// 				Type:     schema.TypeInt,
			// 				Optional: true,
			// 				Computed: true,
			// 				ValidateFunc: validateFuncs(
			// 					validateIntMin(checkConfigRedirectsAttr, 0),
			// 				),
			// 			},
			// 			checkConfigURLAttr: &schema.Schema{
			// 				Type:     schema.TypeString,
			// 				Optional: true,
			// 			},
			// 		},
			// 	},
			// },
			checkMetricLimitAttr: &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
				ValidateFunc: validateFuncs(
					validateIntMin(checkMetricLimitAttr, -1),
				),
			},
			checkMetricNamesAttr: &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			// checkMetricAttr: &schema.Schema{
			// 	Type:     schema.TypeList,
			// 	Required: true,
			// 	MinItems: 1,
			// 	Elem: &schema.Resource{
			// 		Schema: map[string]*schema.Schema{
			// 			checkMetricActiveAttr: &schema.Schema{
			// 				Type:     schema.TypeBool,
			// 				Optional: true,
			// 				Default:  true,
			// 			},
			// 			checkMetricNameAttr: &schema.Schema{
			// 				Type:     schema.TypeString,
			// 				Optional: true,
			// 				Computed: true,
			// 			},
			// 			checkMetricTagsAttr: &schema.Schema{
			// 				Type:         schema.TypeMap,
			// 				Optional:     true,
			// 				ValidateFunc: validateTags,
			// 			},
			// 			checkMetricTypeAttr: &schema.Schema{
			// 				Type:         schema.TypeString,
			// 				Required:     true,
			// 				ValidateFunc: validateMetricType,
			// 			},
			// 			checkMetricUnitsAttr: &schema.Schema{
			// 				Type:     schema.TypeString,
			// 				Optional: true,
			// 				Computed: true,
			// 			},
			// 		},
			// 	},
			// },
			checkNameAttr: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			checkNotesAttr: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				StateFunc: func(v interface{}) string {
					return strings.TrimSpace(v.(string))
				},
			},
			checkPeriodAttr: &schema.Schema{
				Type:      schema.TypeString,
				Optional:  true,
				Computed:  true,
				StateFunc: normalizeTimeDurationStringToSeconds,
				ValidateFunc: validateFuncs(
					validateDurationMin(checkPeriodAttr, defaultCirconusCheckPeriodMin),
					validateDurationMax(checkPeriodAttr, defaultCirconusCheckPeriodMax),
				),
			},
			checkTagsAttr: &schema.Schema{
				Type:         schema.TypeMap,
				Optional:     true,
				ValidateFunc: validateTags,
			},
			checkTargetAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validateHTTPURL(checkTargetAttr, _URLWithoutSchema|_URLWithoutPort),
			},
			checkTimeoutAttr: &schema.Schema{
				Type:     schema.TypeFloat,
				Optional: true,
				Computed: true,
				ValidateFunc: validateFuncs(
					validateDurationMin(checkTimeoutAttr, defaultCirconusTimeoutMin),
					validateDurationMax(checkTimeoutAttr, defaultCirconusTimeoutMax),
				),
			},
			checkTypeAttr: &schema.Schema{
				Type:         schema.TypeString,
				Computed:     true,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validateCheckType,
			},
		}, _CheckDescriptions),
	}
}

func _CheckCreate(d *schema.ResourceData, meta interface{}) error {
	c := _NewCheck()
	if err := c.ParseSchema(d, meta); err != nil {
		return errwrap.Wrapf("error parsing check schema during create: {{err}}", err)
	}

	ctxt := meta.(*providerContext)

	if err := c.Create(ctxt); err != nil {
		return errwrap.Wrapf("error creating check: {{err}}", err)
	}

	d.SetId(c.CID)

	return checkBundleRead(d, meta)
}

func checkBundleExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	ctxt := meta.(*providerContext)

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

func checkBundleRead(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*providerContext)

	cid := d.Id()
	cb, err := ctxt.client.FetchCheckBundle(api.CIDType(&cid))
	if err != nil {
		return err
	}

	var active bool
	switch cb.Status {
	case checkStatusActive:
		active = true
	case checkStatusDisabled:
		active = false
	default:
		return fmt.Errorf("check status unsupported: %q", cb.Status)
	}

	var checkConfig map[config.Key]interface{}
	{
		size, ok := defaultCheckTypeConfigSize[CheckType(cb.Type)]
		if !ok {
			size = defaultCheckTypeConfigSize[defaultCheckTypeName]
		}
		checkConfig = make(map[config.Key]interface{}, size)
	}

	if v, ok := cb.Config[checkConfigAuthMethodAttr]; ok {
		checkConfig[checkConfigAuthMethodAttr] = v
	}

	if v, ok := cb.Config[checkConfigAuthPasswordAttr]; ok {
		checkConfig[checkConfigAuthPasswordAttr] = v
	}

	if v, ok := cb.Config[checkConfigAuthUserAttr]; ok {
		checkConfig[checkConfigAuthUserAttr] = v
	}

	if v, ok := cb.Config[checkConfigCAChainAttr]; ok {
		checkConfig[checkConfigCAChainAttr] = v
	}

	if v, ok := cb.Config[checkConfigCertificateFileAttr]; ok {
		checkConfig[checkConfigCertificateFileAttr] = v
	}

	if v, ok := cb.Config[checkConfigCiphersAttr]; ok {
		checkConfig[checkConfigCiphersAttr] = v
	}

	httpHeaders := make(map[string]string, len(cb.Config))
	headerPrefixLen := len(config.HeaderPrefix)
	for k, v := range cb.Config {
		if len(k) <= headerPrefixLen {
			continue
		}

		if strings.Compare(string(k[:headerPrefixLen]), string(config.HeaderPrefix)) == 0 {
			key := k[headerPrefixLen:]
			httpHeaders[string(key)] = v
		}
	}
	checkConfig[checkConfigHTTPHeadersAttr] = httpHeaders

	if v, ok := cb.Config[checkConfigHTTPVersionAttr]; ok {
		checkConfig[checkConfigHTTPVersionAttr] = v
	}

	if v, ok := cb.Config[checkConfigKeyFileAttr]; ok {
		checkConfig[checkConfigKeyFileAttr] = v
	}

	if v, ok := cb.Config[checkConfigMethodAttr]; ok {
		checkConfig[checkConfigMethodAttr] = v
	}

	if v, ok := cb.Config[checkConfigPayloadAttr]; ok {
		checkConfig[checkConfigPayloadAttr] = v
	}

	if v, ok := cb.Config[checkConfigPortAttr]; ok {
		checkConfig[checkConfigPortAttr] = v
	}

	if v, ok := cb.Config[checkConfigReadLimitAttr]; ok {
		checkConfig[checkConfigReadLimitAttr] = v
	}

	if v, ok := cb.Config[checkConfigRedirectsAttr]; ok {
		checkConfig[checkConfigRedirectsAttr] = v
	}

	if v, ok := cb.Config[checkConfigURLAttr]; ok {
		checkConfig[checkConfigURLAttr] = v
	}

	// NOTE(sean@): todo
	metrics := map[string]interface{}{} // NOTE(sean@): TODO. Also populate checkMetricNamesAttr.
	metricNames := []interface{}{}

	stateSet(d, _CheckCollectorAttr, stringListToSet(cb.Brokers, _CheckCollectorIDAttr))
	d.Set(checkConfigAttr, []interface{}{checkConfig})
	d.Set(checkMetricNameAttr, cb.DisplayName)
	// NOTE(sean@): fixme
	_ = metrics
	// d.Set(checkBundleMetricsAttr, []interface{}{metrics})
	d.Set(checkMetricNamesAttr, metricNames)
	d.Set(checkNotesAttr, cb.Notes)
	d.Set(checkPeriodAttr, fmt.Sprintf("%ds", cb.Period))
	stateSet(d, _CheckActiveAttr, active)
	d.Set(checkTagsAttr, cb.Tags)
	d.Set(checkTargetAttr, cb.Target)
	d.Set(checkTimeoutAttr, cb.Timeout)
	d.Set(checkTypeAttr, cb.Type)

	// Out parameters
	d.Set(checkCheckUUIDsAttr, cb.CheckUUIDs)
	d.Set(checkChecksAttr, cb.Checks)
	d.Set(checkCreatedAttr, cb.Created)
	d.Set(checkLastModifiedAttr, cb.LastModified)
	d.Set(checkLastModifiedByAttr, cb.LastModifedBy)
	d.Set(checkReverseConnectURLsAttr, cb.ReverseConnectURLs)

	d.SetId(cb.CID)

	return nil
}

func checkBundleUpdate(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*providerContext)
	c := _NewCheck()

	if err := c.ParseSchema(d, meta); err != nil {
		return err
	}

	c.CID = d.Id()

	if err := c.Update(ctxt); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("unable to update check %q: {{err}}", d.Id()), err)
	}

	return checkBundleRead(d, meta)
}

func checkBundleDelete(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*providerContext)

	if _, err := ctxt.client.Delete(d.Id()); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("unable to delete check %q: {{err}}", d.Id()), err)
	}

	d.SetId("")

	return nil
}

var jsonAttr = &schema.Schema{
	Type:     schema.TypeSet,
	Optional: true,
	MaxItems: 1,
	MinItems: 1,
	Elem: &schema.Resource{
		Schema: castSchemaToTF(map[_SchemaAttr]*schema.Schema{
			_CheckJSONHTTPHeadersAttr: &schema.Schema{
				Type:         schema.TypeMap,
				Optional:     true,
				ValidateFunc: validateHTTPHeaders,
			},
			_CheckJSONHTTPVersionAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validateHTTPVersion,
			},
			_CheckJSONMethodAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validateRegexp(checkConfigMethodAttr, `\S+`),
			},
			_CheckJSONPortAttr: &schema.Schema{
				Type:     schema.TypeString, // NOTE(sean@): Why isn't this an Int on Circonus's side?  Are they doing an /etc/services lookup?  TODO: convert this to a TypeInt and force users in TF to do a map lookup.
				Optional: true,
				Computed: true,
			},
			_CheckJSONReadLimitAttr: &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
				ValidateFunc: validateFuncs(
					validateIntMin(_CheckJSONReadLimitAttr, 0),
				),
			},
			_CheckJSONRedirectsAttr: &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
				ValidateFunc: validateFuncs(
					validateIntMin(_CheckJSONRedirectsAttr, 0),
				),
			},
			_CheckJSONURLAttr: &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: validateFuncs(
					validateHTTPURL(_CheckJSONURLAttr, _URLIsAbs),
				),
			},
		}, _CheckJSONDescriptions),
	},
}

const (
	// Attributes in circonus_check
	checkConfigAttr      = "config"
	checkMetricLimitAttr = "metric_limit"
	checkMetricAttr      = "metric"
	checkMetricNamesAttr = "metric_names"
	checkNameAttr        = "name"
	checkNotesAttr       = "notes"
	checkPeriodAttr      = "period"
	checkTagsAttr        = "tags"
	checkTargetAttr      = "target"
	checkTimeoutAttr     = "timeout"
	checkTypeAttr        = "type"
)

const (
	// Out parameters for circonus_check
	checkID                     = "id"
	checkCheckUUIDsAttr         = "uuids"
	checkChecksAttr             = "checks"
	checkCreatedAttr            = "created"
	checkLastModifiedAttr       = "last_modified"
	checkLastModifiedByAttr     = "last_modified_by"
	checkReverseConnectURLsAttr = "reverse_connect_urls"
)

const (
	// Attributes in circonus_check.config
	checkConfigAuthMethodAttr      = "auth_method"
	checkConfigAuthPasswordAttr    = "auth_password"
	checkConfigAuthUserAttr        = "auth_user"
	checkConfigCAChainAttr         = "ca_chain"
	checkConfigCertificateFileAttr = "certificate_file"
	checkConfigCiphersAttr         = "ciphers"
	checkConfigHTTPHeadersAttr     = "http_headers"
	checkConfigHTTPVersionAttr     = "http_version"
	checkConfigKeyFileAttr         = "key_file"
	checkConfigMethodAttr          = "method"
	checkConfigPayloadAttr         = "payload"
	checkConfigPortAttr            = "port"
	checkConfigReadLimitAttr       = "read_limit"
	checkConfigRedirectsAttr       = "redirects"
	checkConfigURLAttr             = "url"
)

const (
	// Attributes in circonus_check.metric
	checkMetricNameAttr   = "name"
	checkMetricActiveAttr = "active"
	checkMetricTypeAttr   = "type"
	checkMetricUnitsAttr  = "units"
	checkMetricTagsAttr   = "tags"
)

const (
	// CheckBundle.Status can be one of these values
	checkStatusActive   = "active"
	checkStatusDisabled = "disabled"
)

func makeCheckBundleConfig(checkType CheckType) api.CheckBundleConfig {
	if size, ok := defaultCheckTypeConfigSize[checkType]; ok {
		return make(api.CheckBundleConfig, size)
	}

	return make(api.CheckBundleConfig, defaultCheckTypeConfigSize[defaultCheckTypeName])
}
