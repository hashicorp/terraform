package openstack

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/gophercloud/gophercloud/openstack/containerinfra/v1/clusters"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceContainerInfraClusterV1() *schema.Resource {
	return &schema.Resource{
		Create: resourceContainerInfraClusterV1Create,
		Read:   resourceContainerInfraClusterV1Read,
		Update: resourceContainerInfraClusterV1Update,
		Delete: resourceContainerInfraClusterV1Delete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(60 * time.Minute),
			Update: schema.DefaultTimeout(30 * time.Minute),
			Delete: schema.DefaultTimeout(30 * time.Minute),
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
				Optional: true,
				ForceNew: true,
				Computed: true,
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

			"api_address": {
				Type:     schema.TypeString,
				ForceNew: false,
				Computed: true,
			},

			"coe_version": {
				Type:     schema.TypeString,
				ForceNew: false,
				Computed: true,
			},

			"cluster_template_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: schema.EnvDefaultFunc("OS_MAGNUM_CLUSTER_TEMPLATE", nil),
			},

			"container_version": {
				Type:     schema.TypeString,
				ForceNew: false,
				Computed: true,
			},

			"create_timeout": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"discovery_url": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"docker_volume_size": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"flavor": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"master_flavor": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"keypair": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"labels": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"master_count": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"node_count": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: false,
				Computed: true,
			},

			"master_addresses": {
				Type:     schema.TypeString,
				ForceNew: false,
				Computed: true,
			},

			"node_addresses": {
				Type:     schema.TypeString,
				ForceNew: false,
				Computed: true,
			},

			"stack_id": {
				Type:     schema.TypeString,
				ForceNew: false,
				Computed: true,
			},
		},
	}
}

func resourceContainerInfraClusterV1Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	containerInfraClient, err := config.containerInfraV1Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack container infra client: %s", err)
	}

	// Get and check labels map.
	rawLabels := d.Get("labels").(map[string]interface{})
	labels, err := expandContainerInfraV1LabelsMap(rawLabels)
	if err != nil {
		return err
	}

	// Determine the flavors to use.
	// First check if it was set in the config.
	// If not, try using the appropriate environment variable.
	flavor, err := containerInfraClusterV1Flavor(d)
	if err != nil {
		return fmt.Errorf("Unable to determine openstack_containerinfra_cluster_v1 flavor")
	}

	masterFlavor, err := containerInfraClusterV1Flavor(d)
	if err != nil {
		return fmt.Errorf("Unable to determine openstack_containerinfra_cluster_v1 master_flavor")
	}

	createOpts := clusters.CreateOpts{
		ClusterTemplateID: d.Get("cluster_template_id").(string),
		DiscoveryURL:      d.Get("discovery_url").(string),
		FlavorID:          flavor,
		Keypair:           d.Get("keypair").(string),
		Labels:            labels,
		MasterFlavorID:    masterFlavor,
		Name:              d.Get("name").(string),
	}

	// Set int parameters that will be passed by reference.
	createTimeout := d.Get("create_timeout").(int)
	if createTimeout > 0 {
		createOpts.CreateTimeout = &createTimeout
	}

	dockerVolumeSize := d.Get("docker_volume_size").(int)
	if dockerVolumeSize > 0 {
		createOpts.DockerVolumeSize = &dockerVolumeSize
	}

	masterCount := d.Get("master_count").(int)
	if masterCount > 0 {
		createOpts.MasterCount = &masterCount
	}

	nodeCount := d.Get("node_count").(int)
	if nodeCount > 0 {
		createOpts.NodeCount = &nodeCount
	}

	s, err := clusters.Create(containerInfraClient, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating openstack_containerinfra_cluster_v1: %s", err)
	}

	// Store the Cluster ID.
	d.SetId(s)

	stateConf := &resource.StateChangeConf{
		Pending:      []string{"CREATE_IN_PROGRESS"},
		Target:       []string{"CREATE_COMPLETE"},
		Refresh:      containerInfraClusterV1StateRefreshFunc(containerInfraClient, s),
		Timeout:      d.Timeout(schema.TimeoutCreate),
		Delay:        1 * time.Minute,
		PollInterval: 20 * time.Second,
	}
	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf(
			"Error waiting for openstack_containerinfra_cluster_v1 %s to become ready: %s", s, err)
	}

	log.Printf("[DEBUG] Created openstack_containerinfra_cluster_v1 %s", s)
	return resourceContainerInfraClusterV1Read(d, meta)
}

