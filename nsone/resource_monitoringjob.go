package nsone

import (
	"fmt"
	"github.com/bobtfish/go-nsone-api"
	"github.com/hashicorp/terraform/helper/schema"
	"regexp"
)

func monitoringJobResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"active": &schema.Schema{
				Type:     schema.TypeBool,
				Required: true,
			},
			"regions": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
			},
			"job_type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"frequency": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},
			"rapid_recheck": &schema.Schema{
				Type:     schema.TypeBool,
				Required: true,
			},
			"policy": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
					value := v.(string)
					if !regexp.MustCompile(`^(all|one|quorum)$`).MatchString(value) {
						es = append(es, fmt.Errorf(
							"only all, one, quorum allowed in %q", k))
					}
					return
				},
			},
			"notes": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"config": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
			},
			"notify_delay": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"notify_repeat": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"notify_failback": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"notify_regional": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"notify_list": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
		},
		Create: MonitoringJobCreate,
		Read:   MonitoringJobRead,
		Update: MonitoringJobUpdate,
		Delete: MonitoringJobDelete,
	}
}

func monitoringJobToResourceData(d *schema.ResourceData, r *nsone.MonitoringJob) error {
	d.SetId(r.Id)
	return nil
}

func resourceDataToMonitoringJob(r *nsone.MonitoringJob, d *schema.ResourceData) error {
	r.Id = d.Id()
	return nil
}

func MonitoringJobCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	mj := nsone.MonitoringJob{}
	if err := resourceDataToMonitoringJob(&mj, d); err != nil {
		return err
	}
	if err := client.CreateMonitoringJob(&mj); err != nil {
		return err
	}
	return monitoringJobToResourceData(d, &mj)
}

func MonitoringJobRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	mj, err := client.GetMonitoringJob(d.Id())
	if err != nil {
		return err
	}
	monitoringJobToResourceData(d, &mj)
	return nil
}

func MonitoringJobDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	err := client.DeleteMonitoringJob(d.Id())
	d.SetId("")
	return err
}

func MonitoringJobUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	mj := nsone.MonitoringJob{
		Id: d.Id(),
	}
	if err := resourceDataToMonitoringJob(&mj, d); err != nil {
		return err
	}
	if err := client.UpdateMonitoringJob(&mj); err != nil {
		return err
	}
	monitoringJobToResourceData(d, &mj)
	return nil
}
