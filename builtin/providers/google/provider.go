package google

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"account_file": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				DefaultFunc:  schema.EnvDefaultFunc("GOOGLE_ACCOUNT_FILE", nil),
				ValidateFunc: validateAccountFile,
			},

			"project": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("GOOGLE_PROJECT", nil),
			},

			"region": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("GOOGLE_REGION", nil),
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"google_compute_autoscaler":             resourceComputeAutoscaler(),
			"google_compute_address":                resourceComputeAddress(),
			"google_compute_backend_service":        resourceComputeBackendService(),
			"google_compute_disk":                   resourceComputeDisk(),
			"google_compute_firewall":               resourceComputeFirewall(),
			"google_compute_forwarding_rule":        resourceComputeForwardingRule(),
			"google_compute_http_health_check":      resourceComputeHttpHealthCheck(),
			"google_compute_instance":               resourceComputeInstance(),
			"google_compute_instance_template":      resourceComputeInstanceTemplate(),
			"google_compute_network":                resourceComputeNetwork(),
			"google_compute_project_metadata":       resourceComputeProjectMetadata(),
			"google_compute_route":                  resourceComputeRoute(),
			"google_compute_target_pool":            resourceComputeTargetPool(),
			"google_compute_vpn_gateway":            resourceComputeVpnGateway(),
			"google_compute_vpn_tunnel":             resourceComputeVpnTunnel(),
			"google_container_cluster":              resourceContainerCluster(),
			"google_dns_managed_zone":               resourceDnsManagedZone(),
			"google_dns_record_set":                 resourceDnsRecordSet(),
			"google_compute_instance_group_manager": resourceComputeInstanceGroupManager(),
			"google_storage_bucket":                 resourceStorageBucket(),
			"google_storage_bucket_acl":             resourceStorageBucketAcl(),
			"google_storage_bucket_object":          resourceStorageBucketObject(),
			"google_storage_object_acl":             resourceStorageObjectAcl(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		AccountFile: d.Get("account_file").(string),
		Project:     d.Get("project").(string),
		Region:      d.Get("region").(string),
	}

	if err := config.loadAndValidate(); err != nil {
		return nil, err
	}

	return &config, nil
}

func validateAccountFile(v interface{}, k string) (warnings []string, errors []error) {
	value := v.(string)

	if value == "" {
		return
	}

	var account accountFile
	if err := json.Unmarshal([]byte(value), &account); err != nil {
		warnings = append(warnings, `
account_file is not valid JSON, so we are assuming it is a file path. This
support will be removed in the future. Please update your configuration to use
${file("filename.json")} instead.`)
	} else {
		return
	}

	if _, err := os.Stat(value); err != nil {
		errors = append(errors,
			fmt.Errorf(
				"account_file path could not be read from '%s': %s",
				value,
				err))
	}

	return
}
