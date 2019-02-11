package openstack

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/gophercloud/gophercloud/openstack/containerinfra/v1/clustertemplates"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceContainerInfraClusterTemplateV1() *schema.Resource {
	return &schema.Resource{
		Create: resourceContainerInfraClusterTemplateV1Create,
		Read:   resourceContainerInfraClusterTemplateV1Read,
		Update: resourceContainerInfraClusterTemplateV1Update,
		Delete: resourceContainerInfraClusterTemplateV1Delete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"region": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},

			"project_id": {
				Type:     schema.TypeString,
				ForceNew: true,
				Computed: true,
			},

			"user_id": {
				Type:     schema.TypeString,
				ForceNew: true,
				Computed: true,
			},

			"created_at": {
				Type:     schema.TypeString,
				ForceNew: false,
				Computed: true,
			},

			"updated_at": {
				Type:     schema.TypeString,
				ForceNew: false,
				Computed: true,
			},

			"apiserver_port": {
				Type:         schema.TypeInt,
				Optional:     true,
				ForceNew:     false,
				ValidateFunc: validation.IntBetween(1024, 65535),
			},

			"coe": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},

			"cluster_distro": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
				Computed: true,
			},

			"dns_nameserver": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},

			"docker_storage_driver": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},

			"docker_volume_size": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: false,
			},

			"external_network_id": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},

			"fixed_network": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},

			"fixed_subnet": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},

			"flavor": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
				DefaultFunc: schema.EnvDefaultFunc("OS_MAGNUM_FLAVOR", nil),
			},

			"master_flavor": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
				DefaultFunc: schema.EnvDefaultFunc("OS_MAGNUM_MASTER_FLAVOR", nil),
			},

			"floating_ip_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: false,
			},

			"http_proxy": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},

			"https_proxy": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},

			"image": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    false,
				DefaultFunc: schema.EnvDefaultFunc("OS_MAGNUM_IMAGE", nil),
			},

			"insecure_registry": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},

			"keypair_id": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},

			"labels": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: false,
			},

			"master_lb_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: false,
			},

			"network_driver": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
				Computed: true,
			},

			"no_proxy": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},

			"public": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: false,
			},

			"registry_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: false,
			},

			"server_type": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
				Computed: true,
			},

			"tls_disabled": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: false,
			},

			"volume_driver": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},
		},
	}
}

func resourceContainerInfraClusterTemplateV1Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	containerInfraClient, err := config.containerInfraV1Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack container infra client: %s", err)
	}

	// Get boolean parameters that will be passed by reference.
	floatingIPEnabled := d.Get("floating_ip_enabled").(bool)
	masterLBEnabled := d.Get("master_lb_enabled").(bool)
	public := d.Get("public").(bool)
	registryEnabled := d.Get("registry_enabled").(bool)
	tlsDisabled := d.Get("tls_disabled").(bool)

	// Get and check labels map.
	rawLabels := d.Get("labels").(map[string]interface{})
	labels, err := expandContainerInfraV1LabelsMap(rawLabels)
	if err != nil {
		return err
	}

	createOpts := clustertemplates.CreateOpts{
		COE:                 d.Get("coe").(string),
		DNSNameServer:       d.Get("dns_nameserver").(string),
		DockerStorageDriver: d.Get("docker_storage_driver").(string),
		ExternalNetworkID:   d.Get("external_network_id").(string),
		FixedNetwork:        d.Get("fixed_network").(string),
		FixedSubnet:         d.Get("fixed_subnet").(string),
		FlavorID:            d.Get("flavor").(string),
		MasterFlavorID:      d.Get("master_flavor").(string),
		FloatingIPEnabled:   &floatingIPEnabled,
		HTTPProxy:           d.Get("http_proxy").(string),
		HTTPSProxy:          d.Get("https_proxy").(string),
		ImageID:             d.Get("image").(string),
		InsecureRegistry:    d.Get("insecure_registry").(string),
		KeyPairID:           d.Get("keypair_id").(string),
		Labels:              labels,
		MasterLBEnabled:     &masterLBEnabled,
		Name:                d.Get("name").(string),
		NetworkDriver:       d.Get("network_driver").(string),
		NoProxy:             d.Get("no_proxy").(string),
		Public:              &public,
		RegistryEnabled:     &registryEnabled,
		ServerType:          d.Get("server_type").(string),
		TLSDisabled:         &tlsDisabled,
		VolumeDriver:        d.Get("volume_driver").(string),
	}

	// Set int parameters that will be passed by reference.
	apiServerPort := d.Get("apiserver_port").(int)
	if apiServerPort > 0 {
		createOpts.APIServerPort = &apiServerPort
	}
	dockerVolumeSize := d.Get("docker_volume_size").(int)
	if dockerVolumeSize > 0 {
		createOpts.DockerVolumeSize = &dockerVolumeSize
	}

	log.Printf("[DEBUG] openstack_containerinfra_clustertemplate_v1 create options: %#v", createOpts)

	s, err := clustertemplates.Create(containerInfraClient, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating openstack_containerinfra_clustertemplate_v1: %s", err)
	}

	d.SetId(s.UUID)

	log.Printf("[DEBUG] Created openstack_containerinfra_clustertemplate_v1 %s: %#v", s.UUID, s)
	return resourceContainerInfraClusterTemplateV1Read(d, meta)
}

