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
 *     named check*Attr.
 *
 * 2) API Objects into Statefile data.  api*Attr named constants are parameters
 *    that originate from the API and need to be mapped into the provider's
 *    vernacular.
 */

import (
	"fmt"
	"time"

	"github.com/circonus-labs/circonus-gometrics/api"
	"github.com/circonus-labs/circonus-gometrics/api/config"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
)

const (
	// circonus_check.* global resource attribute names
	checkActiveAttr      = "active"
	checkCAQLAttr        = "caql"
	checkCloudWatchAttr  = "cloudwatch"
	checkCollectorAttr   = "collector"
	checkConsulAttr      = "consul"
	checkHTTPAttr        = "http"
	checkHTTPTrapAttr    = "httptrap"
	checkICMPPingAttr    = "icmp_ping"
	checkJSONAttr        = "json"
	checkMetricAttr      = "metric"
	checkMetricLimitAttr = "metric_limit"
	checkMySQLAttr       = "mysql"
	checkNameAttr        = "name"
	checkNotesAttr       = "notes"
	checkPeriodAttr      = "period"
	checkPostgreSQLAttr  = "postgresql"
	checkStatsdAttr      = "statsd"
	checkTCPAttr         = "tcp"
	checkTagsAttr        = "tags"
	checkTargetAttr      = "target"
	checkTimeoutAttr     = "timeout"
	checkTypeAttr        = "type"

	// circonus_check.collector.* resource attribute names
	checkCollectorIDAttr = "id"

	// circonus_check.metric.* resource attribute names are aliased to
	// circonus_metric.* resource attributes.

	// circonus_check.metric.* resource attribute names
	// metricIDAttr  = "id"

	// Out parameters for circonus_check
	checkOutByCollectorAttr        = "check_by_collector"
	checkOutIDAttr                 = "check_id"
	checkOutChecksAttr             = "checks"
	checkOutCreatedAttr            = "created"
	checkOutLastModifiedAttr       = "last_modified"
	checkOutLastModifiedByAttr     = "last_modified_by"
	checkOutReverseConnectURLsAttr = "reverse_connect_urls"
	checkOutCheckUUIDsAttr         = "uuids"
)

const (
	// Circonus API constants from their API endpoints
	apiCheckTypeCAQLAttr       apiCheckType = "caql"
	apiCheckTypeCloudWatchAttr apiCheckType = "cloudwatch"
	apiCheckTypeConsulAttr     apiCheckType = "consul"
	apiCheckTypeHTTPAttr       apiCheckType = "http"
	apiCheckTypeHTTPTrapAttr   apiCheckType = "httptrap"
	apiCheckTypeICMPPingAttr   apiCheckType = "ping_icmp"
	apiCheckTypeJSONAttr       apiCheckType = "json"
	apiCheckTypeMySQLAttr      apiCheckType = "mysql"
	apiCheckTypePostgreSQLAttr apiCheckType = "postgres"
	apiCheckTypeStatsdAttr     apiCheckType = "statsd"
	apiCheckTypeTCPAttr        apiCheckType = "tcp"
)

