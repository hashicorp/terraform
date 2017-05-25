package google

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/mutexkv"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

// Global MutexKV
var mutexKV = mutexkv.NewMutexKV()

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"credentials": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{
					"GOOGLE_CREDENTIALS",
					"GOOGLE_CLOUD_KEYFILE_JSON",
					"GCLOUD_KEYFILE_JSON",
				}, nil),
				ValidateFunc: validateCredentials,
			},

			"project": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{
					"GOOGLE_PROJECT",
					"GCLOUD_PROJECT",
					"CLOUDSDK_CORE_PROJECT",
				}, nil),
			},

			"region": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{
					"GOOGLE_REGION",
					"GCLOUD_REGION",
					"CLOUDSDK_COMPUTE_REGION",
				}, nil),
			},
		},

		DataSourcesMap: map[string]*schema.Resource{
			"google_compute_network":           dataSourceGoogleComputeNetwork(),
			"google_compute_subnetwork":        dataSourceGoogleComputeSubnetwork(),
			"google_compute_zones":             dataSourceGoogleComputeZones(),
			"google_container_engine_versions": dataSourceGoogleContainerEngineVersions(),
			"google_iam_policy":                dataSourceGoogleIamPolicy(),
			"google_storage_object_signed_url": dataSourceGoogleSignedUrl(),
		},

		ResourcesMap: map[string]*schema.Resource{
			"google_bigquery_dataset":               resourceBigQueryDataset(),
			"google_bigquery_table":                 resourceBigQueryTable(),
			"google_compute_autoscaler":             resourceComputeAutoscaler(),
			"google_compute_address":                resourceComputeAddress(),
			"google_compute_backend_bucket":         resourceComputeBackendBucket(),
			"google_compute_backend_service":        resourceComputeBackendService(),
			"google_compute_disk":                   resourceComputeDisk(),
			"google_compute_snapshot":               resourceComputeSnapshot(),
			"google_compute_firewall":               resourceComputeFirewall(),
			"google_compute_forwarding_rule":        resourceComputeForwardingRule(),
			"google_compute_global_address":         resourceComputeGlobalAddress(),
			"google_compute_global_forwarding_rule": resourceComputeGlobalForwardingRule(),
			"google_compute_health_check":           resourceComputeHealthCheck(),
			"google_compute_http_health_check":      resourceComputeHttpHealthCheck(),
			"google_compute_https_health_check":     resourceComputeHttpsHealthCheck(),
			"google_compute_image":                  resourceComputeImage(),
			"google_compute_instance":               resourceComputeInstance(),
			"google_compute_instance_group":         resourceComputeInstanceGroup(),
			"google_compute_instance_group_manager": resourceComputeInstanceGroupManager(),
			"google_compute_instance_template":      resourceComputeInstanceTemplate(),
			"google_compute_network":                resourceComputeNetwork(),
			"google_compute_project_metadata":       resourceComputeProjectMetadata(),
			"google_compute_region_backend_service": resourceComputeRegionBackendService(),
			"google_compute_route":                  resourceComputeRoute(),
			"google_compute_router":                 resourceComputeRouter(),
			"google_compute_router_interface":       resourceComputeRouterInterface(),
			"google_compute_router_peer":            resourceComputeRouterPeer(),
			"google_compute_ssl_certificate":        resourceComputeSslCertificate(),
			"google_compute_subnetwork":             resourceComputeSubnetwork(),
			"google_compute_target_http_proxy":      resourceComputeTargetHttpProxy(),
			"google_compute_target_https_proxy":     resourceComputeTargetHttpsProxy(),
			"google_compute_target_pool":            resourceComputeTargetPool(),
			"google_compute_url_map":                resourceComputeUrlMap(),
			"google_compute_vpn_gateway":            resourceComputeVpnGateway(),
			"google_compute_vpn_tunnel":             resourceComputeVpnTunnel(),
			"google_container_cluster":              resourceContainerCluster(),
			"google_container_node_pool":            resourceContainerNodePool(),
			"google_dns_managed_zone":               resourceDnsManagedZone(),
			"google_dns_record_set":                 resourceDnsRecordSet(),
			"google_sql_database":                   resourceSqlDatabase(),
			"google_sql_database_instance":          resourceSqlDatabaseInstance(),
			"google_sql_user":                       resourceSqlUser(),
			"google_project":                        resourceGoogleProject(),
			"google_project_iam_policy":             resourceGoogleProjectIamPolicy(),
			"google_project_services":               resourceGoogleProjectServices(),
			"google_pubsub_topic":                   resourcePubsubTopic(),
			"google_pubsub_subscription":            resourcePubsubSubscription(),
			"google_service_account":                resourceGoogleServiceAccount(),
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

// getRegionFromZone returns the region from a zone for Google cloud.
func getRegionFromZone(zone string) string {
	if zone != "" && len(zone) > 2 {
		region := zone[:len(zone)-2]
		return region
	}
	return ""
}

// getRegion reads the "region" field from the given resource data and falls
// back to the provider's value if not given. If the provider's value is not
// given, an error is returned.
func getRegion(d *schema.ResourceData, config *Config) (string, error) {
	res, ok := d.GetOk("region")
	if !ok {
		if config.Region != "" {
			return config.Region, nil
		}
		return "", fmt.Errorf("%q: required field is not set", "region")
	}
	return res.(string), nil
}

// getProject reads the "project" field from the given resource data and falls
// back to the provider's value if not given. If the provider's value is not
// given, an error is returned.
func getProject(d *schema.ResourceData, config *Config) (string, error) {
	res, ok := d.GetOk("project")
	if !ok {
		if config.Project != "" {
			return config.Project, nil
		}
		return "", fmt.Errorf("%q: required field is not set", "project")
	}
	return res.(string), nil
}

func getZonalResourceFromRegion(getResource func(string) (interface{}, error), region string, compute *compute.Service, project string) (interface{}, error) {
	zoneList, err := compute.Zones.List(project).Do()
	if err != nil {
		return nil, err
	}
	var resource interface{}
	for _, zone := range zoneList.Items {
		if strings.Contains(zone.Name, region) {
			resource, err = getResource(zone.Name)
			if err != nil {
				if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
					// Resource was not found in this zone
					continue
				}
				return nil, fmt.Errorf("Error reading Resource: %s", err)
			}
			// Resource was found
			return resource, nil
		}
	}
	// Resource does not exist in this region
	return nil, nil
}

