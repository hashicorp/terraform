package openstack

import (
	"fmt"
	"log"
	"time"

	"github.com/gophercloud/gophercloud/openstack/containerinfra/v1/clusters"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceContainerInfraCluster() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceContainerInfraClusterRead,
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

			"api_address": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"coe_version": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"cluster_template_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"container_version": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"create_timeout": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"discovery_url": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"docker_volume_size": {
				Type:     schema.TypeInt,
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

			"keypair": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"labels": {
				Type:     schema.TypeMap,
				Computed: true,
			},

			"master_count": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"node_count": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"master_addresses": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"node_addresses": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"stack_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceContainerInfraClusterRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	containerInfraClient, err := config.containerInfraV1Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack container infra client: %s", err)
	}

	name := d.Get("name").(string)
	c, err := clusters.Get(containerInfraClient, name).Extract()
	if err != nil {
		return fmt.Errorf("Error getting openstack_containerinfra_cluster_v1 %s: %s", name, err)
	}

	d.SetId(c.UUID)

	d.Set("project_id", c.ProjectID)
	d.Set("user_id", c.UserID)
	d.Set("api_address", c.APIAddress)
	d.Set("coe_version", c.COEVersion)
	d.Set("cluster_template_id", c.ClusterTemplateID)
	d.Set("container_version", c.ContainerVersion)
	d.Set("create_timeout", c.CreateTimeout)
	d.Set("discovery_url", c.DiscoveryURL)
	d.Set("docker_volume_size", c.DockerVolumeSize)
	d.Set("flavor", c.FlavorID)
	d.Set("master_flavor", c.MasterFlavorID)
	d.Set("keypair", c.KeyPair)
	d.Set("master_count", c.MasterCount)
	d.Set("node_count", c.NodeCount)
	d.Set("master_addresses", c.MasterAddresses)
	d.Set("node_addresses", c.NodeAddresses)
	d.Set("stack_id", c.StackID)

	if err := d.Set("labels", c.Labels); err != nil {
		log.Printf("[DEBUG] Unable to set labels for openstack_containerinfra_cluster_v1 %s: %s", c.UUID, err)
	}
	if err := d.Set("created_at", c.CreatedAt.Format(time.RFC3339)); err != nil {
		log.Printf("[DEBUG] Unable to set created_at for openstack_containerinfra_cluster_v1 %s: %s", c.UUID, err)
	}
	if err := d.Set("updated_at", c.UpdatedAt.Format(time.RFC3339)); err != nil {
		log.Printf("[DEBUG] Unable to set updated_at for openstack_containerinfra_cluster_v1 %s: %s", c.UUID, err)
	}

	d.Set("region", GetRegion(d, config))

	return nil
}
