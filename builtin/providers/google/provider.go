package google

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"account_file": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"account_file_contents"},
				DefaultFunc:   schema.EnvDefaultFunc("GOOGLE_ACCOUNT_FILE", nil),
			},

			"account_file_contents": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"account_file"},
				DefaultFunc:   schema.EnvDefaultFunc("GOOGLE_ACCOUNT_FILE_CONTENTS", nil),
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
			"google_compute_autoscaler": resourceComputeAutoscaler(),
			"google_compute_address": resourceComputeAddress(),
			"google_compute_disk": resourceComputeDisk(),
			"google_compute_firewall": resourceComputeFirewall(),
			"google_compute_forwarding_rule": resourceComputeForwardingRule(),
			"google_compute_http_health_check": resourceComputeHttpHealthCheck(),
			"google_compute_instance": resourceComputeInstance(),
			"google_compute_instance_template": resourceComputeInstanceTemplate(),
			"google_compute_network": resourceComputeNetwork(),
			"google_compute_route": resourceComputeRoute(),
			"google_compute_target_pool": resourceComputeTargetPool(),
			"google_container_cluster": resourceContainerCluster(),
			"google_dns_managed_zone": resourceDnsManagedZone(),
			"google_dns_record_set": resourceDnsRecordSet(),
			"google_compute_instance_group_manager": resourceComputeInstanceGroupManager(),
			"google_storage_bucket": resourceStorageBucket(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		AccountFile:         d.Get("account_file").(string),
		AccountFileContents: d.Get("account_file_contents").(string),
		Project:             d.Get("project").(string),
		Region:              d.Get("region").(string),
	}

	if err := config.loadAndValidate(); err != nil {
		return nil, err
	}

	return &config, nil
}
