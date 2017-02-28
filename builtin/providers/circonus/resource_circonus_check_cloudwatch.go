package circonus

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/circonus-labs/circonus-gometrics/api/config"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

const (
	// circonus_check.cloudwatch.* resource attribute names
	checkCloudWatchAPIKeyAttr      schemaAttr = "api_key"
	checkCloudWatchAPISecretAttr   schemaAttr = "api_secret"
	checkCloudWatchDimmensionsAttr schemaAttr = "dimmensions"
	checkCloudWatchMetricAttr      schemaAttr = "metric"
	checkCloudWatchNamespaceAttr   schemaAttr = "namespace"
	checkCloudWatchURLAttr         schemaAttr = "url"
	checkCloudWatchVersionAttr     schemaAttr = "version"
)

var checkCloudWatchDescriptions = attrDescrs{
	checkCloudWatchAPIKeyAttr:      "The AWS API Key",
	checkCloudWatchAPISecretAttr:   "The AWS API Secret",
	checkCloudWatchDimmensionsAttr: "The dimensions to query for the metric",
	checkCloudWatchMetricAttr:      "One or more CloudWatch Metric attributes",
	checkCloudWatchNamespaceAttr:   "The namespace to pull telemetry from",
	checkCloudWatchURLAttr:         "The URL including schema and hostname for the Cloudwatch monitoring server. This value will be used to specify the region - for example, to pull from us-east-1, the URL would be https://monitoring.us-east-1.amazonaws.com.",
	checkCloudWatchVersionAttr:     "The version of the Cloudwatch API to use.",
}

var schemaCheckCloudWatch = &schema.Schema{
	Type:     schema.TypeSet,
	Optional: true,
	MaxItems: 1,
	MinItems: 1,
	Set:      hashCheckCloudWatch,
	Elem: &schema.Resource{
		Schema: castSchemaToTF(map[schemaAttr]*schema.Schema{
			checkCloudWatchAPIKeyAttr: &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				Sensitive:    true,
				ValidateFunc: validateRegexp(checkCloudWatchAPIKeyAttr, `[\S]+`),
				DefaultFunc:  schema.EnvDefaultFunc("AWS_ACCESS_KEY_ID", ""),
			},
			checkCloudWatchAPISecretAttr: &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				Sensitive:    true,
				ValidateFunc: validateRegexp(checkCloudWatchAPISecretAttr, `[\S]+`),
				DefaultFunc:  schema.EnvDefaultFunc("AWS_SECRET_ACCESS_KEY", ""),
			},
			checkCloudWatchDimmensionsAttr: &schema.Schema{
				Type:         schema.TypeMap,
				Required:     true,
				Elem:         schema.TypeString,
				ValidateFunc: validateCheckCloudWatchDimmensions,
			},
			checkCloudWatchMetricAttr: &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				MinItems: 1,
				Set:      schema.HashString,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validateRegexp(checkCloudWatchMetricAttr, `^([\S]+)$`),
				},
			},
			checkCloudWatchNamespaceAttr: &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateRegexp(checkCloudWatchNamespaceAttr, `.+`),
			},
			checkCloudWatchURLAttr: &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateHTTPURL(checkCloudWatchURLAttr, urlIsAbs),
			},
			checkCloudWatchVersionAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      defaultCheckCloudWatchVersion,
				ValidateFunc: validateRegexp(checkCloudWatchVersionAttr, `^[\d]{4}-[\d]{2}-[\d]{2}$`),
			},
		}, checkCloudWatchDescriptions),
	},
}