func resourceContainerInfraClusterTemplateV1Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	containerInfraClient, err := config.containerInfraV1Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack container infra client: %s", err)
	}

	s, err := clustertemplates.Get(containerInfraClient, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "Error retrieving openstack_containerinfra_clustertemplate_v1")
	}

	log.Printf("[DEBUG] Retrieved openstack_containerinfra_clustertemplate_v1 %s: %#v", d.Id(), s)

	if err := d.Set("labels", s.Labels); err != nil {
		return fmt.Errorf("Unable to set labels: %s", err)
	}

	d.Set("apiserver_port", s.APIServerPort)
	d.Set("coe", s.COE)
	d.Set("cluster_distro", s.ClusterDistro)
	d.Set("dns_nameserver", s.DNSNameServer)
	d.Set("docker_storage_driver", s.DockerStorageDriver)
	d.Set("docker_volume_size", s.DockerVolumeSize)
	d.Set("external_network_id", s.ExternalNetworkID)
	d.Set("fixed_network", s.FixedNetwork)
	d.Set("fixed_subnet", s.FixedSubnet)
	d.Set("flavor_id", s.FlavorID)
	d.Set("master_flavor_id", s.MasterFlavorID)
	d.Set("floating_ip_enabled", s.FloatingIPEnabled)
	d.Set("http_proxy", s.HTTPProxy)
	d.Set("https_proxy", s.HTTPSProxy)
	d.Set("image_id", s.ImageID)
	d.Set("insecure_registry", s.InsecureRegistry)
	d.Set("keypair_id", s.KeyPairID)
	d.Set("master_lb_enabled", s.MasterLBEnabled)
	d.Set("network_driver", s.NetworkDriver)
	d.Set("no_proxy", s.NoProxy)
	d.Set("public", s.Public)
	d.Set("registry_enabled", s.RegistryEnabled)
	d.Set("server_type", s.ServerType)
	d.Set("tls_disabled", s.TLSDisabled)
	d.Set("volume_driver", s.VolumeDriver)
	d.Set("region", GetRegion(d, config))
	d.Set("name", s.Name)
	d.Set("project_id", s.ProjectID)
	d.Set("user_id", s.UserID)

	if err := d.Set("created_at", s.CreatedAt.Format(time.RFC3339)); err != nil {
		log.Printf("[DEBUG] Unable to set openstack_containerinfra_clustertemplate_v1 created_at: %s", err)
	}
	if err := d.Set("updated_at", s.UpdatedAt.Format(time.RFC3339)); err != nil {
		log.Printf("[DEBUG] Unable to set openstack_containerinfra_clustertemplate_v1 updated_at: %s", err)
	}

	return nil
}

