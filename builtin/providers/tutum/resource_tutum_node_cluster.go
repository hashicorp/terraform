package tutum

import (
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/tutumcloud/go-tutum/tutum"
)

func resourceTutumNodeCluster() *schema.Resource {
	return &schema.Resource{
		Create: resourceTutumNodeClusterCreate,
		Read:   resourceTutumNodeClusterRead,
		Update: resourceTutumNodeClusterUpdate,
		Delete: resourceTutumNodeClusterDelete,
		Exists: resourceTutumNodeClusterExists,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"node_provider": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"size": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"region": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"disk": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"node_count": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
				ForceNew: false,
			},
			"state": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"tags": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: false,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func resourceTutumNodeClusterCreate(d *schema.ResourceData, meta interface{}) error {
	provider := d.Get("node_provider").(string)
	region := d.Get("region").(string)
	size := d.Get("size").(string)

	opts := &tutum.NodeCreateRequest{
		Name:     d.Get("name").(string),
		Region:   fmt.Sprintf("/api/v1/region/%s/%s/", provider, region),
		NodeType: fmt.Sprintf("/api/v1/nodetype/%s/%s/", provider, size),
	}

	if attr, ok := d.GetOk("disk"); ok {
		opts.Disk = attr.(int)
	}

	if attr, ok := d.GetOk("node_count"); ok {
		opts.Target_num_nodes = attr.(int)
	}

	tags := d.Get("tags.#").(int)
	if tags > 0 {
		opts.Tags = make([]tutum.NodeTag, 0, tags)
		for i := 0; i < tags; i++ {
			key := fmt.Sprintf("tags.%d", i)
			opts.Tags = append(opts.Tags, tutum.NodeTag{Name: d.Get(key).(string)})
		}
	}

	nodeCluster, err := tutum.CreateNodeCluster(*opts)
	if err != nil {
		return err
	}

	if err = nodeCluster.Deploy(); err != nil {
		return fmt.Errorf("Error creating node cluster: %s", err)
	}

	d.SetId(nodeCluster.Uuid)
	d.Set("state", nodeCluster.State)

	stateConf := &resource.StateChangeConf{
		Pending:        []string{"Deploying"},
		Target:         "Deployed",
		Refresh:        newNodeClusterStateRefreshFunc(d, meta),
		Timeout:        60 * time.Minute,
		Delay:          10 * time.Second,
		MinTimeout:     3 * time.Second,
		NotFoundChecks: 60,
	}

	nodeClusterRaw, err := stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for node cluster (%s) to become ready: %s", d.Id(), err)
	}

	nodeCluster = nodeClusterRaw.(tutum.NodeCluster)
	d.Set("state", nodeCluster.State)

	return resourceTutumNodeClusterRead(d, meta)
}

func resourceTutumNodeClusterRead(d *schema.ResourceData, meta interface{}) error {
	nodeCluster, err := tutum.GetNodeCluster(d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404 NOT FOUND") {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error retrieving node cluster: %s", err)
	}

	if nodeCluster.State == "Terminated" {
		d.SetId("")
		return nil
	}

	d.Set("name", nodeCluster.Name)
	d.Set("node_count", nodeCluster.Target_num_nodes)
	d.Set("disk", nodeCluster.Disk)
	d.Set("state", nodeCluster.State)

	return nil
}

func resourceTutumNodeClusterUpdate(d *schema.ResourceData, meta interface{}) error {
	opts := &tutum.NodeCreateRequest{}

	if d.HasChange("node_count") {
		_, newNum := d.GetChange("node_count")
		opts.Target_num_nodes = newNum.(int)
	}

	if d.HasChange("tags") {
		_, newTags := d.GetChange("tags")
		tags := newTags.([]interface{})
		opts.Tags = make([]tutum.NodeTag, 0, len(tags))

		for _, tag := range tags {
			opts.Tags = append(opts.Tags, tutum.NodeTag{Name: tag.(string)})
		}
	}

	nodeCluster, err := tutum.GetNodeCluster(d.Id())
	if err != nil {
		return fmt.Errorf("Error retrieving node cluster (%s): %s", d.Id(), err)
	}

	if err := nodeCluster.Update(*opts); err != nil {
		return fmt.Errorf("Error updating node cluster: %s", err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:        []string{"Scaling"},
		Target:         "Deployed",
		Refresh:        newNodeClusterStateRefreshFunc(d, meta),
		Timeout:        60 * time.Minute,
		Delay:          10 * time.Second,
		MinTimeout:     3 * time.Second,
		NotFoundChecks: 60,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for node cluster (%s) to finish scaling: %s", d.Id(), err)
	}

	return nil
}

func resourceTutumNodeClusterDelete(d *schema.ResourceData, meta interface{}) error {
	nodeCluster, err := tutum.GetNodeCluster(d.Id())
	if err != nil {
		return fmt.Errorf("Error retrieving node cluster (%s): %s", d.Id(), err)
	}

	if nodeCluster.State == "Terminated" {
		d.SetId("")
		return nil
	}

	if err = nodeCluster.Terminate(); err != nil {
		return fmt.Errorf("Error deleting node cluster (%s): %s", d.Id(), err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:        []string{"Terminating", "Empty cluster"},
		Target:         "Terminated",
		Refresh:        newNodeClusterStateRefreshFunc(d, meta),
		Timeout:        60 * time.Minute,
		Delay:          10 * time.Second,
		MinTimeout:     3 * time.Second,
		NotFoundChecks: 60,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for node cluster (%s) to terminate: %s", d.Id(), err)
	}

	d.SetId("")

	return nil
}

func resourceTutumNodeClusterExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	nodeCluster, err := tutum.GetNodeCluster(d.Id())
	if err != nil {
		return false, err
	}

	if nodeCluster.Uuid == d.Id() {
		return true, nil
	}

	return false, nil
}

func newNodeClusterStateRefreshFunc(d *schema.ResourceData, meta interface{}) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		nodeCluster, err := tutum.GetNodeCluster(d.Id())
		if err != nil {
			return nil, "", err
		}

		return nodeCluster, nodeCluster.State, nil
	}
}
