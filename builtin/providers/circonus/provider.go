package circonus

import (
	"bytes"
	"fmt"

	"github.com/circonus-labs/circonus-gometrics/api"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

const (
	apiURLAttr                           = "api_url"
	autoTagAttr                          = "auto_tag"
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
	keyAttr                              = "key"
	maxSeverity                          = 5
	minSeverity                          = 1
)

// Constants that want to be a constant but can't in Go
var (
	validContactHTTPFormats = validStringValues{"json", "params"}
	validContactHTTPMethods = validStringValues{"GET", "POST"}
)

type CheckType string
type ContactMethods string

type providerContext struct {
	// Circonus API client
	client *api.API

	// autoTag, when true, automatically appends defaultCirconusTag
	autoTag bool

	// defaultTag make up the tag to be used when autoTag tags a tag.
	defaultTag typeTag
}

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			apiURLAttr: {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "https://api.circonus.com/v2",
				Description: providerDescription[apiURLAttr],
			},
			autoTagAttr: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: providerDescription[autoTagAttr],
			},
			keyAttr: {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("CIRCONUS_API_TOKEN", nil),
				Description: providerDescription[keyAttr],
			},
		},

		DataSourcesMap: map[string]*schema.Resource{
			"circonus_account":   dataSourceCirconusAccount(),
			"circonus_collector": dataSourceCirconusCollector(),
		},

		ResourcesMap: map[string]*schema.Resource{
			"circonus_check":         resourceCheckBundle(),
			"circonus_contact_group": resourceContactGroup(),
			"circonus_metric":        resourceMetric(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := &api.Config{
		URL:      d.Get(apiURLAttr).(string),
		TokenKey: d.Get(keyAttr).(string),
		TokenApp: tfAppName(),
	}

	client, err := api.NewAPI(config)
	if err != nil {
		return nil, errwrap.Wrapf("Error initializing Circonus: %s", err)
	}

	return &providerContext{
		client:  client,
		autoTag: d.Get(autoTagAttr).(bool),
		defaultTag: typeTag{
			Category: defaultCirconusTagCategory,
			Value:    defaultCirconusTagValue,
		},
	}, nil
}

func tfAppName() string {
	const VersionPrerelease = terraform.VersionPrerelease
	var versionString bytes.Buffer

	fmt.Fprintf(&versionString, "Terraform v%s", terraform.Version)
	if terraform.VersionPrerelease != "" {
		fmt.Fprintf(&versionString, "-%s", terraform.VersionPrerelease)
	}

	return versionString.String()
}