var checkDescriptions = attrDescrs{
	checkActiveAttr:      "If the check is activate or disabled",
	checkCAQLAttr:        "CAQL check configuration",
	checkCloudWatchAttr:  "CloudWatch check configuration",
	checkCollectorAttr:   "The collector(s) that are responsible for gathering the metrics",
	checkConsulAttr:      "Consul check configuration",
	checkHTTPAttr:        "HTTP check configuration",
	checkHTTPTrapAttr:    "HTTP Trap check configuration",
	checkICMPPingAttr:    "ICMP ping check configuration",
	checkJSONAttr:        "JSON check configuration",
	checkMetricAttr:      "Configuration for a stream of metrics",
	checkMetricLimitAttr: `Setting a metric_limit will enable all (-1), disable (0), or allow up to the specified limit of metrics for this check ("N+", where N is a positive integer)`,
	checkMySQLAttr:       "MySQL check configuration",
	checkNameAttr:        "The name of the check bundle that will be displayed in the web interface",
	checkNotesAttr:       "Notes about this check bundle",
	checkPeriodAttr:      "The period between each time the check is made",
	checkPostgreSQLAttr:  "PostgreSQL check configuration",
	checkStatsdAttr:      "statsd check configuration",
	checkTCPAttr:         "TCP check configuration",
	checkTagsAttr:        "A list of tags assigned to the check",
	checkTargetAttr:      "The target of the check (e.g. hostname, URL, IP, etc)",
	checkTimeoutAttr:     "The length of time in seconds (and fractions of a second) before the check will timeout if no response is returned to the collector",
	checkTypeAttr:        "The check type",

	checkOutByCollectorAttr:        "",
	checkOutCheckUUIDsAttr:         "",
	checkOutChecksAttr:             "",
	checkOutCreatedAttr:            "",
	checkOutIDAttr:                 "",
	checkOutLastModifiedAttr:       "",
	checkOutLastModifiedByAttr:     "",
	checkOutReverseConnectURLsAttr: "",
}

var checkCollectorDescriptions = attrDescrs{
	checkCollectorIDAttr: "The ID of the collector",
}

var checkMetricDescriptions = metricDescriptions

func resourceCheck() *schema.Resource {
	return &schema.Resource{
		Create: checkCreate,
		Read:   checkRead,
		Update: checkUpdate,
		Delete: checkDelete,
		Exists: checkExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: convertToHelperSchema(checkDescriptions, map[schemaAttr]*schema.Schema{
			checkActiveAttr: &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			checkCAQLAttr:       schemaCheckCAQL,
			checkCloudWatchAttr: schemaCheckCloudWatch,
			checkCollectorAttr: &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: convertToHelperSchema(checkCollectorDescriptions, map[schemaAttr]*schema.Schema{
						checkCollectorIDAttr: &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateRegexp(checkCollectorIDAttr, config.BrokerCIDRegex),
						},
					}),
				},
			},
			checkConsulAttr:   schemaCheckConsul,
			checkHTTPAttr:     schemaCheckHTTP,
			checkHTTPTrapAttr: schemaCheckHTTPTrap,
			checkJSONAttr:     schemaCheckJSON,
			checkICMPPingAttr: schemaCheckICMPPing,
			checkMetricAttr: &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Set:      checkMetricChecksum,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: convertToHelperSchema(checkMetricDescriptions, map[schemaAttr]*schema.Schema{
						metricActiveAttr: &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						metricNameAttr: &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateRegexp(metricNameAttr, `[\S]+`),
						},
						metricTagsAttr: tagMakeConfigSchema(metricTagsAttr),
						metricTypeAttr: &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateMetricType,
						},
						metricUnitAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							Default:      metricUnit,
							ValidateFunc: validateRegexp(metricUnitAttr, metricUnitRegexp),
						},
					}),
				},
			},
			checkMetricLimitAttr: &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
				ValidateFunc: validateFuncs(
					validateIntMin(checkMetricLimitAttr, -1),
				),
			},
			checkMySQLAttr: schemaCheckMySQL,
			checkNameAttr: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			checkNotesAttr: &schema.Schema{
				Type:      schema.TypeString,
				Optional:  true,
				Computed:  true,
				StateFunc: suppressWhitespace,
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
			checkPostgreSQLAttr: schemaCheckPostgreSQL,
			checkStatsdAttr:     schemaCheckStatsd,
			checkTagsAttr:       tagMakeConfigSchema(checkTagsAttr),
			checkTargetAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validateRegexp(checkTagsAttr, `.+`),
			},
			checkTCPAttr: schemaCheckTCP,
			checkTimeoutAttr: &schema.Schema{
				Type:      schema.TypeString,
				Optional:  true,
				Computed:  true,
				StateFunc: normalizeTimeDurationStringToSeconds,
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

			// Out parameters
			checkOutIDAttr: &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			checkOutByCollectorAttr: &schema.Schema{
				Type:     schema.TypeMap,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			checkOutCheckUUIDsAttr: &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			checkOutChecksAttr: &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			checkOutCreatedAttr: &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},
			checkOutLastModifiedAttr: &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},
			checkOutLastModifiedByAttr: &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			checkOutReverseConnectURLsAttr: &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		}),
	}
}

