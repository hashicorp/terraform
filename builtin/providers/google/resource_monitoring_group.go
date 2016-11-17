package google

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/monitoring/v3"
	"time"
)

func resourceMonitoringGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceMonitoringGroupCreate,
		Read:   resourceMonitoringGroupRead,
		Delete: resourceMonitoringGroupDelete,
		Update: resourceMonitoringGroupUpdate,

		Schema: map[string]*schema.Schema{

			"project": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"Name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"displayName": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"parentName": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
				ForceNew: true,
			},

			"filter": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"isCluster": &schema.Schema{
				Type:     schema.TypeBool,
				Default:  false,
				Optional: true,
			},
		},
	}
}

func getGroup(d *schema.ResourceData, config *Config) (*monitoring.Group, error) {
	call := config.clientMonitoring.Projects.Groups.Get(d.Id())
	group, err := call.Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			log.Printf("[WARN] Removing Group %q because it's gone", d.Get("displayName").(string))
			// The resource doesn't exist anymore
			id := d.Id()
			d.SetId("")

			return nil, fmt.Errorf("Resource %s no longer exists", id)
		}

		return nil, fmt.Errorf("Error reading group: %s", err)
	}

	return group, nil
}

func findGroup(d *schema.ResourceData, config *Config) (*monitoring.Group, error) {
	var res *monitoring.ListGroupsResponse = nil
	var listParam string = ""

	// Determine if we're in a pagination recursion
	nextPageToken, ok := d.GetOk("nextPageToken")
	if !ok {
		project, err := getProject(d, config)
		if err != nil {
			return nil, err
		}
		listParam = fmt.Sprintf("projects/%s", project)
	} else {
		listParam = nextPageToken.(string)
	}
	query := config.clientMonitoring.Projects.Groups.List(listParam)
	// If a parent is configured set that on the query
	v, ok := d.GetOk("parentName")
	if ok {
		query.ChildrenOfGroup(v.(string))
	}
	res, err := query.Do()
	if err != nil {
		return nil, err
	}

	// Check every group on this page for a matching display name
	for _, element := range res.Group {
		if element.DisplayName == d.Get("displayName") {
			return element, nil
		}
	}
	// If there's a next page token we'll need to recurse
	if res.NextPageToken != "" {
		d.Set("nextPageToken", res.NextPageToken)
		return findGroup(d, config)
	} // No next page, done checking and not found.
	return nil, nil
}

func resourceMonitoringGroupCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	// Build the group
	group := &monitoring.Group{}

	if v, ok := d.GetOk("displayName"); ok {
		group.DisplayName = v.(string)
	}

	if v, ok := d.GetOk("parentName"); ok {
		group.ParentName = v.(string)
	}

	if v, ok := d.GetOk("filter"); ok {
		group.Filter = v.(string)
	}

	if v, ok := d.GetOk("isCluster"); ok {
		group.IsCluster = v.(bool)
	}

	// Add the group
	name := fmt.Sprintf("projects/%s", project)
	call := config.clientMonitoring.Projects.Groups.Create(name, group)
	res, err := call.Do()
	if err != nil {
		return err
	}

	// Wait until it's created
	wait := resource.StateChangeConf{
		Delay:          2 * time.Second,
		Pending:        []string{"BUILDING"},
		Target:         []string{"DONE"},
		Timeout:        2 * time.Minute,
		MinTimeout:     1 * time.Second,
		NotFoundChecks: 5,
		Refresh: func() (interface{}, string, error) {
			status := "BUILDING"
			group, err := findGroup(d, config)
			if err != nil {
				log.Printf("[ERROR] Failed to get group: %s", err)
			}
			if group != nil {
				status = "DONE"
			}
			return group, status, err
		},
	}

	_, err = wait.WaitForState()
	if err != nil {
		return err
	}

	d.SetId(res.Name)
	d.Set("Name", res.Name)

	return resourceMonitoringGroupRead(d, meta)
}

func resourceMonitoringGroupRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	_, err := getGroup(d, config)
	if err != nil {
		return fmt.Errorf("Error getting group: %s", err)
	}

	return nil
}

func resourceMonitoringGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	group, err := getGroup(d, config)

	// Enable partial mode for the resource since it is possible
	d.Partial(true)

	if d.HasChange("displayName") {
		if v, ok := d.GetOk("displayName"); ok {
			group.DisplayName = v.(string)
		}
	}

	if d.HasChange("filter") {
		if v, ok := d.GetOk("filter"); ok {
			group.Filter = v.(string)
		}
	}

	if d.HasChange("isCluster") {
		if v, ok := d.GetOk("isCluster"); ok {
			group.IsCluster = v.(bool)
		}
	}

	call := config.clientMonitoring.Projects.Groups.Update(group.Name, group)
	_, err = call.Do()
	if err != nil {
		return err
	}

	// We made it, disable partial mode
	d.Partial(false)

	return resourceMonitoringGroupRead(d, meta)
}

func resourceMonitoringGroupDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	log.Print("[DEBUG] monitoring group delete request")
	_, err := config.clientMonitoring.Projects.Groups.Delete(d.Id()).Do()
	if err != nil {
		return fmt.Errorf("Error deleting group: %s", err)
	}

	d.SetId("")
	return nil
}
