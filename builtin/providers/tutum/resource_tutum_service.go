package tutum

import (
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/tutumcloud/go-tutum/tutum"
)

func resourceTutumService() *schema.Resource {
	return &schema.Resource{
		Create: resourceTutumServiceCreate,
		Read:   resourceTutumServiceRead,
		Update: resourceTutumServiceUpdate,
		Delete: resourceTutumServiceDelete,
		Exists: resourceTutumServiceExists,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"image": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},
			"container_count": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
				ForceNew: false,
			},
			"entrypoint": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: false,
			},
			"redeploy_on_change": &schema.Schema{
				Type:     schema.TypeBool,
				Default:  false,
				Optional: true,
				ForceNew: false,
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

func resourceTutumServiceCreate(d *schema.ResourceData, meta interface{}) error {
	opts := &tutum.ServiceCreateRequest{
		Name:  d.Get("name").(string),
		Image: d.Get("image").(string),
	}

	if attr, ok := d.GetOk("entrypoint"); ok {
		opts.Entrypoint = attr.(string)
	}

	if attr, ok := d.GetOk("container_count"); ok {
		opts.Target_num_containers = attr.(int)
	}

	tags := d.Get("tags.#").(int)
	if tags > 0 {
		opts.Tags = make([]string, 0, tags)
		for i := 0; i < tags; i++ {
			key := fmt.Sprintf("tags.%d", i)
			opts.Tags = append(opts.Tags, d.Get(key).(string))
		}
	}

	service, err := tutum.CreateService(*opts)
	if err != nil {
		return err
	}

	if err = service.Start(); err != nil {
		return fmt.Errorf("Error creating service: %s", err)
	}

	d.SetId(service.Uuid)
	d.Set("state", service.State)

	stateConf := &resource.StateChangeConf{
		Pending:        []string{"Starting"},
		Target:         "Running",
		Refresh:        newServiceStateRefreshFunc(d, meta),
		Timeout:        60 * time.Minute,
		Delay:          10 * time.Second,
		MinTimeout:     3 * time.Second,
		NotFoundChecks: 60,
	}

	serviceRaw, err := stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for service (%s) to become ready: %s", d.Id(), err)
	}

	service = serviceRaw.(tutum.Service)
	d.Set("state", service.State)

	return resourceTutumServiceRead(d, meta)
}

func resourceTutumServiceRead(d *schema.ResourceData, meta interface{}) error {
	service, err := tutum.GetService(d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404 NOT FOUND") {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error retrieving service: %s", err)
	}

	if service.State == "Terminated" {
		d.SetId("")
		return nil
	}

	d.Set("name", service.Name)
	d.Set("image", service.Image_name)
	d.Set("container_count", service.Target_num_containers)
	d.Set("entrypoint", service.Entrypoint)
	d.Set("state", service.State)

	return nil
}

func resourceTutumServiceUpdate(d *schema.ResourceData, meta interface{}) error {
	var change bool

	opts := &tutum.ServiceCreateRequest{}

	if d.HasChange("image") {
		_, newImage := d.GetChange("image")
		opts.Image = newImage.(string)
		change = true
	}

	if d.HasChange("entrypoint") {
		_, newEntrypoint := d.GetChange("entrypoint")
		opts.Entrypoint = newEntrypoint.(string)
		change = true
	}

	if d.HasChange("container_count") {
		_, newNum := d.GetChange("container_count")
		opts.Target_num_containers = newNum.(int)
	}

	if d.HasChange("tags") {
		_, newTags := d.GetChange("tags")
		tags := newTags.([]interface{})
		opts.Tags = make([]string, 0, len(tags))

		for _, tag := range tags {
			opts.Tags = append(opts.Tags, tag.(string))
		}
	}

	service, err := tutum.GetService(d.Id())
	if err != nil {
		return fmt.Errorf("Error retrieving service (%s): %s", d.Id(), err)
	}

	if err := service.Update(*opts); err != nil {
		return fmt.Errorf("Error updating service: %s", err)
	}

	if d.Get("redeploy_on_change").(bool) && change {
		if err := service.Redeploy(tutum.ReuseVolumesOption{Reuse: true}); err != nil {
			return fmt.Errorf("Error redeploying containers: %s", err)
		}

		stateConf := &resource.StateChangeConf{
			Pending:        []string{"Redeploying"},
			Target:         "Running",
			Refresh:        newServiceStateRefreshFunc(d, meta),
			Timeout:        60 * time.Minute,
			Delay:          10 * time.Second,
			MinTimeout:     3 * time.Second,
			NotFoundChecks: 60,
		}

		_, err = stateConf.WaitForState()
		if err != nil {
			return fmt.Errorf("Error waiting for service (%s) to finish scaling: %s", d.Id(), err)
		}
	}

	if d.HasChange("container_count") {
		if err := service.Scale(); err != nil {
			return fmt.Errorf("Error updating service: %s", err)
		}

		stateConf := &resource.StateChangeConf{
			Pending:        []string{"Scaling"},
			Target:         "Running",
			Refresh:        newServiceStateRefreshFunc(d, meta),
			Timeout:        60 * time.Minute,
			Delay:          10 * time.Second,
			MinTimeout:     3 * time.Second,
			NotFoundChecks: 60,
		}

		_, err = stateConf.WaitForState()
		if err != nil {
			return fmt.Errorf("Error waiting for service (%s) to finish scaling: %s", d.Id(), err)
		}
	}

	return nil
}

func resourceTutumServiceDelete(d *schema.ResourceData, meta interface{}) error {
	service, err := tutum.GetService(d.Id())
	if err != nil {
		return fmt.Errorf("Error retrieving service (%s): %s", d.Id(), err)
	}

	if service.State == "Terminated" {
		d.SetId("")
		return nil
	}

	if err = service.TerminateService(); err != nil {
		return fmt.Errorf("Error deleting service (%s): %s", d.Id(), err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:        []string{"Terminating", "Stopped"},
		Target:         "Terminated",
		Refresh:        newServiceStateRefreshFunc(d, meta),
		Timeout:        60 * time.Minute,
		Delay:          10 * time.Second,
		MinTimeout:     3 * time.Second,
		NotFoundChecks: 60,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for service (%s) to terminate: %s", d.Id(), err)
	}

	d.SetId("")

	return nil
}

func resourceTutumServiceExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	service, err := tutum.GetService(d.Id())
	if err != nil {
		return false, err
	}

	if service.Uuid == d.Id() {
		return true, nil
	}

	return false, nil
}

func newServiceStateRefreshFunc(d *schema.ResourceData, meta interface{}) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		service, err := tutum.GetService(d.Id())
		if err != nil {
			return nil, "", err
		}

		if service.State == "Stopped" {
			return nil, "", fmt.Errorf("Service entered 'Stopped' state")
		}

		return service, service.State, nil
	}
}
