package openstack

import (
	"fmt"
	"log"
	"time"

	"github.com/gophercloud/gophercloud/openstack/containerinfra/v1/clustertemplates"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceContainerInfraClusterTemplateV1() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceContainerInfraClusterTemplateV1Read,
		Schema: map[string]*schema.Schema{
			"region": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"project_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"user_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"created_at": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"updated_at": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"apiserver_port": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"coe": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"cluster_distro": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"dns_nameserver": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"docker_storage_driver": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"docker_volume_size": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"external_network_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"fixed_network": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"fixed_subnet": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"flavor": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"master_flavor": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"floating_ip_enabled": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"http_proxy": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"https_proxy": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"image": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"insecure_registry": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"keypair_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"labels": {
				Type:     schema.TypeMap,
				Computed: true,
			},

			"master_lb_enabled": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"network_driver": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"no_proxy": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"public": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"registry_enabled": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"server_type": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"tls_disabled": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"volume_driver": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceContainerInfraClusterTemplateV1Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	containerInfraClient, err := config.containerInfraV1Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack container infra client: %s", err)
	}

	name := d.Get("name").(string)
	ct, err := clustertemplates.Get(containerInfraClient, name).Extract()
	if err != nil {
		return fmt.Errorf("Error getting openstack_containerinfra_clustertemplate_v1 %s: %s", name, err)
	}

	d.SetId(ct.UUID)

	d.Set("project_id", ct.ProjectID)
	d.Set("user_id", ct.UserID)
	d.Set("apiserver_port", ct.APIServerPort)
	d.Set("coe", ct.COE)
	d.Set("cluster_distro", ct.ClusterDistro)
	d.Set("dns_nameserver", ct.DNSNameServer)
	d.Set("docker_storage_driver", ct.DockerStorageDriver)
	d.Set("docker_volume_size", ct.DockerVolumeSize)
	d.Set("external_network_id", ct.ExternalNetworkID)
	d.Set("fixed_network", ct.FixedNetwork)
	d.Set("fixed_subnet", ct.FixedSubnet)
	d.Set("flavor", ct.FlavorID)
	d.Set("master_flavor", ct.MasterFlavorID)
	d.Set("floating_ip_enabled", ct.FloatingIPEnabled)
	d.Set("http_proxy", ct.HTTPProxy)
	d.Set("https_proxy", ct.HTTPSProxy)
	d.Set("image", ct.ImageID)
	d.Set("insecure_registry", ct.InsecureRegistry)
	d.Set("keypair_id", ct.KeyPairID)
	d.Set("labels", ct.Labels)
	d.Set("master_lb_enabled", ct.MasterLBEnabled)
	d.Set("network_driver", ct.NetworkDriver)
	d.Set("no_proxy", ct.NoProxy)
	d.Set("public", ct.Public)
	d.Set("registry_enabled", ct.RegistryEnabled)
	d.Set("server_type", ct.ServerType)
	d.Set("tls_disabled", ct.TLSDisabled)
	d.Set("volume_driver", ct.VolumeDriver)

	if err := d.Set("created_at", ct.CreatedAt.Format(time.RFC3339)); err != nil {
		log.Printf("[DEBUG] Unable to set openstack_containerinfra_clustertemplate_v1 created_at: %s", err)
	}
	if err := d.Set("updated_at", ct.UpdatedAt.Format(time.RFC3339)); err != nil {
		log.Printf("[DEBUG] Unable to set openstack_containerinfra_clustertemplate_v1 updated_at: %s", err)
	}

	d.Set("region", GetRegion(d, config))

	return nil
}
