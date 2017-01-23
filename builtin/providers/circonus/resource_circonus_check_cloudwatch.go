package circonus

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/circonus-labs/circonus-gometrics/api/config"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/y0ssar1an/q"
)

const (
	// circonus_check.cloudwatch.* resource attribute names
	_CheckCloudWatchAPIKeyAttr      _SchemaAttr = "api_key"
	_CheckCloudWatchAPISecretAttr   _SchemaAttr = "api_secret"
	_CheckCloudWatchDimmensionsAttr _SchemaAttr = "dimmensions"
	_CheckCloudWatchMetricAttr      _SchemaAttr = "metric"
	_CheckCloudWatchNamespaceAttr   _SchemaAttr = "namespace"
	_CheckCloudWatchURLAttr         _SchemaAttr = "url"
	_CheckCloudWatchVersionAttr     _SchemaAttr = "version"
)

var _CheckCloudWatchDescriptions = _AttrDescrs{
	_CheckCloudWatchAPIKeyAttr:      "The AWS API Key",
	_CheckCloudWatchAPISecretAttr:   "The AWS API Secret",
	_CheckCloudWatchDimmensionsAttr: "The dimensions to query for the metric",
	_CheckCloudWatchMetricAttr:      "One or more CloudWatch Metric attributes",
	_CheckCloudWatchNamespaceAttr:   "The namespace to pull telemetry from",
	_CheckCloudWatchURLAttr:         "The URL including schema and hostname for the Cloudwatch monitoring server. This value will be used to specify the region - for example, to pull from us-east-1, the URL would be https://monitoring.us-east-1.amazonaws.com.",
	_CheckCloudWatchVersionAttr:     "The version of the Cloudwatch API to use.",
}

var _SchemaCheckCloudWatch = &schema.Schema{
	Type:     schema.TypeSet,
	Optional: true,
	MaxItems: 1,
	MinItems: 1,
	Set:      hashCheckCloudWatch,
	Elem: &schema.Resource{
		Schema: _CastSchemaToTF(map[_SchemaAttr]*schema.Schema{
			_CheckCloudWatchAPIKeyAttr: &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				Sensitive:    true,
				ValidateFunc: _ValidateRegexp(_CheckCloudWatchAPIKeyAttr, `[\S]+`),
				DefaultFunc:  schema.EnvDefaultFunc("AWS_ACCESS_KEY_ID", ""),
			},
			_CheckCloudWatchAPISecretAttr: &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				Sensitive:    true,
				ValidateFunc: _ValidateRegexp(_CheckCloudWatchAPISecretAttr, `[\S]+`),
				DefaultFunc:  schema.EnvDefaultFunc("AWS_SECRET_ACCESS_KEY", ""),
			},
			_CheckCloudWatchDimmensionsAttr: &schema.Schema{
				Type:         schema.TypeMap,
				Required:     true,
				Elem:         schema.TypeString,
				ValidateFunc: _ValidateCheckCloudWatchDimmensions,
			},
			_CheckCloudWatchMetricAttr: &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				MinItems: 1,
				Set:      schema.HashString,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: _ValidateRegexp(_CheckCloudWatchMetricAttr, `^([\S]+)$`),
				},
			},
			_CheckCloudWatchNamespaceAttr: &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: _ValidateRegexp(_CheckCloudWatchNamespaceAttr, `.+`),
			},
			_CheckCloudWatchURLAttr: &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: _ValidateHTTPURL(_CheckCloudWatchURLAttr, _URLIsAbs),
			},
			_CheckCloudWatchVersionAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      defaultCheckCloudWatchVersion,
				ValidateFunc: _ValidateRegexp(_CheckCloudWatchVersionAttr, `^[\d]{4}-[\d]{2}-[\d]{2}$`),
			},
		}, _CheckCloudWatchDescriptions),
	},
}

