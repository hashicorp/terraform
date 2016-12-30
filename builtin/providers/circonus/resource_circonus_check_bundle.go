package circonus

import (
	"errors"
	"fmt"
	"strings"

	"github.com/circonus-labs/circonus-gometrics/api"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/y0ssar1an/q"
)

const (
	// Attributes in circonus_check_bundle
	checkBundleActiveAttr             = "active"
	checkBundleID                     = "id"
	checkBundleBrokersAttr            = "brokers"
	checkBundleCheckUUIDsAttr         = "uuids"
	checkBundleChecksAttr             = "checks"
	checkBundleConfigAttr             = "config"
	checkBundleMetricsAttr            = "metrics"
	checkBundleNameAttr               = "name"
	checkBundleNotesAttr              = "notes"
	checkBundleCreatedAttr            = "created"
	checkBundleLastModifiedAttr       = "last_modified"
	checkBundleLastModifiedByAttr     = "last_modified_by"
	checkBundleReverseConnectURLsAttr = "reverse_connect_urls"
	checkBundlePeriodAttr             = "period"
	checkBundleTagsAttr               = "tags"
	checkBundleTargetAttr             = "target"
	checkBundleTimeoutAttr            = "timeout"
	checkBundleTypeAttr               = "type"

	// Attributes in circonus_check_bundle.config
	checkBundleConfigAsyncAttr         = "async_metrics"
	checkBundleConfigHTTPVersionAttr   = "http_version"
	checkBundleConfigMethodAttr        = "method"
	checkBundleConfigPayloadAttr       = "payload"
	checkBundleConfigPortAttr          = "port"
	checkBundleConfigReadLimitAttr     = "read_limit"
	checkBundleConfigSecretAttr        = "secret"
	checkBundleConfigSubmissionURLAttr = "submission_url"
	checkBundleConfigURLAttr           = "url"

	// Attributes in circonus_check_bundle.metrics
	checkBundleMetricNameAttr   = "name"
	checkBundleMetricActiveAttr = "active"
	checkBundleMetricTypeAttr   = "type"
	checkBundleMetricUnitsAttr  = "units"
	checkBundleMetricTagsAttr   = "tags"
)

func resourceCheckBundle() *schema.Resource {
	return &schema.Resource{
		Create: checkBundleCreate,
		Read:   checkBundleRead,
		Update: checkBundleUpdate,
		Delete: checkBundleDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			checkBundleActiveAttr: &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
				Description: "If the check is activate or disabled",
			},
			checkBundleCheckUUIDsAttr: &schema.Schema{
				Type:        schema.TypeList,
				Computed:    true,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "",
			},
			checkBundleChecksAttr: &schema.Schema{
				Type:        schema.TypeList,
				Computed:    true,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "",
			},
			checkBundleBrokersAttr: &schema.Schema{
				Type:        schema.TypeList,
				Required:    true,
				MinItems:    1,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "",
			},
			checkBundleConfigAttr: &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						checkBundleConfigAsyncAttr: &schema.Schema{
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true, // verify
							Description: "",
						},
						checkBundleConfigSecretAttr: &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Description: "",
						},
						checkBundleConfigSubmissionURLAttr: &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Description: "",
						},
						checkBundleConfigHTTPVersionAttr: &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Description: "",
						},
						checkBundleConfigMethodAttr: &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Description: "",
						},
						checkBundleConfigPayloadAttr: &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Description: "",
						},
						checkBundleConfigPortAttr: &schema.Schema{
							Type:        schema.TypeString, // why isn't this an Int?
							Optional:    true,
							Description: "",
						},
						checkBundleConfigReadLimitAttr: &schema.Schema{
							Type:        schema.TypeString, // why isn't this an Int?
							Optional:    true,
							Description: "",
						},
						checkBundleConfigURLAttr: &schema.Schema{
							Type:        schema.TypeString, // why isn't this an Int?
							Optional:    true,
							Description: "",
						},
					},
				},
			},
			checkBundleMetricsAttr: &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						checkBundleMetricNameAttr: &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Name of the metric",
						},
						checkBundleMetricTypeAttr: &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							Description:  "Name of the metric",
							ValidateFunc: validateCheckBundleMetricType,
						},
						checkBundleMetricUnitsAttr: &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							Description: "Units for the metric",
						},
						checkBundleMetricActiveAttr: &schema.Schema{
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
							Description: "True if metric is active and collecting data",
						},
						checkBundleMetricTagsAttr: &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Schema{
								Type:         schema.TypeString,
								ValidateFunc: validateTag,
							},
							Description: "Tags assigned to a metric",
						},
					},
				},
			},
			checkBundleNameAttr: &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The name of the check bundle that will be displayed in the web interface",
			},
			checkBundleNotesAttr: &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Notes about this check bundle",
			},
			checkBundlePeriodAttr: &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: "The period between each time the check is made",
			},
			checkBundleTagsAttr: &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validateTag,
				},
				Description: "An array of tags",
			},
			checkBundleTargetAttr: &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The target of the check (e.g. hostname, URL, IP, etc)",
			},
			checkBundleTimeoutAttr: &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: "The length of time in seconds before the check will timeout if no response is returned to the broker",
			},
			checkBundleTypeAttr: &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The check type",
			},
		},
	}
}

