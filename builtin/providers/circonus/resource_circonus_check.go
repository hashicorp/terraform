package circonus

import (
	"fmt"
	"strings"

	"github.com/circonus-labs/circonus-gometrics/api"
	"github.com/circonus-labs/circonus-gometrics/api/config"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
)

/*
 * Note to future readers: The `circonus_check` resource is actually a facade for
 * the check_bundle call.  check_bundle is an implementation detail that we mask
 * over and expose just a "check" even though the "check" is actually a
 * check_bundle.
 */

type CheckType string

const (
	// Attributes in circonus_check
	checkActiveAttr      = "active"
	checkBrokersAttr     = "brokers"
	checkConfigAttr      = "config"
	checkMetricLimitAttr = "metric_limit"
	checkMetricAttr      = "metric"
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

const (
	// CheckBundle.Metric.Status can be one of these values
	metricStatusActive    = "active"
	metricStatusAvailable = "available"
)

func makeCheckBundleConfig(checkType CheckType) api.CheckBundleConfig {
	if size, ok := defaultCheckTypeConfigSize[checkType]; ok {
		return make(api.CheckBundleConfig, size)
	}

	return make(api.CheckBundleConfig, defaultCheckTypeConfigSize[defaultCheckTypeName])
}

func resourceCheckBundle() *schema.Resource {
	return &schema.Resource{
		Create: checkBundleCreate,
		Read:   checkBundleRead,
		Update: checkBundleUpdate,
		Delete: checkBundleDelete,
		Exists: checkBundleExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			checkActiveAttr: &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
				Description: "If the check is activate or disabled",
			},
			checkBrokersAttr: &schema.Schema{
				Type:        schema.TypeList,
				Required:    true,
				MinItems:    1,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "The broker(s) that are responsible for gathering the metrics",
			},
			checkConfigAttr: &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						checkConfigAuthMethodAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							Computed:     true,
							Description:  "The HTTP Authentication method",
							ValidateFunc: validateAuthMethod,
						},
						checkConfigAuthPasswordAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							Description:  "The HTTP Authentication user password",
							ValidateFunc: validateAuthPassword,
						},
						checkConfigAuthUserAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							Description:  "The HTTP Authentication user name",
							ValidateFunc: validateAuthUser,
						},
						checkConfigCAChainAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							Description:  "A path to a file containing all the certificate authorities that should be loaded to validate the remote certificate (for SSL checks)",
							ValidateFunc: validateCAChain,
						},
						checkConfigCertificateFileAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							Description:  "A path to a file containing the client certificate that will be presented to the remote server (for SSL checks)",
							ValidateFunc: validateCertificateFile,
						},
						checkConfigCiphersAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							Computed:     true,
							Description:  "A list of ciphers to be used in the SSL protocol (for SSL checks)",
							ValidateFunc: validateCiphers,
						},
						checkConfigHTTPHeadersAttr: &schema.Schema{
							Type:         schema.TypeMap,
							Optional:     true,
							Description:  "Map of HTTP Headers to send along with HTTP Requests",
							ValidateFunc: validateHTTPHeaders,
						},
						checkConfigHTTPVersionAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							Computed:     true,
							Description:  "Sets the HTTP version for the check to use",
							ValidateFunc: validateHTTPVersion,
						},
						checkConfigKeyFileAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							Description:  "A path to a file containing key to be used in conjunction with the cilent certificate (for SSL checks)",
							ValidateFunc: validateKeyFile,
						},
						checkConfigMethodAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							Computed:     true,
							Description:  "The HTTP method to use",
							ValidateFunc: validateMethod,
						},
						checkConfigPayloadAttr: &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							Description: "The information transferred as the payload of an HTTP request",
						},
						checkConfigPortAttr: &schema.Schema{
							Type:        schema.TypeString, // NOTE(sean@): Why isn't this an Int on Circonus's side?  Are they doing an /etc/services lookup?  TODO: convert this to a TypeInt and force users in TF to do a map lookup.
							Optional:    true,
							Computed:    true,
							Description: "Specifies the port on which the management interface can be reached",
						},
						checkConfigReadLimitAttr: &schema.Schema{
							Type:         schema.TypeInt,
							Optional:     true,
							Computed:     true,
							Description:  "Sets an approximate limit on the data read (0 means no limit)",
							ValidateFunc: validateReadLimit,
						},
						checkConfigRedirectsAttr: &schema.Schema{
							Type:         schema.TypeInt,
							Optional:     true,
							Computed:     true,
							Description:  `The maximum number of Location header redirects to follow (0 means no limit)`,
							ValidateFunc: validateRedirectLimit,
						},
						checkConfigURLAttr: &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Description: "The URL including schema and hostname (as you would type into a browser's location bar)",
						},
					},
				},
			},
			checkMetricLimitAttr: &schema.Schema{
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				Description:  `Setting a metric_limit will enable all (-1), disable (0), or allow up to the specified limit of metrics for this check ("N+", where N is a positive integer)`,
				ValidateFunc: validateMetricLimit,
			},
			checkMetricAttr: &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						checkMetricActiveAttr: &schema.Schema{
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
							Description: "True if metric is active and collecting data",
						},
						checkMetricNameAttr: &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							Description: "Name of the metric",
						},
						checkMetricTagsAttr: &schema.Schema{
							Type:        schema.TypeList,
							Optional:    true,
							Computed:    true,
							Description: "Tags assigned to a metric",
							Elem: &schema.Schema{
								Type:         schema.TypeString,
								ValidateFunc: validateTag,
							},
						},
						checkMetricTypeAttr: &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							Description:  "Type of the metric",
							ValidateFunc: validateMetricType,
						},
						checkMetricUnitsAttr: &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							Description: "Units for the metric",
						},
					},
				},
			},
			checkNameAttr: &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The name of the check bundle that will be displayed in the web interface",
			},
			checkNotesAttr: &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Notes about this check bundle",
				StateFunc: func(v interface{}) string {
					return strings.TrimSpace(v.(string))
				},
			},
			checkPeriodAttr: &schema.Schema{
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				Description:  "The period between each time the check is made",
				ValidateFunc: validatePeriod,
			},
			checkTagsAttr: &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validateTag,
				},
				Description: "An array of tags",
			},
			checkTargetAttr: &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The target of the check (e.g. hostname, URL, IP, etc)",
			},
			checkTimeoutAttr: &schema.Schema{
				Type:         schema.TypeFloat,
				Optional:     true,
				Computed:     true,
				Description:  "The length of time in seconds (and fractions of a second) before the check will timeout if no response is returned to the broker",
				ValidateFunc: validateTimeout,
			},
			checkTypeAttr: &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The check type",
				ValidateFunc: validateCheckType,
			},
		},
	}
}

