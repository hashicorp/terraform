package dockercloud

import (
	"fmt"
	"strings"
	"time"

	"github.com/docker/go-dockercloud/dockercloud"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDockercloudStack() *schema.Resource {
	return &schema.Resource{
		Create: resourceDockercloudStackCreate,
		Read:   resourceDockercloudStackRead,
		Delete: resourceDockercloudStackDelete,
		Exists: resourceDockercloudStackExists,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"uri": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceDockercloudStackCreate(d *schema.ResourceData, meta interface{}) error {
	opts := dockercloud.StackCreateRequest{
		Name: d.Get("name").(string),
	}

	stack, err := dockercloud.CreateStack(opts)
	if err != nil {
		if strings.Contains(err.Error(), "409 CONFLICT") {
			return fmt.Errorf("Duplicate stack name: %s", opts.Name)
		}
		return err
	}

	d.SetId(stack.Uuid)
	d.Set("uri", stack.Resource_uri)

	return resourceDockercloudStackRead(d, meta)
}

func resourceDockercloudStackRead(d *schema.ResourceData, meta interface{}) error {
	stack, err := dockercloud.GetStack(d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404 NOT FOUND") {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error retrieving stack: %s", err)
	}

	if stack.State == "Terminated" {
		d.SetId("")
		return nil
	}

	d.Set("uri", stack.Resource_uri)

	return nil
}

func resourceDockercloudStackDelete(d *schema.ResourceData, meta interface{}) error {
	stack, err := dockercloud.GetStack(d.Id())
	if err != nil {
		return fmt.Errorf("Error retrieving stack (%s): %s", d.Id(), err)
	}

	if stack.State == "Terminated" {
		d.SetId("")
		return nil
	}

	if err = stack.Terminate(); err != nil {
		return fmt.Errorf("Error deleting stack (%s): %s", d.Id(), err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:        []string{"Terminating", "Stopped"},
		Target:         []string{"Terminated"},
		Refresh:        newStackStateRefreshFunc(d, meta),
		Timeout:        60 * time.Minute,
		Delay:          10 * time.Second,
		MinTimeout:     3 * time.Second,
		NotFoundChecks: 60,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for stack (%s) to terminate: %s", d.Id(), err)
	}

	d.SetId("")

	return nil
}

func resourceDockercloudStackExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	stack, err := dockercloud.GetStack(d.Id())
	if err != nil {
		return false, err
	}

	if stack.Uuid == d.Id() {
		return true, nil
	}

	return false, nil
}

func newStackStateRefreshFunc(d *schema.ResourceData, meta interface{}) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		stack, err := dockercloud.GetStack(d.Id())
		if err != nil {
			return nil, "", err
		}

		if stack.State == "Stopped" {
			return nil, "", fmt.Errorf("Stack entered 'Stopped' state")
		}

		return stack, stack.State, nil
	}
}
