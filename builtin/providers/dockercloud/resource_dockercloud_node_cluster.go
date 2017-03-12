package dockercloud

import (
	"fmt"
	"strings"
	"time"

	"github.com/docker/go-dockercloud/dockercloud"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

const (
	nodeTypeBasePath = "/api/infra/v1/nodetype"
	regionBasePath   = "/api/infra/v1/region"
)

func resourceDockercloudNodeCluster() *schema.Resource {
	return &schema.Resource{
		Create: resourceDockercloudNodeClusterCreate,
		Read:   resourceDockercloudNodeClusterRead,
		Update: resourceDockercloudNodeClusterUpdate,
		Delete: resourceDockercloudNodeClusterDelete,
		Exists: resourceDockercloudNodeClusterExists,

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
			},
			"tags": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func resourceDockercloudNodeClusterCreate(d *schema.ResourceData, meta interface{}) error {
	provider := d.Get("node_provider").(string)
	region := d.Get("region").(string)
	size := d.Get("size").(string)

	opts := dockercloud.NodeCreateRequest{
		Name:     d.Get("name").(string),
		Region:   fmt.Sprintf("%s/%s/%s/", regionBasePath, provider, region),
		NodeType: fmt.Sprintf("%s/%s/%s/", nodeTypeBasePath, provider, size),
	}

	if attr, ok := d.GetOk("disk"); ok {
		opts.Disk = attr.(int)
	}

	if attr, ok := d.GetOk("node_count"); ok {
		opts.Target_num_nodes = attr.(int)
	}

	tags := d.Get("tags").([]interface{})
	if len(tags) > 0 {
		opts.Tags = make([]dockercloud.NodeTag, len(tags))
		for i, tag := range tags {
			opts.Tags[i] = dockercloud.NodeTag{Name: tag.(string)}
		}
	}

	nodeCluster, err := dockercloud.CreateNodeCluster(opts)
	if err != nil {
		return err
	}

	if err = nodeCluster.Deploy(); err != nil {
		return fmt.Errorf("Error creating node cluster: %s", err)
	}

	d.SetId(nodeCluster.Uuid)

	stateConf := &resource.StateChangeConf{
		Pending:        []string{"Deploying"},
		Target:         []string{"Deployed"},
		Refresh:        newNodeClusterStateRefreshFunc(d, meta),
		Timeout:        60 * time.Minute,
		Delay:          10 * time.Second,
		MinTimeout:     3 * time.Second,
		NotFoundChecks: 60,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for node cluster (%s) to become ready: %s", d.Id(), err)
	}

	return resourceDockercloudNodeClusterRead(d, meta)
}

func resourceDockercloudNodeClusterRead(d *schema.ResourceData, meta interface{}) error {
	nodeCluster, err := dockercloud.GetNodeCluster(d.Id())
	if err != nil {
		if err.(dockercloud.HttpError).StatusCode == 404 {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error retrieving node cluster: %s", err)
	}

	if nodeCluster.State == "Terminated" {
		d.SetId("")
		return nil
	}

	provider, size, err := parseResourceURI(nodeCluster.NodeType, nodeTypeBasePath)
	if err != nil {
		return err
	}

	_, region, err := parseResourceURI(nodeCluster.Region, regionBasePath)
	if err != nil {
		return err
	}

	d.Set("name", nodeCluster.Name)
	d.Set("node_provider", provider)
	d.Set("size", size)
	d.Set("region", region)
	d.Set("disk", nodeCluster.Disk)
	d.Set("node_count", nodeCluster.Target_num_nodes)
	d.Set("tags", flattenNodeTags(nodeCluster.Tags))

	return nil
}

func resourceDockercloudNodeClusterUpdate(d *schema.ResourceData, meta interface{}) error {
	var opts dockercloud.NodeCreateRequest

	if d.HasChange("node_count") {
		_, newNum := d.GetChange("node_count")
		opts.Target_num_nodes = newNum.(int)
	}

	if d.HasChange("tags") {
		_, newTags := d.GetChange("tags")
		tags := newTags.([]interface{})
		opts.Tags = make([]dockercloud.NodeTag, 0, len(tags))

		for _, tag := range tags {
			opts.Tags = append(opts.Tags, dockercloud.NodeTag{Name: tag.(string)})
		}
	}

	nodeCluster, err := dockercloud.GetNodeCluster(d.Id())
	if err != nil {
		return fmt.Errorf("Error retrieving node cluster (%s): %s", d.Id(), err)
	}

	if err = nodeCluster.Update(opts); err != nil {
		return fmt.Errorf("Error updating node cluster: %s", err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:        []string{"Scaling"},
		Target:         []string{"Deployed"},
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

	return resourceDockercloudNodeClusterRead(d, meta)
}

func resourceDockercloudNodeClusterDelete(d *schema.ResourceData, meta interface{}) error {
	nodeCluster, err := dockercloud.GetNodeCluster(d.Id())
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
		Target:         []string{"Terminated"},
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

func resourceDockercloudNodeClusterExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	nodeCluster, err := dockercloud.GetNodeCluster(d.Id())
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
		nodeCluster, err := dockercloud.GetNodeCluster(d.Id())
		if err != nil {
			return nil, "", err
		}

		return nodeCluster, nodeCluster.State, nil
	}
}

func flattenNodeTags(t []dockercloud.NodeTag) []string {
	ret := make([]string, len(t))

	for i, tag := range t {
		ret[i] = tag.Name
	}

	return ret
}

// parseResourceURI returns the provider and value of the resource provided
// For example: /api/infra/v1/region/aws/t2.nano/ would return "aws, "t2.nano", nil
func parseResourceURI(uri string, base string) (string, string, error) {
	s := strings.Split(strings.Trim(strings.TrimPrefix(uri, base), "/"), "/")
	if len(s) != 2 {
		return "", "", fmt.Errorf("Unknown URI format: %s %+v", uri, s)
	}
	return s[0], s[1], nil
}