// _CheckAPIToStateCloudWatch reads the Config data out of _Check.CheckBundle into the
// statefile.
func _CheckAPIToStateCloudWatch(c *_Check, d *schema.ResourceData) error {
	cloudwatchConfig := make(map[string]interface{}, len(c.Config))

	// swamp is a sanity check: it must be empty by the time this method returns
	swamp := make(map[config.Key]string, len(c.Config))
	for k, v := range c.Config {
		swamp[k] = v
	}

	saveStringConfigToState := func(apiKey config.Key, attrName _SchemaAttr) {
		if v, ok := c.Config[apiKey]; ok {
			cloudwatchConfig[string(attrName)] = v
		}

		delete(swamp, apiKey)
	}

	saveStringConfigToState(config.APIKey, _CheckCloudWatchAPIKeyAttr)
	saveStringConfigToState(config.APISecret, _CheckCloudWatchAPISecretAttr)

	dimmensions := make(map[string]interface{}, len(c.Config))
	dimmensionPrefixLen := len(config.DimPrefix)
	for k, v := range c.Config {
		if len(k) <= dimmensionPrefixLen {
			continue
		}

		if strings.Compare(string(k[:dimmensionPrefixLen]), string(config.DimPrefix)) == 0 {
			key := k[dimmensionPrefixLen:]
			dimmensions[string(key)] = v
			q.Q(dimmensions)
		}
		delete(swamp, k)
	}
	cloudwatchConfig[string(_CheckCloudWatchDimmensionsAttr)] = dimmensions

	metricSet := schema.NewSet(schema.HashString, nil)
	metricList := strings.Split(c.Config[config.CloudwatchMetrics], ",")
	for _, m := range metricList {
		metricSet.Add(m)
	}
	cloudwatchConfig[string(_CheckCloudWatchMetricAttr)] = metricSet

	saveStringConfigToState(config.Namespace, _CheckCloudWatchNamespaceAttr)
	saveStringConfigToState(config.URL, _CheckCloudWatchURLAttr)
	saveStringConfigToState(config.Version, _CheckCloudWatchVersionAttr)

	whitelistedConfigKeys := map[config.Key]struct{}{
		config.ReverseSecretKey: struct{}{},
		config.SubmissionURL:    struct{}{},
	}

	for k, _ := range swamp {
		if _, ok := whitelistedConfigKeys[k]; ok {
			delete(c.Config, k)
		}

		if _, ok := whitelistedConfigKeys[k]; !ok {
			panic(fmt.Sprintf("PROVIDER BUG: API Config not empty: %#v", swamp))
		}
	}

	q.Q(cloudwatchConfig)
	_StateSet(d, _CheckCloudWatchAttr, schema.NewSet(hashCheckCloudWatch, []interface{}{cloudwatchConfig}))

	return nil
}

// hashCheckCloudWatch creates a stable hash of the normalized values
func hashCheckCloudWatch(v interface{}) int {
	m := v.(map[string]interface{})
	b := &bytes.Buffer{}
	b.Grow(defaultHashBufSize)

	writeString := func(attrName _SchemaAttr) {
		if v, ok := m[string(attrName)]; ok && v.(string) != "" {
			fmt.Fprint(b, strings.TrimSpace(v.(string)))
		}
	}

	// Order writes to the buffer using lexically sorted list for easy visual
	// reconciliation with other lists.
	writeString(_CheckCloudWatchAPIKeyAttr)
	writeString(_CheckCloudWatchAPISecretAttr)

	if dimmensionsRaw, ok := m[string(_CheckCloudWatchDimmensionsAttr)]; ok {
		dimmensionMap := dimmensionsRaw.(map[string]interface{})
		dimmensions := make([]string, 0, len(dimmensionMap))
		for k, _ := range dimmensionMap {
			dimmensions = append(dimmensions, k)
		}

		sort.Strings(dimmensions)
		for i, _ := range dimmensions {
			fmt.Fprint(b, dimmensions[i])
		}
	}

	if metricsRaw, ok := m[string(_CheckCloudWatchMetricAttr)]; ok {
		metricListRaw := flattenSet(metricsRaw.(*schema.Set))
		for i, _ := range metricListRaw {
			if metricListRaw[i] == nil {
				continue
			}
			fmt.Fprint(b, *metricListRaw[i])
		}
	}

	writeString(_CheckCloudWatchNamespaceAttr)
	writeString(_CheckCloudWatchURLAttr)
	writeString(_CheckCloudWatchVersionAttr)

	s := b.String()
	return hashcode.String(s)
}

func _CheckConfigToAPICloudWatch(c *_Check, ctxt *_ProviderContext, l _InterfaceList) error {
	c.Type = string(_APICheckTypeCloudWatchAttr)

	// Iterate over all `cloudwatch` attributes, even though we have a max of 1 in the
	// schema.
	for _, mapRaw := range l {
		cloudwatchConfig := _NewInterfaceMap(mapRaw)
		ar := _NewMapReader(ctxt, cloudwatchConfig)

		if s, ok := ar.GetStringOK(_CheckCloudWatchAPIKeyAttr); ok {
			c.Config[config.APIKey] = s
		}

		if s, ok := ar.GetStringOK(_CheckCloudWatchAPISecretAttr); ok {
			c.Config[config.APISecret] = s
		}

		if dimmensions := cloudwatchConfig.CollectMap(_CheckCloudWatchDimmensionsAttr); dimmensions != nil {
			for k, v := range dimmensions {
				dimKey := config.DimPrefix + config.Key(k)
				c.Config[dimKey] = v
			}
		}

		if metricsSet, ok := ar.GetSetAsListOK(_CheckCloudWatchMetricAttr); ok {
			metrics := metricsSet.List()
			sort.Strings(metrics)
			c.Config[config.CloudwatchMetrics] = strings.Join(metrics, ",")
		}

		if s, ok := ar.GetStringOK(_CheckCloudWatchNamespaceAttr); ok {
			c.Config[config.Namespace] = s
		}

		if s, ok := ar.GetStringOK(_CheckCloudWatchURLAttr); ok {
			c.Config[config.URL] = s
		}

		if s, ok := ar.GetStringOK(_CheckCloudWatchVersionAttr); ok {
			c.Config[config.Version] = s
		}
	}

	return nil
}
