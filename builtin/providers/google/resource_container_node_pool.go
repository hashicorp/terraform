package google

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/container/v1"
	"google.golang.org/api/googleapi"
)

func resourceContainerNodePool() *schema.Resource {
	return &schema.Resource{
		Create: resourceContainerNodePoolCreate,
		Read:   resourceContainerNodePoolRead,
		Delete: resourceContainerNodePoolDelete,
		Exists: resourceContainerNodePoolExists,

		Schema: map[string]*schema.Schema{
			"project": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"name": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ConflictsWith: []string{"name_prefix"},
				ForceNew:      true,
			},

			"name_prefix": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"zone": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"cluster": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"initial_node_count": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceContainerNodePoolCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	zone := d.Get("zone").(string)
	cluster := d.Get("cluster").(string)
	nodeCount := d.Get("initial_node_count").(int)

	var name string
	if v, ok := d.GetOk("name"); ok {
		name = v.(string)
	} else if v, ok := d.GetOk("name_prefix"); ok {
		name = resource.PrefixedUniqueId(v.(string))
	} else {
		name = resource.UniqueId()
	}

	nodePool := &container.NodePool{
		Name:             name,
		InitialNodeCount: int64(nodeCount),
	}

	req := &container.CreateNodePoolRequest{
		NodePool: nodePool,
	}

	op, err := config.clientContainer.Projects.Zones.Clusters.NodePools.Create(project, zone, cluster, req).Do()

	if err != nil {
		return fmt.Errorf("Error creating NodePool: %s", err)
	}

	waitErr := containerOperationWait(config, op, project, zone, "creating GKE NodePool", 10, 3)
	if waitErr != nil {
		// The resource didn't actually create
		d.SetId("")
		return waitErr
	}

	log.Printf("[INFO] GKE NodePool %s has been created", name)

	d.SetId(name)

	return resourceContainerNodePoolRead(d, meta)
}

func resourceContainerNodePoolRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	zone := d.Get("zone").(string)
	name := d.Get("name").(string)
	cluster := d.Get("cluster").(string)

	nodePool, err := config.clientContainer.Projects.Zones.Clusters.NodePools.Get(
		project, zone, cluster, name).Do()
	if err != nil {
		return fmt.Errorf("Error reading NodePool: %s", err)
	}

	d.Set("name", nodePool.Name)
	d.Set("initial_node_count", nodePool.InitialNodeCount)

	return nil
}

func resourceContainerNodePoolDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	zone := d.Get("zone").(string)
	name := d.Get("name").(string)
	cluster := d.Get("cluster").(string)

	op, err := config.clientContainer.Projects.Zones.Clusters.NodePools.Delete(
		project, zone, cluster, name).Do()
	if err != nil {
		return fmt.Errorf("Error deleting NodePool: %s", err)
	}

	// Wait until it's deleted
	waitErr := containerOperationWait(config, op, project, zone, "deleting GKE NodePool", 10, 2)
	if waitErr != nil {
		return waitErr
	}

	log.Printf("[INFO] GKE NodePool %s has been deleted", d.Id())

	d.SetId("")

	return nil
}

func resourceContainerNodePoolExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return false, err
	}

	zone := d.Get("zone").(string)
	name := d.Get("name").(string)
	cluster := d.Get("cluster").(string)

	_, err = config.clientContainer.Projects.Zones.Clusters.NodePools.Get(
		project, zone, cluster, name).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			log.Printf("[WARN] Removing Container NodePool %q because it's gone", name)
			// The resource doesn't exist anymore
			return false, err
		}
		// There was some other error in reading the resource
		return true, err
	}
	return true, nil
}
