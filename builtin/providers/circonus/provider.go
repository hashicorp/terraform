package circonus

import (
	"fmt"

	"github.com/circonus-labs/circonus-gometrics/api"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

const (
	defaultCirconus404ErrorString        = "API response code 404:"
	defaultCirconusAggregationWindow     = "300s"
	defaultCirconusAlertMinEscalateAfter = "300s"
	defaultCirconusCheckPeriodMax        = "300s"
	defaultCirconusCheckPeriodMin        = "30s"
	defaultCirconusHTTPFormat            = "json"
	defaultCirconusHTTPMethod            = "POST"
	defaultCirconusSlackUsername         = "Circonus"
	defaultCirconusTimeoutMax            = "300s"
	defaultCirconusTimeoutMin            = "0s"
	maxSeverity                          = 5
	minSeverity                          = 1
)

var providerDescription = map[string]string{
	providerAPIURLAttr:  "URL of the Circonus API",
	providerAutoTagAttr: "Signals that the provider should automatically add a tag to all API calls denoting that the resource was created by Terraform",
	providerKeyAttr:     "API token used to authenticate with the Circonus API",
}

// Constants that want to be a constant but can't in Go
var (
	validContactHTTPFormats = validStringValues{"json", "params"}
	validContactHTTPMethods = validStringValues{"GET", "POST"}
)

type contactMethods string

// globalAutoTag controls whether or not the provider should automatically add a
// tag to each resource.
//
// NOTE(sean): This is done as a global variable because the diff suppress
// functions does not have access to the providerContext, only the key, old, and
// new values.
var globalAutoTag bool

type providerContext struct {
	// Circonus API client
	client *api.API

	// autoTag, when true, automatically appends defaultCirconusTag
	autoTag bool

	// defaultTag make up the tag to be used when autoTag tags a tag.
	defaultTag circonusTag
}

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			providerAPIURLAttr: {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "https://api.circonus.com/v2",
				Description: providerDescription[providerAPIURLAttr],
			},
			providerAutoTagAttr: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     defaultAutoTag,
				Description: providerDescription[providerAutoTagAttr],
			},
			providerKeyAttr: {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("CIRCONUS_API_TOKEN", nil),
				Description: providerDescription[providerKeyAttr],
			},
		},

		DataSourcesMap: map[string]*schema.Resource{
			"circonus_account":   dataSourceCirconusAccount(),
			"circonus_collector": dataSourceCirconusCollector(),
		},

		ResourcesMap: map[string]*schema.Resource{
			"circonus_check":          resourceCheck(),
			"circonus_contact_group":  resourceContactGroup(),
			"circonus_graph":          resourceGraph(),
			"circonus_metric":         resourceMetric(),
			"circonus_metric_cluster": resourceMetricCluster(),
			"circonus_rule_set":       resourceRuleSet(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	globalAutoTag = d.Get(providerAutoTagAttr).(bool)

	config := &api.Config{
		URL:      d.Get(providerAPIURLAttr).(string),
		TokenKey: d.Get(providerKeyAttr).(string),
		TokenApp: tfAppName(),
	}

	client, err := api.NewAPI(config)
	if err != nil {
		return nil, errwrap.Wrapf("Error initializing Circonus: %s", err)
	}

	return &providerContext{
		client:     client,
		autoTag:    d.Get(providerAutoTagAttr).(bool),
		defaultTag: defaultCirconusTag,
	}, nil
}

func tfAppName() string {
	return fmt.Sprintf("Terraform v%s", terraform.VersionString())
}