// checkAPIToStateCloudWatch reads the Config data out of circonusCheck.CheckBundle into the
// statefile.
func checkAPIToStateCloudWatch(c *circonusCheck, d *schema.ResourceData) error {
	cloudwatchConfig := make(map[string]interface{}, len(c.Config))

	// swamp is a sanity check: it must be empty by the time this method returns
	swamp := make(map[config.Key]string, len(c.Config))
	for k, v := range c.Config {
		swamp[k] = v
	}

	saveStringConfigToState := func(apiKey config.Key, attrName schemaAttr) {
		if v, ok := c.Config[apiKey]; ok {
			cloudwatchConfig[string(attrName)] = v
		}

		delete(swamp, apiKey)
	}

	saveStringConfigToState(config.APIKey, checkCloudWatchAPIKeyAttr)
	saveStringConfigToState(config.APISecret, checkCloudWatchAPISecretAttr)

	dimmensions := make(map[string]interface{}, len(c.Config))
	dimmensionPrefixLen := len(config.DimPrefix)
	for k, v := range c.Config {
		if len(k) <= dimmensionPrefixLen {
			continue
		}

		if strings.Compare(string(k[:dimmensionPrefixLen]), string(config.DimPrefix)) == 0 {
			key := k[dimmensionPrefixLen:]
			dimmensions[string(key)] = v
		}
		delete(swamp, k)
	}
	cloudwatchConfig[string(checkCloudWatchDimmensionsAttr)] = dimmensions

	metricSet := schema.NewSet(schema.HashString, nil)
	metricList := strings.Split(c.Config[config.CloudwatchMetrics], ",")
	for _, m := range metricList {
		metricSet.Add(m)
	}
	cloudwatchConfig[string(checkCloudWatchMetricAttr)] = metricSet

	saveStringConfigToState(config.Namespace, checkCloudWatchNamespaceAttr)
	saveStringConfigToState(config.URL, checkCloudWatchURLAttr)
	saveStringConfigToState(config.Version, checkCloudWatchVersionAttr)

	whitelistedConfigKeys := map[config.Key]struct{}{
		config.ReverseSecretKey: struct{}{},
		config.SubmissionURL:    struct{}{},
	}

	for k := range swamp {
		if _, ok := whitelistedConfigKeys[k]; ok {
			delete(c.Config, k)
		}

		if _, ok := whitelistedConfigKeys[k]; !ok {
			panic(fmt.Sprintf("PROVIDER BUG: API Config not empty: %#v", swamp))
		}
	}

	if err := d.Set(checkCloudWatchAttr, schema.NewSet(hashCheckCloudWatch, []interface{}{cloudwatchConfig})); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Unable to store check %q attribute: {{err}}", checkCloudWatchAttr), err)
	}

	return nil
}

// hashCheckCloudWatch creates a stable hash of the normalized values
func hashCheckCloudWatch(v interface{}) int {
	m := v.(map[string]interface{})
	b := &bytes.Buffer{}
	b.Grow(defaultHashBufSize)

	writeString := func(attrName schemaAttr) {
		if v, ok := m[string(attrName)]; ok && v.(string) != "" {
			fmt.Fprint(b, strings.TrimSpace(v.(string)))
		}
	}

	// Order writes to the buffer using lexically sorted list for easy visual
	// reconciliation with other lists.
	writeString(checkCloudWatchAPIKeyAttr)
	writeString(checkCloudWatchAPISecretAttr)

	if dimmensionsRaw, ok := m[string(checkCloudWatchDimmensionsAttr)]; ok {
		dimmensionMap := dimmensionsRaw.(map[string]interface{})
		dimmensions := make([]string, 0, len(dimmensionMap))
		for k := range dimmensionMap {
			dimmensions = append(dimmensions, k)
		}

		sort.Strings(dimmensions)
		for i := range dimmensions {
			fmt.Fprint(b, dimmensions[i])
		}
	}

	if metricsRaw, ok := m[string(checkCloudWatchMetricAttr)]; ok {
		metricListRaw := flattenSet(metricsRaw.(*schema.Set))
		for i := range metricListRaw {
			if metricListRaw[i] == nil {
				continue
			}
			fmt.Fprint(b, *metricListRaw[i])
		}
	}

	writeString(checkCloudWatchNamespaceAttr)
	writeString(checkCloudWatchURLAttr)
	writeString(checkCloudWatchVersionAttr)

	s := b.String()
	return hashcode.String(s)
}

func checkConfigToAPICloudWatch(c *circonusCheck, ctxt *providerContext, l interfaceList) error {
	c.Type = string(apiCheckTypeCloudWatchAttr)

	// Iterate over all `cloudwatch` attributes, even though we have a max of 1 in the
	// schema.
	for _, mapRaw := range l {
		cloudwatchConfig := newInterfaceMap(mapRaw)
		ar := newMapReader(ctxt, cloudwatchConfig)

		if s, ok := ar.GetStringOK(checkCloudWatchAPIKeyAttr); ok {
			c.Config[config.APIKey] = s
		}

		if s, ok := ar.GetStringOK(checkCloudWatchAPISecretAttr); ok {
			c.Config[config.APISecret] = s
		}

		if dimmensions := cloudwatchConfig.CollectMap(checkCloudWatchDimmensionsAttr); dimmensions != nil {
			for k, v := range dimmensions {
				dimKey := config.DimPrefix + config.Key(k)
				c.Config[dimKey] = v
			}
		}

		if metricsSet, ok := ar.GetSetAsListOK(checkCloudWatchMetricAttr); ok {
			metrics := metricsSet.List()
			sort.Strings(metrics)
			c.Config[config.CloudwatchMetrics] = strings.Join(metrics, ",")
		}

		if s, ok := ar.GetStringOK(checkCloudWatchNamespaceAttr); ok {
			c.Config[config.Namespace] = s
		}

		if s, ok := ar.GetStringOK(checkCloudWatchURLAttr); ok {
			c.Config[config.URL] = s
		}

		if s, ok := ar.GetStringOK(checkCloudWatchVersionAttr); ok {
			c.Config[config.Version] = s
		}
	}

	return nil
}