func checkCreate(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*providerContext)
	c := newCheck()
	if err := c.ParseConfig(d); err != nil {
		return errwrap.Wrapf("error parsing check schema during create: {{err}}", err)
	}

	if err := c.Create(ctxt); err != nil {
		return errwrap.Wrapf("error creating check: {{err}}", err)
	}

	d.SetId(c.CID)

	return checkRead(d, meta)
}

func checkExists(d *schema.ResourceData, meta interface{}) (bool, error) {
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

// checkRead pulls data out of the CheckBundle object and stores it into the
// appropriate place in the statefile.
func checkRead(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*providerContext)

	cid := d.Id()
	c, err := loadCheck(ctxt, api.CIDType(&cid))
	if err != nil {
		return err
	}

	d.SetId(c.CID)

	// Global circonus_check attributes are saved first, followed by the check
	// type specific attributes handled below in their respective checkRead*().

	checkIDsByCollector := make(map[string]interface{}, len(c.Checks))
	for i, b := range c.Brokers {
		checkIDsByCollector[b] = c.Checks[i]
	}

	var checkID string
	if len(c.Checks) == 1 {
		checkID = c.Checks[0]
	}

	metrics := schema.NewSet(checkMetricChecksum, nil)
	for _, m := range c.Metrics {
		metricAttrs := map[string]interface{}{
			string(metricActiveAttr): metricAPIStatusToBool(m.Status),
			string(metricNameAttr):   m.Name,
			string(metricTagsAttr):   tagsToState(apiToTags(m.Tags)),
			string(metricTypeAttr):   m.Type,
			string(metricUnitAttr):   indirect(m.Units),
		}

		metrics.Add(metricAttrs)
	}

	// Write the global circonus_check parameters followed by the check
	// type-specific parameters.

	d.Set(checkActiveAttr, checkAPIStatusToBool(c.Status))

	if err := d.Set(checkCollectorAttr, stringListToSet(c.Brokers, checkCollectorIDAttr)); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Unable to store check %q attribute: {{err}}", checkCollectorAttr), err)
	}

	d.Set(checkMetricLimitAttr, c.MetricLimit)
	d.Set(checkNameAttr, c.DisplayName)
	d.Set(checkNotesAttr, c.Notes)
	d.Set(checkPeriodAttr, fmt.Sprintf("%ds", c.Period))

	if err := d.Set(checkMetricAttr, metrics); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Unable to store check %q attribute: {{err}}", checkMetricAttr), err)
	}

	if err := d.Set(checkTagsAttr, c.Tags); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Unable to store check %q attribute: {{err}}", checkTagsAttr), err)
	}

	d.Set(checkTargetAttr, c.Target)

	{
		t, _ := time.ParseDuration(fmt.Sprintf("%fs", c.Timeout))
		d.Set(checkTimeoutAttr, t.String())
	}

	d.Set(checkTypeAttr, c.Type)

	// Last step: parse a check_bundle's config into the statefile.
	if err := parseCheckTypeConfig(&c, d); err != nil {
		return errwrap.Wrapf("Unable to parse check config: {{err}}", err)
	}

	// Out parameters
	if err := d.Set(checkOutByCollectorAttr, checkIDsByCollector); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Unable to store check %q attribute: {{err}}", checkOutByCollectorAttr), err)
	}

	if err := d.Set(checkOutCheckUUIDsAttr, c.CheckUUIDs); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Unable to store check %q attribute: {{err}}", checkOutCheckUUIDsAttr), err)
	}

	if err := d.Set(checkOutChecksAttr, c.Checks); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Unable to store check %q attribute: {{err}}", checkOutChecksAttr), err)
	}

	if checkID != "" {
		d.Set(checkOutIDAttr, checkID)
	}

	d.Set(checkOutCreatedAttr, c.Created)
	d.Set(checkOutLastModifiedAttr, c.LastModified)
	d.Set(checkOutLastModifiedByAttr, c.LastModifedBy)

	if err := d.Set(checkOutReverseConnectURLsAttr, c.ReverseConnectURLs); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Unable to store check %q attribute: {{err}}", checkOutReverseConnectURLsAttr), err)
	}

	return nil
}