func checkBundleCreate(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*api.API)

	in, err := getCheckBundleInput(d, meta)
	if err != nil {
		return err
	}

	cb, err := c.CreateCheckBundle(in)
	if err != nil {
		return err
	}

	d.SetId(cb.CID)

	return checkBundleRead(d, meta)
}

func checkBundleExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	c := meta.(*api.API)

	cid := d.Id()
	cb, err := c.FetchCheckBundle(api.CIDType(&cid))
	if err != nil {
		return false, err
	}

	if cb.CID == "" {
		return false, nil
	}

	return true, nil
}

func checkBundleRead(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*api.API)

	cid := d.Id()
	cb, err := c.FetchCheckBundle(api.CIDType(&cid))
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
	metrics := map[string]interface{}{} // NOTE(sean@): TODO

	d.Set(checkBrokersAttr, cb.Brokers)
	d.Set(checkConfigAttr, []interface{}{checkConfig})
	d.Set(checkMetricNameAttr, cb.DisplayName)
	// NOTE(sean@): fixme
	_ = metrics
	// d.Set(checkBundleMetricsAttr, []interface{}{metrics})
	d.Set(checkNotesAttr, cb.Notes)
	d.Set(checkPeriodAttr, cb.Period)
	d.Set(checkActiveAttr, active)
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
	c := meta.(*api.API)

	in, err := getCheckBundleInput(d, meta)
	if err != nil {
		return err
	}

	in.CID = d.Id()

	if _, err := c.UpdateCheckBundle(in); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("unable to update check %q: {{err}}", d.Id()), err)
	}

	return checkBundleRead(d, meta)
}

func checkBundleDelete(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*api.API)

	if _, err := c.Delete(d.Id()); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("unable to delete check %q: {{err}}", d.Id()), err)
	}

	d.SetId("")

	return nil
}

