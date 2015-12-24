package google

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform/helper/pathorcontents"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"account_file": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				DefaultFunc:  schema.EnvDefaultFunc("GOOGLE_ACCOUNT_FILE", nil),
				ValidateFunc: validateAccountFile,
				Deprecated:   "Use the credentials field instead",
			},

			"credentials": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				DefaultFunc:  schema.EnvDefaultFunc("GOOGLE_CREDENTIALS", nil),
				ValidateFunc: validateCredentials,
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
			"google_compute_global_address":         resourceComputeGlobalAddress(),
			"google_compute_global_forwarding_rule": resourceComputeGlobalForwardingRule(),
			"google_compute_http_health_check":      resourceComputeHttpHealthCheck(),
			"google_compute_https_health_check":     resourceComputeHttpsHealthCheck(),
			"google_compute_instance":               resourceComputeInstance(),
			"google_compute_instance_group_manager": resourceComputeInstanceGroupManager(),
			"google_compute_instance_template":      resourceComputeInstanceTemplate(),
			"google_compute_network":                resourceComputeNetwork(),
			"google_compute_project_metadata":       resourceComputeProjectMetadata(),
			"google_compute_route":                  resourceComputeRoute(),
			"google_compute_ssl_certificate":        resourceComputeSslCertificate(),
			"google_compute_target_http_proxy":      resourceComputeTargetHttpProxy(),
			"google_compute_target_https_proxy":     resourceComputeTargetHttpsProxy(),
			"google_compute_target_pool":            resourceComputeTargetPool(),
			"google_compute_url_map":                resourceComputeUrlMap(),
			"google_compute_vpn_gateway":            resourceComputeVpnGateway(),
			"google_compute_vpn_tunnel":             resourceComputeVpnTunnel(),
			"google_container_cluster":              resourceContainerCluster(),
			"google_dns_managed_zone":               resourceDnsManagedZone(),
			"google_dns_record_set":                 resourceDnsRecordSet(),
			"google_sql_database":                   resourceSqlDatabase(),
			"google_sql_database_instance":          resourceSqlDatabaseInstance(),
			"google_pubsub_topic":                   resourcePubsubTopic(),
			"google_pubsub_subscription":            resourcePubsubSubscription(),
			"google_storage_bucket":                 resourceStorageBucket(),
			"google_storage_bucket_acl":             resourceStorageBucketAcl(),
			"google_storage_bucket_object":          resourceStorageBucketObject(),
			"google_storage_object_acl":             resourceStorageObjectAcl(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	credentials := d.Get("credentials").(string)
	if credentials == "" {
		credentials = d.Get("account_file").(string)
	}
	config := Config{
		Credentials: credentials,
		Project:     d.Get("project").(string),
		Region:      d.Get("region").(string),
	}

	if err := config.loadAndValidate(); err != nil {
		return nil, err
	}

	return &config, nil
}

func validateAccountFile(v interface{}, k string) (warnings []string, errors []error) {
	if v == nil {
		return
	}

	value := v.(string)

	if value == "" {
		return
	}

	contents, wasPath, err := pathorcontents.Read(value)
	if err != nil {
		errors = append(errors, fmt.Errorf("Error loading Account File: %s", err))
	}
	if wasPath {
		warnings = append(warnings, `account_file was provided as a path instead of 
as file contents. This support will be removed in the future. Please update
your configuration to use ${file("filename.json")} instead.`)
	}

	var account accountFile
	if err := json.Unmarshal([]byte(contents), &account); err != nil {
		errors = append(errors,
			fmt.Errorf("account_file not valid JSON '%s': %s", contents, err))
	}

	return
}

func validateCredentials(v interface{}, k string) (warnings []string, errors []error) {
	if v == nil || v.(string) == "" {
		return
	}
	creds := v.(string)
	var account accountFile
	if err := json.Unmarshal([]byte(creds), &account); err != nil {
		errors = append(errors,
			fmt.Errorf("credentials are not valid JSON '%s': %s", creds, err))
	}

	return
}
