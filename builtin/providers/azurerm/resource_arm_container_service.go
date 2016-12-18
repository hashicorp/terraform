package azurerm

import (
	"fmt"
	"log"

	"net/http"

	"time"

	"github.com/Azure/azure-sdk-for-go/arm/containerservice"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceArmContainerService() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmContainerServiceCreate,
		Read:   resourceArmContainerServiceRead,
		Update: resourceArmContainerServiceCreate,
		Delete: resourceArmContainerServiceDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"location": {
				Type:      schema.TypeString,
				Required:  true,
				ForceNew:  true,
				StateFunc: azureRMNormalizeLocation,
			},

			"resource_group_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"orchestration_platform": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateArmContainerServiceOrchestrationPlatform,
			},

			"master_profile": {
				Type:     schema.TypeSet,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"count": {
							Type:     schema.TypeInt,
							Required: true,
						},

						"dns_prefix": {
							Type:     schema.TypeString,
							Required: true,
						},

						"fqdn": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},

			"linux_profile": {
				Type:     schema.TypeSet,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"admin_username": {
							Type:     schema.TypeString,
							Required: true,
						},
						"ssh_keys": {
							Type:     schema.TypeList,
							Required: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"key_data": {
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
					},
				},
			},

			"agent_pool_profile": {
				Type:     schema.TypeSet,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},

						"count": {
							Type:     schema.TypeInt,
							Required: true,
						},

						"dns_prefix": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},

						"fqdn": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},

						"vm_size": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},

			"service_principal": {
				Type:     schema.TypeSet,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"client_id": {
							Type:     schema.TypeString,
							Required: true,
						},

						"client_secret": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},

			"diagnostics": {
				Type:     schema.TypeSet,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Required: true,
						},

						"storage_uri": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceArmContainerServiceCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	containerServiceClient := client.containerServicesClient

	log.Printf("[INFO] preparing arguments for Azure ARM Container Service creation.")

	resGroup := d.Get("resource_group_name").(string)
	name := d.Get("name").(string)
	location := d.Get("location").(string)

	orchestrationPlatform := d.Get("orchestration_platform").(string)

	masterProfile, err := expandAzureRmContainerServiceMasterProfile(d)
	if err != nil {
		return err
	}

	linuxProfile, err := expandAzureRmContainerServiceLinuxProfile(d)
	if err != nil {
		return err
	}

	agentProfiles, err := expandAzureRmContainerServiceAgentProfiles(d)
	if err != nil {
		return err
	}

	diagosticsProfile, err := expandAzureRmContainerServiceDiagnostics(d)
	if err != nil {
		return err
	}

	servicePrincipalProfile, err := expandAzureRmContainerServiceServicePrincipal(d)
	if err != nil {
		return err
	}

	tags := d.Get("tags").(map[string]interface{})

	parameters := containerservice.ContainerService{
		Name:     &name,
		Location: &location,
		Properties: &containerservice.Properties{
			MasterProfile: &masterProfile,
			LinuxProfile:  &linuxProfile,
			OrchestratorProfile: &containerservice.OrchestratorProfile{
				OrchestratorType: containerservice.OchestratorTypes(orchestrationPlatform),
			},
			AgentPoolProfiles:       &agentProfiles,
			DiagnosticsProfile:      &diagosticsProfile,
			ServicePrincipalProfile: &servicePrincipalProfile,
		},
		Tags: expandTags(tags),
	}

	_, err = containerServiceClient.CreateOrUpdate(resGroup, name, parameters, make(chan struct{}))
	if err != nil {
		return err
	}

	read, err := containerServiceClient.Get(resGroup, name)
	if err != nil {
		return err
	}

	if read.ID == nil {
		return fmt.Errorf("Cannot read Container Service %s (resource group %s) ID", name, resGroup)
	}

	log.Printf("[DEBUG] Waiting for Container Service (%s) to become available", d.Get("name"))
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"Updating", "Creating"},
		Target:     []string{"Succeeded"},
		Refresh:    containerServiceStateRefreshFunc(client, resGroup, name),
		Timeout:    30 * time.Minute,
		MinTimeout: 15 * time.Second,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf("Error waiting for Container Service (%s) to become available: %s", d.Get("name"), err)
	}

	d.SetId(*read.ID)

	return resourceArmContainerServiceRead(d, meta)
}