func getCheckBundleInput(d *schema.ResourceData, meta interface{}) (*api.CheckBundle, error) {
	c := meta.(*api.API)

	cb := c.NewCheckBundle()

	if name, ok := d.GetOk(checkNameAttr); ok {
		cb.DisplayName = name.(string)
	}

	if status, ok := d.GetOk(checkActiveAttr); ok {
		statusString := checkStatusActive
		if !status.(bool) {
			statusString = checkStatusDisabled
		}

		cb.Status = statusString
	}

	if brokersRaw, ok := d.GetOk(checkBrokersAttr); ok {
		brokersList := brokersRaw.([]interface{})
		cb.Brokers = make([]string, 0, len(brokersList))
		for _, brokerRaw := range brokersList {
			cb.Brokers = append(cb.Brokers, brokerRaw.(string))
		}
	}

	var checkType CheckType
	if v, ok := d.GetOk(checkTypeAttr); ok {
		cb.Type = v.(string)
		checkType = CheckType(cb.Type)
	}

	if bundleConfigRaw, ok := d.GetOk(checkConfigAttr); ok {
		bundleConfigList := bundleConfigRaw.([]interface{})
		checkConfig := makeCheckBundleConfig(checkType)
		switch configLen := len(bundleConfigList); {
		case configLen > 1:
			return nil, fmt.Errorf("config doesn't match schema: count %d", configLen)
		case configLen == 1:
			configMap := bundleConfigList[0].(map[string]interface{})
			if v, ok := configMap[checkConfigHTTPHeadersAttr]; ok {
				headerMap := v.(map[string]interface{})
				for hK, hV := range headerMap {
					hKey := config.HeaderPrefix + config.Key(hK)
					checkConfig[hKey] = hV.(string)
				}
			}

			if v, ok := configMap[checkConfigHTTPVersionAttr]; ok {
				checkConfig[checkConfigHTTPVersionAttr] = v.(string)
			}

			if v, ok := configMap[checkConfigPortAttr]; ok {
				checkConfig[checkConfigPortAttr] = v.(string)
			}

			if v, ok := configMap[checkConfigReadLimitAttr]; ok {
				checkConfig[checkConfigReadLimitAttr] = fmt.Sprintf("%d", v.(int))
			}

			if v, ok := configMap[checkConfigRedirectsAttr]; ok {
				checkConfig[checkConfigRedirectsAttr] = fmt.Sprintf("%d", v.(int))
			}

			if v, ok := configMap[checkConfigURLAttr]; ok {
				checkConfig[checkConfigURLAttr] = v.(string)
			}
		case configLen == 0:
			// Default config values, if any
		}

		cb.Config = checkConfig
	}

	if v, ok := d.GetOk(checkMetricLimitAttr); ok {
		cb.MetricLimit = v.(int)
	}

	if bundleMetricsRaw, ok := d.GetOk(checkMetricAttr); ok {
		bundleMetricsList := bundleMetricsRaw.([]interface{})
		cb.Metrics = make([]api.CheckBundleMetric, 0, len(bundleMetricsList))

		if len(bundleMetricsList) == 0 {
			return nil, fmt.Errorf("at least one metric must be specified per check bundle")
		}

		for _, metricRaw := range bundleMetricsList {
			metricMap := metricRaw.(map[string]interface{})

			var metricName string
			if v, ok := metricMap[checkMetricNameAttr]; ok {
				metricName = v.(string)
			}

			var metricStatus string = metricStatusActive
			if v, ok := metricMap[checkMetricActiveAttr]; ok {
				if !v.(bool) {
					metricStatus = metricStatusAvailable
				}
			}

			var metricTags []string
			if tagsRaw, ok := metricMap[checkMetricTagsAttr]; ok {
				if err := validateTags(tagsRaw); err != nil {
					return nil, err
				}

				tagsList := tagsRaw.([]interface{})
				metricTags = make([]string, 0, len(tagsList))
				for _, tagRaw := range tagsList {
					metricTags = append(metricTags, tagRaw.(string))
				}
			}

			var metricType string
			if v, ok := metricMap[checkMetricTypeAttr]; ok {
				metricType = v.(string)
			}

			var metricUnits string
			if v, ok := metricMap[checkMetricUnitsAttr]; ok {
				metricUnits = v.(string)
			}

			cb.Metrics = append(cb.Metrics, api.CheckBundleMetric{
				Name:   metricName,
				Status: metricStatus,
				Tags:   metricTags,
				Type:   metricType,
				Units:  metricUnits,
			})
		}
	}

	if v, ok := d.GetOk(checkNotesAttr); ok {
		cb.Notes = v.(string)
	}

	if v, ok := d.GetOk(checkPeriodAttr); ok {
		cb.Period = v.(int)
	}

	if tagsRaw, ok := d.GetOk(checkTagsAttr); ok {
		if err := validateTags(tagsRaw); err != nil {
			return nil, err
		}

		tagsList := tagsRaw.([]interface{})
		cb.Tags = make([]string, 0, len(tagsList))
		for _, tagRaw := range tagsList {
			cb.Tags = append(cb.Tags, tagRaw.(string))
		}
	}

	if v, ok := d.GetOk(checkTargetAttr); ok {
		cb.Target = v.(string)
	}

	if v, ok := d.GetOk(checkTimeoutAttr); ok {
		cb.Timeout = v.(float64)
	}

	if err := validateCheck(cb); err != nil {
		return nil, err
	}

	return cb, nil
}