func resourceContainerInfraClusterV1Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	containerInfraClient, err := config.containerInfraV1Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack container infra client: %s", err)
	}

	s, err := clusters.Get(containerInfraClient, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "Error retrieving openstack_containerinfra_cluster_v1")
	}

	log.Printf("[DEBUG] Retrieved openstack_containerinfra_cluster_v1 %s: %#v", d.Id(), s)

	if err := d.Set("labels", s.Labels); err != nil {
		return fmt.Errorf("Unable to set openstack_containerinfra_cluster_v1 labels: %s", err)
	}

	d.Set("name", s.Name)
	d.Set("api_address", s.APIAddress)
	d.Set("coe_version", s.COEVersion)
	d.Set("cluster_template_id", s.ClusterTemplateID)
	d.Set("container_version", s.ContainerVersion)
	d.Set("create_timeout", s.CreateTimeout)
	d.Set("discovery_url", s.DiscoveryURL)
	d.Set("docker_volume_size", s.DockerVolumeSize)
	d.Set("flavor", s.FlavorID)
	d.Set("master_flavor", s.MasterFlavorID)
	d.Set("keypair", s.KeyPair)
	d.Set("master_count", s.MasterCount)
	d.Set("node_count", s.NodeCount)
	d.Set("master_addresses", s.MasterAddresses)
	d.Set("node_addresses", s.NodeAddresses)
	d.Set("stack_id", s.StackID)

	if err := d.Set("created_at", s.CreatedAt.Format(time.RFC3339)); err != nil {
		log.Printf("[DEBUG] Unable to set openstack_containerinfra_cluster_v1 created_at: %s", err)
	}
	if err := d.Set("updated_at", s.UpdatedAt.Format(time.RFC3339)); err != nil {
		log.Printf("[DEBUG] Unable to set openstack_containerinfra_cluster_v1 updated_at: %s", err)
	}

	return nil
}

func resourceContainerInfraClusterV1Update(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	containerInfraClient, err := config.containerInfraV1Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack container infra client: %s", err)
	}

	updateOpts := []clusters.UpdateOptsBuilder{}

	if d.HasChange("node_count") {
		v := d.Get("node_count").(int)
		nodeCount := strconv.Itoa(v)
		updateOpts = append(updateOpts, clusters.UpdateOpts{
			Op:    clusters.ReplaceOp,
			Path:  strings.Join([]string{"/", "node_count"}, ""),
			Value: nodeCount,
		})
	}

	log.Printf(
		"[DEBUG] Updating openstack_containerinfra_cluster_v1 %s with options: %#v", d.Id(), updateOpts)

	_, err = clusters.Update(containerInfraClient, d.Id(), updateOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error updating openstack_containerinfra_cluster_v1 %s: %s", d.Id(), err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:      []string{"UPDATE_IN_PROGRESS"},
		Target:       []string{"UPDATE_COMPLETE"},
		Refresh:      containerInfraClusterV1StateRefreshFunc(containerInfraClient, d.Id()),
		Timeout:      d.Timeout(schema.TimeoutUpdate),
		Delay:        1 * time.Minute,
		PollInterval: 20 * time.Second,
	}
	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf(
			"Error waiting for openstack_containerinfra_cluster_v1 %s to become updated: %s", d.Id(), err)
	}

	return resourceContainerInfraClusterV1Read(d, meta)
}

func resourceContainerInfraClusterV1Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	containerInfraClient, err := config.containerInfraV1Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack container infra client: %s", err)
	}

	if err := clusters.Delete(containerInfraClient, d.Id()).ExtractErr(); err != nil {
		return CheckDeleted(d, err, "Error deleting openstack_containerinfra_cluster_v1")
	}

	stateConf := &resource.StateChangeConf{
		Pending:      []string{"DELETE_IN_PROGRESS"},
		Target:       []string{"DELETE_COMPLETE"},
		Refresh:      containerInfraClusterV1StateRefreshFunc(containerInfraClient, d.Id()),
		Timeout:      d.Timeout(schema.TimeoutDelete),
		Delay:        30 * time.Second,
		PollInterval: 10 * time.Second,
	}
	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf(
			"Error waiting for openstack_containerinfra_cluster_v1 %s to become deleted: %s", d.Id(), err)
	}

	return nil
}