// getNetworkLink reads the "network" field from the given resource data and if the value:
// - is a resource URL, returns the string unchanged
// - is the network name only, then looks up the resource URL using the google client
func getNetworkLink(d *schema.ResourceData, config *Config, field string) (string, error) {
	if v, ok := d.GetOk(field); ok {
		network := v.(string)

		project, err := getProject(d, config)
		if err != nil {
			return "", err
		}

		if !strings.HasPrefix(network, "https://www.googleapis.com/compute/") {
			// Network value provided is just the name, lookup the network SelfLink
			networkData, err := config.clientCompute.Networks.Get(
				project, network).Do()
			if err != nil {
				return "", fmt.Errorf("Error reading network: %s", err)
			}
			network = networkData.SelfLink
		}

		return network, nil

	} else {
		return "", nil
	}
}

// getNetworkName reads the "network" field from the given resource data and if the value:
// - is a resource URL, extracts the network name from the URL and returns it
// - is the network name only (i.e not prefixed with http://www.googleapis.com/compute/...), is returned unchanged
func getNetworkName(d *schema.ResourceData, field string) (string, error) {
	if v, ok := d.GetOk(field); ok {
		network := v.(string)
		return getNetworkNameFromSelfLink(network)
	}
	return "", nil
}

func getNetworkNameFromSelfLink(network string) (string, error) {
	if strings.HasPrefix(network, "https://www.googleapis.com/compute/") {
		// extract the network name from SelfLink URL
		networkName := network[strings.LastIndex(network, "/")+1:]
		if networkName == "" {
			return "", fmt.Errorf("network url not valid")
		}
		return networkName, nil
	}

	return network, nil
}

func getRouterLockName(region string, router string) string {
	return fmt.Sprintf("router/%s/%s", region, router)
}

func handleNotFoundError(err error, d *schema.ResourceData, resource string) error {
	if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
		log.Printf("[WARN] Removing %s because it's gone", resource)
		// The resource doesn't exist anymore
		d.SetId("")

		return nil
	}

	return fmt.Errorf("Error reading %s: %s", resource, err)
}

func linkDiffSuppress(k, old, new string, d *schema.ResourceData) bool {
	parts := strings.Split(old, "/")
	if parts[len(parts)-1] == new {
		return true
	}
	return false
}