func validateCheckBundleMetricType(v interface{}, key string) (warnings []string, errors []error) {
	value := v.(string)
	switch value {
	case "caql", "composite", "histogram", "numeric", "text":
	default:
		errors = append(errors, fmt.Errorf("unsupported metric type %s", value))
	}

	return warnings, errors
}

func validateTag(v interface{}, key string) (warnings []string, errors []error) {
	tag := v.(string)
	if !strings.ContainsRune(tag, ':') {
		errors = append(errors, fmt.Errorf("tag %q is missing a category", tag))
	}

	return warnings, errors
}

func validateTags(v interface{}) error {
	for i, tagRaw := range v.([]interface{}) {
		tag := tagRaw.(string)
		if !strings.ContainsRune(tag, ':') {
			return fmt.Errorf("tag %q at position %d in tag list is missing a category", tag, i+1)
		}
	}

	return nil
}

// timeout: 0-300s

// Valid types: 'caql', 'cim', 'circonuswindowsagent', 'circonuswindowsagent:nad', 'collectd', 'composite', 'dcm', 'dhcp', 'dns', 'elasticsearch', 'external', 'ganglia', 'googleanalytics', 'haproxy', 'http', 'http:apache', 'httptrap', 'imap', 'jmx', 'json', 'json:couchdb', 'json:mongodb', 'json:nad', 'json:riak', 'ldap', 'memcached', 'munin', 'mysql', 'newrelic_rpm', 'nginx', 'nrpe', 'ntp', 'oracle', 'ping_icmp', 'pop3', 'postgres', 'redis', 'resmon', 'smtp', 'snmp', 'snmp:momentum', 'sqlserver', 'ssh2', 'statsd', 'tcp', 'varnish', 'keynote', 'keynote_pulse', 'cloudwatch', 'ec_console', or 'mongodb'

