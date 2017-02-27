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
	checkActiveAttr      schemaAttr = "active"
	checkCAQLAttr        schemaAttr = "caql"
	checkCloudWatchAttr  schemaAttr = "cloudwatch"
	checkCollectorAttr   schemaAttr = "collector"
	checkHTTPAttr        schemaAttr = "http"
	checkHTTPTrapAttr    schemaAttr = "httptrap"
	checkICMPPingAttr    schemaAttr = "icmp_ping"
	checkJSONAttr        schemaAttr = "json"
	checkMetricLimitAttr schemaAttr = "metric_limit"
	checkMySQLAttr       schemaAttr = "mysql"
	checkNameAttr        schemaAttr = "name"
	checkNotesAttr       schemaAttr = "notes"
	checkPeriodAttr      schemaAttr = "period"
	checkPostgreSQLAttr  schemaAttr = "postgresql"
	checkStreamAttr      schemaAttr = "stream"
	checkTagsAttr        schemaAttr = "tags"
	checkTargetAttr      schemaAttr = "target"
	checkTCPAttr         schemaAttr = "tcp"
	checkTimeoutAttr     schemaAttr = "timeout"
	checkTypeAttr        schemaAttr = "type"

	// circonus_check.collector.* resource attribute names
	checkCollectorIDAttr schemaAttr = "id"

	// circonus_check.stream.* resource attribute names are aliased to
	// circonus_metric.* resource attributes.

	// circonus_check.streams.* resource attribute names
	// metricIDAttr schemaAttr = "id"

	// Out parameters for circonus_check
	checkOutByCollectorAttr        schemaAttr = "check_by_collector"
	checkOutCheckUUIDsAttr         schemaAttr = "uuids"
	checkOutChecksAttr             schemaAttr = "checks"
	checkOutCreatedAttr            schemaAttr = "created"
	checkOutLastModifiedAttr       schemaAttr = "last_modified"
	checkOutLastModifiedByAttr     schemaAttr = "last_modified_by"
	checkOutReverseConnectURLsAttr schemaAttr = "reverse_connect_urls"
)

const (
	// Circonus API constants from their API endpoints
	apiCheckTypeCAQLAttr       apiCheckType = "caql"
	apiCheckTypeCloudWatchAttr apiCheckType = "cloudwatch"
	apiCheckTypeHTTPAttr       apiCheckType = "http"
	apiCheckTypeHTTPTrapAttr   apiCheckType = "httptrap"
	apiCheckTypeICMPPingAttr   apiCheckType = "ping_icmp"
	apiCheckTypeJSONAttr       apiCheckType = "json"
	apiCheckTypeMySQLAttr      apiCheckType = "mysql"
	apiCheckTypePostgreSQLAttr apiCheckType = "postgres"
	apiCheckTypeTCPAttr        apiCheckType = "tcp"
)

var checkDescriptions = attrDescrs{
	checkActiveAttr:      "If the check is activate or disabled",
	checkCAQLAttr:        "CAQL check configuration",
	checkCloudWatchAttr:  "CloudWatch check configuration",
	checkCollectorAttr:   "The collector(s) that are responsible for gathering the metrics",
	checkHTTPAttr:        "HTTP check configuration",
	checkHTTPTrapAttr:    "HTTP Trap check configuration",
	checkICMPPingAttr:    "ICMP ping check configuration",
	checkJSONAttr:        "JSON check configuration",
	checkMetricLimitAttr: `Setting a metric_limit will enable all (-1), disable (0), or allow up to the specified limit of metrics for this check ("N+", where N is a positive integer)`,
	checkMySQLAttr:       "MySQL check configuration",
	checkNameAttr:        "The name of the check bundle that will be displayed in the web interface",
	checkNotesAttr:       "Notes about this check bundle",
	checkPeriodAttr:      "The period between each time the check is made",
	checkPostgreSQLAttr:  "PostgreSQL check configuration",
	checkStreamAttr:      "Configuration for a stream of metrics",
	checkTagsAttr:        "A list of tags assigned to the check",
	checkTargetAttr:      "The target of the check (e.g. hostname, URL, IP, etc)",
	checkTCPAttr:         "TCP check configuration",
	checkTimeoutAttr:     "The length of time in seconds (and fractions of a second) before the check will timeout if no response is returned to the collector",
	checkTypeAttr:        "The check type",

	checkOutChecksAttr:             "",
	checkOutByCollectorAttr:        "",
	checkOutCheckUUIDsAttr:         "",
	checkOutCreatedAttr:            "",
	checkOutLastModifiedAttr:       "",
	checkOutLastModifiedByAttr:     "",
	checkOutReverseConnectURLsAttr: "",
}