func resourceArmContainerServiceRead(d *schema.ResourceData, meta interface{}) error {
	containerServiceClient := meta.(*ArmClient).containerServicesClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["containerServices"]

	resp, err := containerServiceClient.Get(resGroup, name)
	if err != nil {
		return fmt.Errorf("Error making Read request on Azure Container Service %s: %s", name, err)
	}
	if resp.StatusCode == http.StatusNotFound {
		d.SetId("")
		return nil
	}

	d.Set("name", resp.Name)
	d.Set("location", azureRMNormalizeLocation(*resp.Location))
	d.Set("resource_group_name", resGroup)

	d.Set("orchestration_platform", string(resp.Properties.OrchestratorProfile.OrchestratorType))

	if resp.Properties.MasterProfile != nil {
		// TODO: flatten
		d.Set("master_count", resp.Properties.MasterProfile.Count)
		d.Set("master_dns_prefix", resp.Properties.MasterProfile.DNSPrefix)
		d.Set("master_fqdn", resp.Properties.MasterProfile.Fqdn)
	}

	if resp.Properties.LinuxProfile != nil {
		// TODO: flatten
		d.Set("linux_admin_username", resp.Properties.LinuxProfile.AdminUsername)
		d.Set("linux_admin_ssh_keys", parseAzureRmContainerServiceSSHKeys(resp.Properties.LinuxProfile.SSH.PublicKeys))
	}

	if resp.Properties.AgentPoolProfiles != nil {
		profiles := parseAzureRmContainerServiceAgentProfiles(resp.Properties.AgentPoolProfiles)
		d.Set("agent_pool_profile", profiles)
	}

	if resp.Properties.DiagnosticsProfile != nil && resp.Properties.DiagnosticsProfile.VMDiagnostics != nil {
		d.Set("diagnostics_enabled", resp.Properties.DiagnosticsProfile.VMDiagnostics.Enabled)
		d.Set("diagnostics_storage_uri", resp.Properties.DiagnosticsProfile.VMDiagnostics.StorageURI)
	}

	if resp.Properties.ServicePrincipalProfile != nil {
		d.Set("client_id", resp.Properties.ServicePrincipalProfile.ClientID)
		d.Set("client_secret", resp.Properties.ServicePrincipalProfile.Secret)
	}

	flattenAndSetTags(d, resp.Tags)

	return nil
}

func resourceArmContainerServiceDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	containerServiceClient := client.containerServicesClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["containerServices"]

	resp, err := containerServiceClient.Delete(resGroup, name, make(chan struct{}))

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Error issuing Azure ARM delete request of Container Service '%s': %s", name, err)
	}

	return nil
}

func parseAzureRmContainerServiceSSHKeys(keys *[]containerservice.SSHPublicKey) []*string {

	sshKeys := make([]*string, 0, len(*keys))

	for _, key := range *keys {
		sshKeys = append(sshKeys, key.KeyData)
	}

	return sshKeys

}

func parseAzureRmContainerServiceAgentProfiles(pools *[]containerservice.AgentPoolProfile) *[]interface{} {
	profiles := make([]interface{}, len(*pools))

	for _, v := range *pools {
		profile := make(map[string]interface{}, 5)

		profile["name"] = v.Name
		profile["count"] = v.Count
		profile["dns_prefix"] = v.DNSPrefix
		profile["fqdn"] = v.Fqdn
		profile["vm_size"] = string(v.VMSize)

		profiles = append(profiles, profile)
	}

	return &profiles
}

func expandAzureRmContainerServiceDiagnostics(d *schema.ResourceData) (containerservice.DiagnosticsProfile, error) {
	configs := d.Get("diagnostics").(*schema.Set).List()
	profile := containerservice.DiagnosticsProfile{}

	for _, configRaw := range configs {
		data := configRaw.(map[string]interface{})

		enabled := data["enabled"].(bool)
		storage_uri := data["storage_uri"].(string)

		profile = containerservice.DiagnosticsProfile{
			VMDiagnostics: &containerservice.VMDiagnostics{
				Enabled: &enabled,
			},
		}

		if storage_uri != "" {
			profile.VMDiagnostics.StorageURI = &storage_uri
		}
	}

	return profile, nil
}