func checkUpdate(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*providerContext)
	c := newCheck()
	if err := c.ParseConfig(d); err != nil {
		return err
	}

	c.CID = d.Id()
	if err := c.Update(ctxt); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("unable to update check %q: {{err}}", d.Id()), err)
	}

	return checkRead(d, meta)
}

func checkDelete(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*providerContext)

	if _, err := ctxt.client.Delete(d.Id()); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("unable to delete check %q: {{err}}", d.Id()), err)
	}

	d.SetId("")

	return nil
}

func checkMetricChecksum(v interface{}) int {
	m := v.(map[string]interface{})
	csum := metricChecksum(m)
	return csum
}

// ParseConfig reads Terraform config data and stores the information into a
// Circonus CheckBundle object.
func (c *circonusCheck) ParseConfig(d *schema.ResourceData) error {
	if v, found := d.GetOk(checkActiveAttr); found {
		c.Status = checkActiveToAPIStatus(v.(bool))
	}

	if v, found := d.GetOk(checkCollectorAttr); found {
		l := v.(*schema.Set).List()
		c.Brokers = make([]string, 0, len(l))

		for _, mapRaw := range l {
			mapAttrs := mapRaw.(map[string]interface{})

			if mv, mapFound := mapAttrs[checkCollectorIDAttr]; mapFound {
				c.Brokers = append(c.Brokers, mv.(string))
			}
		}
	}

	if v, found := d.GetOk(checkMetricLimitAttr); found {
		c.MetricLimit = v.(int)
	}

	if v, found := d.GetOk(checkNameAttr); found {
		c.DisplayName = v.(string)
	}

	if v, found := d.GetOk(checkNotesAttr); found {
		s := v.(string)
		c.Notes = &s
	}

	if v, found := d.GetOk(checkPeriodAttr); found {
		d, err := time.ParseDuration(v.(string))
		if err != nil {
			return errwrap.Wrapf(fmt.Sprintf("unable to parse %q as a duration: {{err}}", checkPeriodAttr), err)
		}

		c.Period = uint(d.Seconds())
	}

	if v, found := d.GetOk(checkMetricAttr); found {
		metricList := v.(*schema.Set).List()
		c.Metrics = make([]api.CheckBundleMetric, 0, len(metricList))

		for _, metricListRaw := range metricList {
			metricAttrs := metricListRaw.(map[string]interface{})

			var id string
			if av, found := metricAttrs[metricIDAttr]; found {
				id = av.(string)
			} else {
				var err error
				id, err = newMetricID()
				if err != nil {
					return errwrap.Wrapf("unable to create a new metric ID: {{err}}", err)
				}
			}

			m := newMetric()
			if err := m.ParseConfigMap(id, metricAttrs); err != nil {
				return errwrap.Wrapf("unable to parse config: {{err}}", err)
			}

			c.Metrics = append(c.Metrics, m.CheckBundleMetric)
		}
	}

	if v, found := d.GetOk(checkTagsAttr); found {
		c.Tags = derefStringList(flattenSet(v.(*schema.Set)))
	}

	if v, found := d.GetOk(checkTargetAttr); found {
		c.Target = v.(string)
	}

	if v, found := d.GetOk(checkTimeoutAttr); found {
		d, err := time.ParseDuration(v.(string))
		if err != nil {
			return errwrap.Wrapf(fmt.Sprintf("unable to parse %q as a duration: {{err}}", checkTimeoutAttr), err)
		}

		t := float32(d.Seconds())
		c.Timeout = t
	}

	// Last step: parse the individual check types
	if err := checkConfigToAPI(c, d); err != nil {
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

// checkConfigToAPI parses the Terraform config into the respective per-check
// type api.Config attributes.
func checkConfigToAPI(c *circonusCheck, d *schema.ResourceData) error {
	checkTypeParseMap := map[string]func(*circonusCheck, interfaceList) error{
		checkCAQLAttr:       checkConfigToAPICAQL,
		checkCloudWatchAttr: checkConfigToAPICloudWatch,
		checkConsulAttr:     checkConfigToAPIConsul,
		checkHTTPAttr:       checkConfigToAPIHTTP,
		checkHTTPTrapAttr:   checkConfigToAPIHTTPTrap,
		checkICMPPingAttr:   checkConfigToAPIICMPPing,
		checkJSONAttr:       checkConfigToAPIJSON,
		checkMySQLAttr:      checkConfigToAPIMySQL,
		checkPostgreSQLAttr: checkConfigToAPIPostgreSQL,
		checkStatsdAttr:     checkConfigToAPIStatsd,
		checkTCPAttr:        checkConfigToAPITCP,
	}

	for checkType, fn := range checkTypeParseMap {
		if listRaw, found := d.GetOk(checkType); found {
			switch u := listRaw.(type) {
			case []interface{}:
				if err := fn(c, u); err != nil {
					return errwrap.Wrapf(fmt.Sprintf("Unable to parse type %q: {{err}}", string(checkType)), err)
				}
			case *schema.Set:
				if err := fn(c, u.List()); err != nil {
					return errwrap.Wrapf(fmt.Sprintf("Unable to parse type %q: {{err}}", string(checkType)), err)
				}
			default:
				return fmt.Errorf("PROVIDER BUG: unsupported check type interface: %q", checkType)
			}
		}
	}

	return nil
}

// parseCheckTypeConfig parses an API Config object and stores the result in the
// statefile.
func parseCheckTypeConfig(c *circonusCheck, d *schema.ResourceData) error {
	checkTypeConfigHandlers := map[apiCheckType]func(*circonusCheck, *schema.ResourceData) error{
		apiCheckTypeCAQLAttr:       checkAPIToStateCAQL,
		apiCheckTypeCloudWatchAttr: checkAPIToStateCloudWatch,
		apiCheckTypeConsulAttr:     checkAPIToStateConsul,
		apiCheckTypeHTTPAttr:       checkAPIToStateHTTP,
		apiCheckTypeHTTPTrapAttr:   checkAPIToStateHTTPTrap,
		apiCheckTypeICMPPingAttr:   checkAPIToStateICMPPing,
		apiCheckTypeJSONAttr:       checkAPIToStateJSON,
		apiCheckTypeMySQLAttr:      checkAPIToStateMySQL,
		apiCheckTypePostgreSQLAttr: checkAPIToStatePostgreSQL,
		apiCheckTypeStatsdAttr:     checkAPIToStateStatsd,
		apiCheckTypeTCPAttr:        checkAPIToStateTCP,
	}

	var checkType apiCheckType = apiCheckType(c.Type)
	fn, ok := checkTypeConfigHandlers[checkType]
	if !ok {
		return fmt.Errorf("check type %q not supported", c.Type)
	}

	if err := fn(c, d); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("unable to parse the API config for %q: {{err}}", c.Type), err)
	}

	return nil
}