var checkCollectorDescriptions = attrDescrs{
	checkCollectorIDAttr: "The ID of the collector",
}

var checkStreamDescriptions = metricDescriptions

func newCheckResource() *schema.Resource {
	return &schema.Resource{
		Create: checkCreate,
		Read:   checkRead,
		Update: checkUpdate,
		Delete: checkDelete,
		Exists: checkExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: castSchemaToTF(map[schemaAttr]*schema.Schema{
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
					Schema: castSchemaToTF(map[schemaAttr]*schema.Schema{
						checkCollectorIDAttr: &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateRegexp(checkCollectorIDAttr, config.BrokerCIDRegex),
						},
					}, checkCollectorDescriptions),
				},
			},
			checkHTTPAttr:     schemaCheckHTTP,
			checkHTTPTrapAttr: schemaCheckHTTPTrap,
			checkJSONAttr:     schemaCheckJSON,
			checkICMPPingAttr: schemaCheckICMPPing,
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
			checkStreamAttr: &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Set:      checkStreamChecksum,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: castSchemaToTF(map[schemaAttr]*schema.Schema{
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
					}, checkStreamDescriptions),
				},
			},
			checkTagsAttr: tagMakeConfigSchema(checkTagsAttr),
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
		}, checkDescriptions),
	}
}

func checkCreate(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*providerContext)
	c := newCheck()
	cr := newConfigReader(ctxt, d)
	if err := c.ParseConfig(cr); err != nil {
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

	streams := schema.NewSet(checkStreamChecksum, nil)
	for _, m := range c.Metrics {
		streamAttrs := map[string]interface{}{
			string(metricActiveAttr): metricAPIStatusToBool(m.Status),
			string(metricNameAttr):   m.Name,
			string(metricTagsAttr):   tagsToState(apiToTags(m.Tags)),
			string(metricTypeAttr):   m.Type,
			string(metricUnitAttr):   indirect(m.Units),
		}

		streams.Add(streamAttrs)
	}

	// Write the global circonus_check parameters followed by the check
	// type-specific parameters.

	stateSet(d, checkActiveAttr, checkAPIStatusToBool(c.Status))
	stateSet(d, checkCollectorAttr, stringListToSet(c.Brokers, checkCollectorIDAttr))
	stateSet(d, checkMetricLimitAttr, c.MetricLimit)
	stateSet(d, checkNameAttr, c.DisplayName)
	stateSet(d, checkNotesAttr, c.Notes)
	stateSet(d, checkPeriodAttr, fmt.Sprintf("%ds", c.Period))
	stateSet(d, checkStreamAttr, streams)
	stateSet(d, checkTagsAttr, c.Tags)
	stateSet(d, checkTargetAttr, c.Target)
	{
		t, _ := time.ParseDuration(fmt.Sprintf("%fs", c.Timeout))
		stateSet(d, checkTimeoutAttr, t.String())
	}
	stateSet(d, checkTypeAttr, c.Type)

	// Last step: parse a check_bundle's config into the statefile.
	if err := parseCheckTypeConfig(&c, d); err != nil {
		return errwrap.Wrapf("Unable to parse check config: {{err}}", err)
	}

	// Out parameters
	stateSet(d, checkOutByCollectorAttr, checkIDsByCollector)
	stateSet(d, checkOutCheckUUIDsAttr, c.CheckUUIDs)
	stateSet(d, checkOutChecksAttr, c.Checks)
	stateSet(d, checkOutCreatedAttr, c.Created)
	stateSet(d, checkOutLastModifiedAttr, c.LastModified)
	stateSet(d, checkOutLastModifiedByAttr, c.LastModifedBy)
	stateSet(d, checkOutReverseConnectURLsAttr, c.ReverseConnectURLs)

	return nil
}

