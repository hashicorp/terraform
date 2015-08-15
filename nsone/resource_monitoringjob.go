package nsone

import (
	"fmt"
	"github.com/bobtfish/go-nsone-api"
	"github.com/hashicorp/terraform/helper/schema"
	"regexp"
	"strconv"
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
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"job_type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
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
				Optional: true,
			},
			"config": &schema.Schema{
				Type:     schema.TypeMap,
				Required: true,
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
	r.Name = d.Get("name").(string)
	r.JobType = d.Get("job_type").(string)
	r.Active = d.Get("active").(bool)
	raw_regions := d.Get("regions").([]interface{})
	r.Regions = make([]string, len(raw_regions))
	for i, v := range raw_regions {
		r.Regions[i] = v.(string)
	}
	r.Frequency = d.Get("frequency").(int)
	r.RapidRecheck = d.Get("rapid_recheck").(bool)
	var raw_rules []interface{}
	if r := d.Get("rules"); r != nil {
		raw_rules = r.([]interface{})
	}
	r.Rules = make([]nsone.MonitoringJobRule, len(raw_rules))
	for i, v := range raw_rules {
		rule := v.(map[string]interface{})
		r.Rules[i] = nsone.MonitoringJobRule{
			Value:      rule["value"].(int),
			Comparison: rule["comparison"].(string),
			Key:        rule["key"].(string),
		}
	}
	config := make(map[string]interface{})
	if raw_config := d.Get("config"); raw_config != nil {
		for k, v := range raw_config.(map[string]interface{}) {
			if i, err := strconv.Atoi(v.(string)); err == nil {
				config[k] = i
			} else {
				config[k] = v
			}
		}
	}
	r.Config = config
	r.RegionScope = "fixed"
	r.Policy = d.Get("policy").(string)
	if v, ok = d.GetOk("notes"); ok {
		r.Notes = v.(string)
	}
	r.Frequency = d.Get("frequency").(int)
	if v, ok := d.GetOk("notify_delay"); ok {
		r.NotifyDelay = v.(int)
	}
	if v, ok := d.GetOk("notify_repeat"); ok {
		r.NotifyRepeat = v.(int)
	}
	if v, ok := d.GetOk("notify_regional"); ok {
		r.NotifyRegional = v.(bool)
	}
	if v, ok := d.GetOk("notify_failback"); ok {
		r.NotifyFailback = v.(bool)
	}
	if v, ok := d.GetOk("notify_list"); ok {
		r.NotifyList = v.(string)
	}
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