func checkBundleCreate(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*api.API)

	name := d.Get(checkBundleNameAttr).(string)

	status := d.Get(checkBundleActiveAttr).(bool)
	statusString := "active"
	if !status {
		statusString = "disabled"
	}

	var brokers []string
	if brokersRaw, ok := d.GetOk(checkBundleBrokersAttr); ok {
		brokersList := brokersRaw.([]interface{})
		brokers = make([]string, 0, len(brokersList))
		for _, brokerRaw := range brokersList {
			brokers = append(brokers, brokerRaw.(string))
		}
	}

	var config api.CheckBundleConfig
	if bundleConfigRaw, ok := d.GetOk(checkBundleConfigAttr); ok {
		q.Q(bundleConfigRaw)
		bundleConfigList := bundleConfigRaw.(*schema.Set).List()
		for _, configRaw := range bundleConfigList {
			configMap := configRaw.(map[string]interface{})
			if v, ok := configMap[checkBundleConfigURLAttr]; ok {
				config.URL = v.(string)
			}
		}
	}

	var bundleMetrics []api.CheckBundleMetric
	if bundleMetricsRaw, ok := d.GetOk(checkBundleMetricsAttr); ok {
		bundleMetricsList := bundleMetricsRaw.(*schema.Set).List()
		bundleMetrics = make([]api.CheckBundleMetric, 0, len(bundleMetricsList))

		for _, metricRaw := range bundleMetricsList {
			metricMap := metricRaw.(map[string]interface{})

			var metricName string
			if v, ok := metricMap[checkBundleMetricNameAttr]; ok {
				metricName = v.(string)
			}

			var metricStatus string = "active"
			if v, ok := metricMap[checkBundleMetricActiveAttr]; ok {
				if !v.(bool) {
					metricStatus = "available"
				}
			}

			var metricTags []string
			if tagsRaw, ok := metricMap[checkBundleMetricTagsAttr]; ok {
				if err := validateTags(tagsRaw); err != nil {
					return err
				}

				tagsList := tagsRaw.([]interface{})
				metricTags = make([]string, 0, len(tagsList))
				for _, tagRaw := range tagsList {
					metricTags = append(metricTags, tagRaw.(string))
				}
			}

			var metricType string
			if v, ok := metricMap[checkBundleMetricTypeAttr]; ok {
				metricType = v.(string)
			}

			var metricUnits string
			if v, ok := metricMap[checkBundleMetricUnitsAttr]; ok {
				metricUnits = v.(string)
			}

			bundleMetrics = append(bundleMetrics, api.CheckBundleMetric{
				Name:   metricName,
				Status: metricStatus,
				Tags:   metricTags,
				Type:   metricType,
				Units:  metricUnits,
			})
		}
	}

	var bundleTags []string
	if tagsRaw, ok := d.GetOk(checkBundleTagsAttr); ok {
		if err := validateTags(tagsRaw); err != nil {
			return err
		}

		tagsList := tagsRaw.([]interface{})
		bundleTags = make([]string, 0, len(tagsList))
		for _, tagRaw := range tagsList {
			bundleTags = append(bundleTags, tagRaw.(string))
		}
	}

	target := d.Get(checkBundleTargetAttr).(string)
	checkType := d.Get(checkBundleTypeAttr).(string)

	bundleConfig := &api.CheckBundle{
		Brokers:     brokers,
		Config:      config,
		DisplayName: name,
		Metrics:     bundleMetrics,
		Status:      statusString,
		Tags:        bundleTags,
		Target:      target,
		Type:        checkType,
	}
	q.Q(config, bundleConfig)

	checkBundle, err := c.CreateCheckBundle(bundleConfig)
	if err != nil {
		return err
	}

	d.SetId(checkBundle.CID)

	return checkBundleRead(d, meta)
}

func checkBundleRead(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*api.API)

	cb, err := c.FetchCheckBundleByCID(api.CIDType(d.Id()))
	if err != nil {
		return err
	}

	var active bool
	switch cb.Status {
	case "active":
		active = true
	case "disabled":
		active = false
	default:
		return fmt.Errorf("check_bundle status unsupported: %q", cb.Status)
	}

	// NOTE(sean@): todo
	config := map[string]interface{}{
		checkBundleConfigURLAttr: cb.Config.URL,
	}

	// NOTE(sean@): todo
	metrics := map[string]interface{}{} // NOTE(sean@): TODO

	// NOTE(sean@): todo
	tags := []interface{}{} // NOTE(sean@): TODO

	d.SetId(cb.CID)
	d.Set(checkBundleCheckUUIDsAttr, cb.CheckUUIDs)
	d.Set(checkBundleChecksAttr, cb.Checks)
	d.Set(checkBundleCreatedAttr, cb.Created)
	d.Set(checkBundleLastModifiedAttr, cb.LastModified)
	d.Set(checkBundleLastModifiedByAttr, cb.LastModifedBy)
	d.Set(checkBundleBrokersAttr, cb.Brokers)
	d.Set(checkBundleConfigAttr, []interface{}{config})
	d.Set(checkBundleMetricNameAttr, cb.DisplayName)
	// NOTE(sean@): fixme
	_ = metrics
	// d.Set(checkBundleMetricsAttr, []interface{}{metrics})
	d.Set(checkBundleNotesAttr, cb.Notes)
	d.Set(checkBundlePeriodAttr, cb.Period)
	d.Set(checkBundleActiveAttr, active)
	d.Set(checkBundleTagsAttr, []interface{}{tags})
	d.Set(checkBundleTargetAttr, cb.Target)
	d.Set(checkBundleTimeoutAttr, cb.Timeout)
	d.Set(checkBundleTypeAttr, cb.Type)

	return nil
}

func checkBundleUpdate(d *schema.ResourceData, meta interface{}) error {
	return errors.New("update check bundle not implemented")
}

func checkBundleDelete(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*api.API)

	if _, err := c.Delete(d.Id()); err != nil {
		fmt.Errorf("unable to delete check_bundle %q: %v", d.Id(), err)
	}

	d.SetId("")

	return nil
}