func resourceContainerInfraClusterTemplateV1Update(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	containerInfraClient, err := config.containerInfraV1Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack container infra client: %s", err)
	}

	updateOpts := []clustertemplates.UpdateOptsBuilder{}

	if d.HasChange("name") {
		v := d.Get("name").(string)
		updateOpts = containerInfraClusterTemplateV1AppendUpdateOpts(updateOpts, "name", v)
	}

	if d.HasChange("apiserver_port") {
		v := d.Get("apiserver_port").(int)
		apiServerPort := strconv.Itoa(v)
		updateOpts = containerInfraClusterTemplateV1AppendUpdateOpts(
			updateOpts, "apiserver_port", apiServerPort)
	}

	if d.HasChange("coe") {
		v := d.Get("coe").(string)
		updateOpts = containerInfraClusterTemplateV1AppendUpdateOpts(updateOpts, "coe", v)
	}

	if d.HasChange("cluster_distro") {
		v := d.Get("cluster_distro").(string)
		updateOpts = containerInfraClusterTemplateV1AppendUpdateOpts(updateOpts, "cluster_distro", v)
	}

	if d.HasChange("dns_nameserver") {
		v := d.Get("dns_nameserver").(string)
		updateOpts = containerInfraClusterTemplateV1AppendUpdateOpts(updateOpts, "dns_nameserver", v)
	}

	if d.HasChange("docker_storage_driver") {
		v := d.Get("docker_storage_driver").(string)
		updateOpts = containerInfraClusterTemplateV1AppendUpdateOpts(updateOpts, "docker_storage_driver", v)
	}

	if d.HasChange("docker_volume_size") {
		v := d.Get("docker_volume_size").(int)
		dockerVolumeSize := strconv.Itoa(v)
		updateOpts = containerInfraClusterTemplateV1AppendUpdateOpts(
			updateOpts, "docker_volume_size", dockerVolumeSize)
	}

	if d.HasChange("external_network_id") {
		v := d.Get("external_network_id").(string)
		updateOpts = containerInfraClusterTemplateV1AppendUpdateOpts(updateOpts, "external_network_id", v)
	}

	if d.HasChange("fixed_network") {
		v := d.Get("fixed_network").(string)
		updateOpts = containerInfraClusterTemplateV1AppendUpdateOpts(updateOpts, "fixed_network", v)
	}

	if d.HasChange("fixed_subnet") {
		v := d.Get("fixed_subnet").(string)
		updateOpts = containerInfraClusterTemplateV1AppendUpdateOpts(updateOpts, "fixed_subnet", v)
	}

	if d.HasChange("flavor") {
		v := d.Get("flavor").(string)
		updateOpts = containerInfraClusterTemplateV1AppendUpdateOpts(updateOpts, "flavor_id", v)
	}

	if d.HasChange("master_flavor") {
		v := d.Get("master_flavor").(string)
		updateOpts = containerInfraClusterTemplateV1AppendUpdateOpts(updateOpts, "master_flavor_id", v)
	}

	if d.HasChange("floating_ip_enabled") {
		v := d.Get("floating_ip_enabled").(bool)
		floatingIPEnabled := strconv.FormatBool(v)
		updateOpts = containerInfraClusterTemplateV1AppendUpdateOpts(
			updateOpts, "floating_ip_enabled", floatingIPEnabled)
	}

	if d.HasChange("http_proxy") {
		v := d.Get("http_proxy").(string)
		updateOpts = containerInfraClusterTemplateV1AppendUpdateOpts(updateOpts, "http_proxy", v)
	}

	if d.HasChange("https_proxy") {
		v := d.Get("https_proxy").(string)
		updateOpts = containerInfraClusterTemplateV1AppendUpdateOpts(updateOpts, "https_proxy", v)
	}

	if d.HasChange("image") {
		v := d.Get("image").(string)
		updateOpts = containerInfraClusterTemplateV1AppendUpdateOpts(updateOpts, "image_id", v)
	}

	if d.HasChange("insecure_registry") {
		v := d.Get("insecure_registry").(string)
		updateOpts = containerInfraClusterTemplateV1AppendUpdateOpts(updateOpts, "insecure_registry", v)
	}

	if d.HasChange("keypair_id") {
		v := d.Get("keypair_id").(string)
		updateOpts = containerInfraClusterTemplateV1AppendUpdateOpts(updateOpts, "keypair_id", v)
	}

	if d.HasChange("labels") {
		rawLabels := d.Get("labels").(map[string]interface{})
		v, err := expandContainerInfraV1LabelsString(rawLabels)
		if err != nil {
			return err
		}
		updateOpts = containerInfraClusterTemplateV1AppendUpdateOpts(updateOpts, "labels", v)
	}

	if d.HasChange("master_lb_enabled") {
		v := d.Get("master_lb_enabled").(bool)
		masterLBEnabled := strconv.FormatBool(v)
		updateOpts = containerInfraClusterTemplateV1AppendUpdateOpts(
			updateOpts, "master_lb_enabled", masterLBEnabled)
	}

	if d.HasChange("network_driver") {
		v := d.Get("network_driver").(string)
		updateOpts = containerInfraClusterTemplateV1AppendUpdateOpts(updateOpts, "network_driver", v)
	}

	if d.HasChange("no_proxy") {
		v := d.Get("no_proxy").(string)
		updateOpts = containerInfraClusterTemplateV1AppendUpdateOpts(updateOpts, "no_proxy", v)
	}

	if d.HasChange("public") {
		v := d.Get("public").(bool)
		public := strconv.FormatBool(v)
		updateOpts = containerInfraClusterTemplateV1AppendUpdateOpts(updateOpts, "public", public)
	}

	if d.HasChange("registry_enabled") {
		v := d.Get("registry_enabled").(bool)
		registryEnabled := strconv.FormatBool(v)
		updateOpts = containerInfraClusterTemplateV1AppendUpdateOpts(
			updateOpts, "registry_enabled", registryEnabled)
	}

	if d.HasChange("server_type") {
		v := d.Get("server_type").(string)
		updateOpts = containerInfraClusterTemplateV1AppendUpdateOpts(updateOpts, "server_type", v)
	}

	if d.HasChange("tls_disabled") {
		v := d.Get("tls_disabled").(bool)
		tlsDisabled := strconv.FormatBool(v)
		updateOpts = containerInfraClusterTemplateV1AppendUpdateOpts(updateOpts, "tls_disabled", tlsDisabled)
	}

	if d.HasChange("volume_driver") {
		v := d.Get("volume_driver").(string)
		updateOpts = containerInfraClusterTemplateV1AppendUpdateOpts(updateOpts, "volume_driver", v)
	}

	log.Printf(
		"[DEBUG] Updating openstack_containerinfra_clustertemplate_v1 %s with options: %#v", d.Id(), updateOpts)

	_, err = clustertemplates.Update(containerInfraClient, d.Id(), updateOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error updating openstack_containerinfra_clustertemplate_v1 %s: %s", d.Id(), err)
	}

	return resourceContainerInfraClusterTemplateV1Read(d, meta)
}

func resourceContainerInfraClusterTemplateV1Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	containerInfraClient, err := config.containerInfraV1Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack container infra client: %s", err)
	}

	if err := clustertemplates.Delete(containerInfraClient, d.Id()).ExtractErr(); err != nil {
		return CheckDeleted(d, err, "Error deleting openstack_containerinfra_clustertemplate_v1")
	}

	return nil
}