func expandAzureRmContainerServiceLinuxProfile(d *schema.ResourceData) (containerservice.LinuxProfile, error) {
	profiles := d.Get("linux_profile").(*schema.Set).List()
	config := profiles[0].(map[string]interface{})

	adminUsername := config["admin_username"].(string)
	linuxKeys := config["ssh_keys"].([]interface{})
	sshPublicKeys := []containerservice.SSHPublicKey{}

	for _, key := range linuxKeys {

		sshKey, ok := key.(map[string]interface{})
		if !ok {
			continue
		}
		keyData := sshKey["key_data"].(string)

		sshPublicKey := containerservice.SSHPublicKey{
			KeyData: &keyData,
		}

		sshPublicKeys = append(sshPublicKeys, sshPublicKey)
	}

	profile := containerservice.LinuxProfile{
		AdminUsername: &adminUsername,
		SSH: &containerservice.SSHConfiguration{
			PublicKeys: &sshPublicKeys,
		},
	}

	return profile, nil
}

func expandAzureRmContainerServiceMasterProfile(d *schema.ResourceData) (containerservice.MasterProfile, error) {
	configs := d.Get("master_profile").(*schema.Set).List()
	profile := containerservice.MasterProfile{}

	for _, configRaw := range configs {
		data := configRaw.(map[string]interface{})

		count := int32(data["count"].(int))
		dnsPrefix := data["dns_prefix"].(string)

		profile = containerservice.MasterProfile{
			Count:     &count,
			DNSPrefix: &dnsPrefix,
		}
	}

	return profile, nil
}

func expandAzureRmContainerServiceServicePrincipal(d *schema.ResourceData) (containerservice.ServicePrincipalProfile, error) {
	configs := d.Get("service_principal").(*schema.Set).List()
	config := configs[0].(map[string]interface{})

	clientId := config["client_id"].(string)
	clientSecret := config["client_secret"].(string)

	principal := containerservice.ServicePrincipalProfile{
		ClientID: &clientId,
		Secret:   &clientSecret,
	}

	return principal, nil
}

func expandAzureRmContainerServiceAgentProfiles(d *schema.ResourceData) ([]containerservice.AgentPoolProfile, error) {
	configs := d.Get("agent_pool_profile").(*schema.Set).List()
	profiles := make([]containerservice.AgentPoolProfile, 0, len(configs))

	for _, configRaw := range configs {
		data := configRaw.(map[string]interface{})

		name := data["name"].(string)
		count := int32(data["count"].(int))
		dnsPrefix := data["dns_prefix"].(string)
		fqdn := data["fqdn"].(string)
		vmSize := data["vm_size"].(string)

		profile := containerservice.AgentPoolProfile{
			Name:      &name,
			Count:     &count,
			VMSize:    containerservice.VMSizeTypes(vmSize),
			DNSPrefix: &dnsPrefix,
			Fqdn:      &fqdn,
		}

		profiles = append(profiles, profile)
	}

	return profiles, nil
}

func containerServiceStateRefreshFunc(client *ArmClient, resourceGroupName string, containerServiceName string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		res, err := client.containerServicesClient.Get(resourceGroupName, containerServiceName)
		if err != nil {
			return nil, "", fmt.Errorf("Error issuing read request in containerServiceStateRefreshFunc to Azure ARM for Container Service '%s' (RG: '%s'): %s", containerServiceName, resourceGroupName, err)
		}

		return res, *res.Properties.ProvisioningState, nil
	}
}

func validateArmContainerServiceOrchestrationPlatform(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	capacities := map[string]bool{
		"DCOS":       true,
		"Kubernetes": true,
		"Swarm":      true,
	}

	if !capacities[value] {
		errors = append(errors, fmt.Errorf("Container Service: Orchestration Platgorm can only be DCOS / Kubernetes / Swarm"))
	}
	return
}
