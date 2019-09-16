package test

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			// Optional attribute to label a particular instance for a test
			// that has multiple instances of this provider, so that they
			// can be distinguished using the test_provider_label data source.
			"label": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"test_resource":                  testResource(),
			"test_resource_gh12183":          testResourceGH12183(),
			"test_resource_import_other":     testResourceImportOther(),
			"test_resource_import_removed":   testResourceImportRemoved(),
			"test_resource_with_custom_diff": testResourceCustomDiff(),
			"test_resource_timeout":          testResourceTimeout(),
			"test_resource_diff_suppress":    testResourceDiffSuppress(),
			"test_resource_force_new":        testResourceForceNew(),
			"test_resource_nested":           testResourceNested(),
			"test_resource_nested_set":       testResourceNestedSet(),
			"test_resource_state_func":       testResourceStateFunc(),
			"test_resource_deprecated":       testResourceDeprecated(),
			"test_resource_defaults":         testResourceDefaults(),
			"test_resource_list":             testResourceList(),
			"test_resource_list_set":         testResourceListSet(),
			"test_resource_map":              testResourceMap(),
			"test_resource_computed_set":     testResourceComputedSet(),
			"test_resource_config_mode":      testResourceConfigMode(),
			"test_resource_nested_id":        testResourceNestedId(),
			"test_undeleteable":              testResourceUndeleteable(),
			"test_resource_required_min":     testResourceRequiredMin(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"test_data_source":    testDataSource(),
			"test_provider_label": providerLabelDataSource(),
		},
		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	return d.Get("label"), nil
}