func checkUpdate(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*providerContext)
	c := newCheck()
	cr := newConfigReader(ctxt, d)
	if err := c.ParseConfig(cr); err != nil {
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

func checkStreamChecksum(v interface{}) int {
	m := v.(map[string]interface{})

	ar := newMapReader(nil, m)
	csum := metricChecksum(ar)
	return csum
}

// ParseConfig reads Terraform config data and stores the information into a
// Circonus CheckBundle object.
func (c *circonusCheck) ParseConfig(ar attrReader) error {
	if status, ok := ar.GetBoolOK(checkActiveAttr); ok {
		c.Status = checkActiveToAPIStatus(status)
	}

	if collectorsList, ok := ar.GetSetAsListOK(checkCollectorAttr); ok {
		c.Brokers = collectorsList.CollectList(checkCollectorIDAttr)
	}

	if i, ok := ar.GetIntOK(checkMetricLimitAttr); ok {
		c.MetricLimit = i
	}

	if name, ok := ar.GetStringOK(checkNameAttr); ok {
		c.DisplayName = name
	}

	c.Notes = ar.GetStringPtr(checkNotesAttr)

	if d, ok := ar.GetDurationOK(checkPeriodAttr); ok {
		c.Period = uint(d.Seconds())
	}

	if streamList, ok := ar.GetSetAsListOK(checkStreamAttr); ok {
		c.Metrics = make([]api.CheckBundleMetric, 0, len(streamList))

		for _, metricListRaw := range streamList {
			metricAttrs := newInterfaceMap(metricListRaw)
			mr := newMapReader(ar.Context(), metricAttrs)

			var id string
			if v, ok := mr.GetStringOK(metricIDAttr); ok {
				id = v
			} else {
				var err error
				id, err = newMetricID()
				if err != nil {
					return errwrap.Wrapf("unable to create a new metric ID: {{err}}", err)
				}
			}

			m := newMetric()
			if err := m.ParseConfig(id, mr); err != nil {
				return errwrap.Wrapf("unable to parse config: {{err}}", err)
			}

			c.Metrics = append(c.Metrics, m.CheckBundleMetric)
		}
	}

	c.Tags = tagsToAPI(ar.GetTags(checkTagsAttr))

	if s, ok := ar.GetStringOK(checkTargetAttr); ok {
		c.Target = s
	}

	if d, ok := ar.GetDurationOK(checkTimeoutAttr); ok {
		t := float32(d.Seconds())
		c.Timeout = t
	}

	// Last step: parse the individual check types
	if err := checkConfigToAPI(c, ar); err != nil {
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
func checkConfigToAPI(c *circonusCheck, ar attrReader) error {
	checkTypeParseMap := map[schemaAttr]func(*circonusCheck, *providerContext, interfaceList) error{
		checkCAQLAttr:       checkConfigToAPICAQL,
		checkCloudWatchAttr: checkConfigToAPICloudWatch,
		checkHTTPAttr:       checkConfigToAPIHTTP,
		checkHTTPTrapAttr:   checkConfigToAPIHTTPTrap,
		checkICMPPingAttr:   checkConfigToAPIICMPPing,
		checkJSONAttr:       checkConfigToAPIJSON,
		checkMySQLAttr:      checkConfigToAPIMySQL,
		checkPostgreSQLAttr: checkConfigToAPIPostgreSQL,
		checkTCPAttr:        checkConfigToAPITCP,
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

// parseCheckTypeConfig parses an API Config object and stores the result in the
// statefile.
func parseCheckTypeConfig(c *circonusCheck, d *schema.ResourceData) error {
	checkTypeConfigHandlers := map[apiCheckType]func(*circonusCheck, *schema.ResourceData) error{
		apiCheckTypeCAQLAttr:       checkAPIToStateCAQL,
		apiCheckTypeCloudWatchAttr: checkAPIToStateCloudWatch,
		apiCheckTypeHTTPAttr:       checkAPIToStateHTTP,
		apiCheckTypeHTTPTrapAttr:   checkAPIToStateHTTPTrap,
		apiCheckTypeICMPPingAttr:   checkAPIToStateICMPPing,
		apiCheckTypeJSONAttr:       checkAPIToStateJSON,
		apiCheckTypeMySQLAttr:      checkAPIToStateMySQL,
		apiCheckTypePostgreSQLAttr: checkAPIToStatePostgreSQL,
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
