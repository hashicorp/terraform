package ovh

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/ovh/go-ovh/ovh"
)

func resourceVRackPublicCloudAttachment() *schema.Resource {
	return &schema.Resource{
		Create: resourceVRackPublicCloudAttachmentCreate,
		Read:   resourceVRackPublicCloudAttachmentRead,
		Delete: resourceVRackPublicCloudAttachmentDelete,

		Schema: map[string]*schema.Schema{
			"vrack_id": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: schema.EnvDefaultFunc("OVH_VRACK_ID", ""),
			},
			"project_id": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: schema.EnvDefaultFunc("OVH_PROJECT_ID", ""),
			},
		},
	}
}

func resourceVRackPublicCloudAttachmentCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	vrackId := d.Get("vrack_id").(string)
	projectId := d.Get("project_id").(string)

	if err := vrackPublicCloudAttachmentExists(vrackId, projectId, config.OVHClient); err == nil {
		//set id
		d.SetId(fmt.Sprintf("vrack_%s-cloudproject_%s-attach", vrackId, projectId))
		return nil
	}

	params := &VRackAttachOpts{Project: projectId}
	r := VRackAttachTaskResponse{}

	log.Printf("[DEBUG] Will Attach VRack %s -> PublicCloud %s", vrackId, params.Project)
	endpoint := fmt.Sprintf("/vrack/%s/cloudProject", vrackId)

	err := config.OVHClient.Post(endpoint, params, &r)
	if err != nil {
		return fmt.Errorf("Error calling %s with params %s:\n\t %q", endpoint, params, err)
	}

	log.Printf("[DEBUG] Waiting for Attachement Task id %d: VRack %s ->  PublicCloud %s", r.Id, vrackId, params.Project)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"init", "todo", "doing"},
		Target:     []string{"completed"},
		Refresh:    waitForVRackTaskCompleted(config.OVHClient, vrackId, r.Id),
		Timeout:    10 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for vrack (%s) to attach to public cloud (%s): %s", vrackId, params.Project, err)
	}
	log.Printf("[DEBUG] Created Attachement Task id %d: VRack %s ->  PublicCloud %s", r.Id, vrackId, params.Project)

	//set id
	d.SetId(fmt.Sprintf("vrack_%s-cloudproject_%s-attach", vrackId, params.Project))

	return nil
}

func resourceVRackPublicCloudAttachmentRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	vrackId := d.Get("vrack_id").(string)
	params := &VRackAttachOpts{Project: d.Get("project_id").(string)}
	r := VRackAttachTaskResponse{}
	endpoint := fmt.Sprintf("/vrack/%s/cloudProject/%s", vrackId, params.Project)

	err := config.OVHClient.Get(endpoint, &r)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] Read VRack %s ->  PublicCloud %s", vrackId, params.Project)

	return nil
}

func resourceVRackPublicCloudAttachmentDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	vrackId := d.Get("vrack_id").(string)
	params := &VRackAttachOpts{Project: d.Get("project_id").(string)}

	r := VRackAttachTaskResponse{}
	endpoint := fmt.Sprintf("/vrack/%s/cloudProject/%s", vrackId, params.Project)

	err := config.OVHClient.Delete(endpoint, &r)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Waiting for Attachment Deletion Task id %d: VRack %s ->  PublicCloud %s", r.Id, vrackId, params.Project)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"init", "todo", "doing"},
		Target:     []string{"completed"},
		Refresh:    waitForVRackTaskCompleted(config.OVHClient, vrackId, r.Id),
		Timeout:    10 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for vrack (%s) to attach to public cloud (%s): %s", vrackId, params.Project, err)
	}
	log.Printf("[DEBUG] Removed Attachement id %d: VRack %s ->  PublicCloud %s", r.Id, vrackId, params.Project)

	d.SetId("")
	return nil
}

func vrackPublicCloudAttachmentExists(vrackId, projectId string, c *ovh.Client) error {
	type attachResponse struct {
		VRack   string `json:"vrack"`
		Project string `json:"project"`
	}

	r := attachResponse{}

	endpoint := fmt.Sprintf("/vrack/%s/cloudProject/%s", vrackId, projectId)

	err := c.Get(endpoint, &r)
	if err != nil {
		return fmt.Errorf("Error while querying %s: %q\n", endpoint, err)
	}
	log.Printf("[DEBUG] Read Attachment %s -> VRack:%s, Cloud Project: %s", endpoint, r.VRack, r.Project)

	return nil
}

// AttachmentStateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// an Attachment Task.
func waitForVRackTaskCompleted(c *ovh.Client, serviceName string, taskId int) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		r := VRackAttachTaskResponse{}
		endpoint := fmt.Sprintf("/vrack/%s/task/%d", serviceName, taskId)
		err := c.Get(endpoint, &r)
		if err != nil {
			if err.(*ovh.APIError).Code == 404 {
				log.Printf("[DEBUG] Task id %d on VRack %s completed", taskId, serviceName)
				return taskId, "completed", nil
			} else {
				return taskId, "", err
			}
		}

		log.Printf("[DEBUG] Pending Task id %d on VRack %s status: %s", r.Id, serviceName, r.Status)
		return taskId, r.Status, nil
	}
}
