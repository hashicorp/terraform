package azurerm

import (
	"fmt"
	"log"

	"net/http"

	"time"

	"bytes"

	"github.com/Azure/azure-sdk-for-go/arm/containerservice"
	"github.com/hashicorp/terraform/helper/hashcode"
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
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      1,
							ValidateFunc: validateArmContainerServiceMasterProfileCount,
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
						"ssh_key": {
							Type:     schema.TypeList,
							Required: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"key_data": {
										Type:     schema.TypeString,
										Required: true,
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
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      1,
							ValidateFunc: validateArmContainerServiceAgentPoolProfileCount,
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
				Optional: true,
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

			"diagnostics_profile": {
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
							Computed: true,
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

	diagnosticsProfile, err := expandAzureRmContainerServiceDiagnostics(d)
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
			AgentPoolProfiles:  &agentProfiles,
			DiagnosticsProfile: &diagnosticsProfile,
		},
		Tags: expandTags(tags),
	}

	servicePrincipalProfile, _ := expandAzureRmContainerServiceServicePrincipal(d)
	if servicePrincipalProfile != nil {
		parameters.ServicePrincipalProfile = servicePrincipalProfile
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

	masterProfiles := flattenAzureRmContainerServiceMasterProfile(*resp.Properties.MasterProfile)
	if masterProfiles != nil {
		d.Set("master_profile", masterProfiles)
	}

	linuxProfile := flattenAzureRmContainerServiceLinuxProfile(*resp.Properties.LinuxProfile)
	if linuxProfile != nil {
		d.Set("linux_profile", linuxProfile)
	}

	agentPoolProfiles := flattenAzureRmContainerServiceAgentPoolProfiles(resp.Properties.AgentPoolProfiles)
	if agentPoolProfiles != nil {
		d.Set("agent_pool_profile", agentPoolProfiles)
	}

	servicePrincipal := flattenAzureRmContainerServiceServicePrincipalProfile(resp.Properties.ServicePrincipalProfile)
	if servicePrincipal != nil {
		d.Set("service_principal", servicePrincipal)
	}

	diagnosticProfile := flattenAzureRmContainerServiceDiagnosticsProfile(resp.Properties.DiagnosticsProfile)
	if diagnosticProfile != nil {
		d.Set("diagnostics_profile", diagnosticProfile)
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

func flattenAzureRmContainerServiceMasterProfile(profile containerservice.MasterProfile) *schema.Set {
	masterProfiles := &schema.Set{
		F: resourceAzureRMContainerServiceMasterProfileHash,
	}

	masterProfile := make(map[string]interface{}, 2)

	masterProfile["count"] = profile.Count
	masterProfile["dns_prefix"] = profile.DNSPrefix

	masterProfiles.Add(masterProfile)

	return masterProfiles
}

func flattenAzureRmContainerServiceLinuxProfile(profile containerservice.LinuxProfile) *schema.Set {
	profiles := schema.Set{
		F: resourceAzureRMContainerServiceLinuxProfilesHash,
	}

	values := map[string]interface{}{}

	sshKeys := schema.Set{
		F: resourceAzureRMContainerServiceLinuxProfilesSSHKeysHash,
	}
	for _, ssh := range *profile.SSH.PublicKeys {
		keys := map[string]interface{}{}
		keys["key_data"] = ssh.KeyData
		sshKeys.Add(keys)
	}

	values["admin_username"] = profile.AdminUsername
	values["ssh_key"] = sshKeys.List()
	profiles.Add(values)

	return &profiles
}

func flattenAzureRmContainerServiceAgentPoolProfiles(profiles *[]containerservice.AgentPoolProfile) *schema.Set {
	agentPoolProfiles := schema.Set{
		F: resourceAzureRMContainerServiceAgentPoolProfilesHash,
	}

	for _, profile := range *profiles {
		agentPoolProfile := map[string]interface{}{}
		agentPoolProfile["count"] = profile.Count
		agentPoolProfile["dns_prefix"] = profile.DNSPrefix
		agentPoolProfile["fqdn"] = profile.Fqdn
		agentPoolProfile["name"] = profile.Name
		agentPoolProfile["vm_size"] = string(profile.VMSize)
		agentPoolProfiles.Add(agentPoolProfile)
	}

	return &agentPoolProfiles
}

func flattenAzureRmContainerServiceServicePrincipalProfile(profile *containerservice.ServicePrincipalProfile) *schema.Set {

	if profile == nil {
		return nil
	}

	servicePrincipalProfiles := schema.Set{
		F: resourceAzureRMContainerServiceServicePrincipalProfileHash,
	}

	values := map[string]interface{}{}

	values["client_id"] = profile.ClientID
	values["client_secret"] = profile.Secret

	servicePrincipalProfiles.Add(values)

	return &servicePrincipalProfiles
}

func flattenAzureRmContainerServiceDiagnosticsProfile(profile *containerservice.DiagnosticsProfile) *schema.Set {
	diagnosticProfiles := schema.Set{
		F: resourceAzureRMContainerServiceDiagnosticProfilesHash,
	}

	values := map[string]interface{}{}

	values["enabled"] = profile.VMDiagnostics.Enabled
	values["storage_uri"] = profile.VMDiagnostics.StorageURI
	diagnosticProfiles.Add(values)

	return &diagnosticProfiles
}

func expandAzureRmContainerServiceDiagnostics(d *schema.ResourceData) (containerservice.DiagnosticsProfile, error) {
	configs := d.Get("diagnostics_profile").(*schema.Set).List()
	profile := containerservice.DiagnosticsProfile{}

	for _, configRaw := range configs {
		data := configRaw.(map[string]interface{})

		enabled := data["enabled"].(bool)

		profile = containerservice.DiagnosticsProfile{
			VMDiagnostics: &containerservice.VMDiagnostics{
				Enabled: &enabled,
			},
		}
	}

	return profile, nil
}

func expandAzureRmContainerServiceLinuxProfile(d *schema.ResourceData) (containerservice.LinuxProfile, error) {
	profiles := d.Get("linux_profile").(*schema.Set).List()
	config := profiles[0].(map[string]interface{})

	adminUsername := config["admin_username"].(string)
	linuxKeys := config["ssh_key"].([]interface{})
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

func expandAzureRmContainerServiceServicePrincipal(d *schema.ResourceData) (*containerservice.ServicePrincipalProfile, error) {

	value, exists := d.GetOk("service_principal")
	if !exists {
		return nil, nil
	}

	configs := value.(*schema.Set).List()

	config := configs[0].(map[string]interface{})

	clientId := config["client_id"].(string)
	clientSecret := config["client_secret"].(string)

	principal := containerservice.ServicePrincipalProfile{
		ClientID: &clientId,
		Secret:   &clientSecret,
	}

	return &principal, nil
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

func resourceAzureRMContainerServiceMasterProfileHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	count := m["count"].(*int32)
	dnsPrefix := m["dns_prefix"].(*string)

	buf.WriteString(fmt.Sprintf("%d-", count))
	buf.WriteString(fmt.Sprintf("%s-", *dnsPrefix))

	return hashcode.String(buf.String())
}

func resourceAzureRMContainerServiceLinuxProfilesHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	adminUsername := m["admin_username"].(*string)

	buf.WriteString(fmt.Sprintf("%s-", *adminUsername))

	return hashcode.String(buf.String())
}

func resourceAzureRMContainerServiceLinuxProfilesSSHKeysHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	keyData := m["key_data"].(*string)

	buf.WriteString(fmt.Sprintf("%s-", *keyData))

	return hashcode.String(buf.String())
}

func resourceAzureRMContainerServiceAgentPoolProfilesHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	count := m["count"].(*int32)
	dnsPrefix := m["dns_prefix"].(*string)
	fqdn := m["fqdn"].(*string)
	name := m["name"].(*string)
	vm_size := m["vm_size"].(string)

	buf.WriteString(fmt.Sprintf("%d-", count))
	buf.WriteString(fmt.Sprintf("%s-", *dnsPrefix))
	buf.WriteString(fmt.Sprintf("%s-", *fqdn))
	buf.WriteString(fmt.Sprintf("%s-", *name))
	buf.WriteString(fmt.Sprintf("%s-", vm_size))

	return hashcode.String(buf.String())
}

func resourceAzureRMContainerServiceServicePrincipalProfileHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	clientId := m["client_id"].(*string)
	clientSecret := m["client_secret"].(*string)

	buf.WriteString(fmt.Sprintf("%s-", *clientId))
	if clientSecret != nil {
		buf.WriteString(fmt.Sprintf("%s-", *clientSecret))
	}

	return hashcode.String(buf.String())
}

func resourceAzureRMContainerServiceDiagnosticProfilesHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	enabled := m["enabled"].(*bool)

	buf.WriteString(fmt.Sprintf("%t", *enabled))

	return hashcode.String(buf.String())
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

func validateArmContainerServiceMasterProfileCount(v interface{}, k string) (ws []string, errors []error) {
	value := v.(int)
	capacities := map[int]bool{
		1: true,
		3: true,
		5: true,
	}

	if !capacities[value] {
		errors = append(errors, fmt.Errorf("The number of master nodes must be 1, 3 or 5."))
	}
	return
}

func validateArmContainerServiceAgentPoolProfileCount(v interface{}, k string) (ws []string, errors []error) {
	value := v.(int)
	if value > 100 || 0 >= value {
		errors = append(errors, fmt.Errorf("The Count for an Agent Pool Profile can only be between 1 and 100."))
	}
	return
}
