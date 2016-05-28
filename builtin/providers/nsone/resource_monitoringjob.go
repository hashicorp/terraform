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
			"rules": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"value": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"comparison": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"key": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
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
	d.Set("name", r.Name)
	d.Set("job_type", r.JobType)
	d.Set("active", r.Active)
	d.Set("regions", r.Regions)
	d.Set("frequency", r.Frequency)
	d.Set("rapid_recheck", r.RapidRecheck)
	config := make(map[string]string)
	for k, v := range r.Config {
		if k == "ssl" {
			if v.(bool) {
				config[k] = "1"
			} else {
				config[k] = "0"
			}
		} else {
			switch t := v.(type) {
			case string:
				config[k] = t
			case float64:
				config[k] = strconv.FormatFloat(t, 'f', -1, 64)
			}
		}
	}
	err := d.Set("config", config)
	if err != nil {
		panic(fmt.Errorf("[DEBUG] Error setting Config error: %#v %#v", r.Config, err))
	}
	d.Set("policy", r.Policy)
	d.Set("notes", r.Notes)
	d.Set("frequency", r.Frequency)
	d.Set("notify_delay", r.NotifyDelay)
	d.Set("notify_repeat", r.NotifyRepeat)
	d.Set("notify_regional", r.NotifyRegional)
	d.Set("notify_failback", r.NotifyFailback)
	d.Set("notify_list", r.NotifyList)
	if len(r.Rules) > 0 {
		rules := make([]map[string]interface{}, len(r.Rules))
		for i, r := range r.Rules {
			m := make(map[string]interface{})
			m["value"] = r.Value
			m["comparison"] = r.Comparison
			m["key"] = r.Key
			rules[i] = m
		}
	}
	return nil
}

func resourceDataToMonitoringJob(r *nsone.MonitoringJob, d *schema.ResourceData) error {
	r.Id = d.Id()
	r.Name = d.Get("name").(string)
	r.JobType = d.Get("job_type").(string)
	r.Active = d.Get("active").(bool)
	rawRegions := d.Get("regions").([]interface{})
	r.Regions = make([]string, len(rawRegions))
	for i, v := range rawRegions {
		r.Regions[i] = v.(string)
	}
	r.Frequency = d.Get("frequency").(int)
	r.RapidRecheck = d.Get("rapid_recheck").(bool)
	var rawRules []interface{}
	if rawRules := d.Get("rules"); rawRules != nil {
		r.Rules = make([]nsone.MonitoringJobRule, len(rawRules.([]interface{})))
		for i, v := range rawRules.([]interface{}) {
			rule := v.(map[string]interface{})
			r.Rules[i] = nsone.MonitoringJobRule{
				Value:      rule["value"].(string),
				Comparison: rule["comparison"].(string),
				Key:        rule["key"].(string),
			}
		}
	} else {
		r.Rules = make([]nsone.MonitoringJobRule, 0)
	}
	for i, v := range rawRules {
		rule := v.(map[string]interface{})
		r.Rules[i] = nsone.MonitoringJobRule{
			Comparison: rule["comparison"].(string),
			Key:        rule["key"].(string),
		}
		value := rule["value"].(string)
		if i, err := strconv.Atoi(value); err == nil {
			r.Rules[i].Value = i
		} else {
			r.Rules[i].Value = value
		}
	}
	config := make(map[string]interface{})
	if rawConfig := d.Get("config"); rawConfig != nil {
		for k, v := range rawConfig.(map[string]interface{}) {
			if k == "ssl" {
				if v.(string) == "1" {
					config[k] = true
				}
			} else {
				if i, err := strconv.Atoi(v.(string)); err == nil {
					config[k] = i
				} else {
					config[k] = v
				}
			}
		}
	}
	r.Config = config
	r.RegionScope = "fixed"
	r.Policy = d.Get("policy").(string)
	if v, ok := d.GetOk("notes"); ok {
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

// MonitoringJobCreate Creates monitoring job in ns1
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

// MonitoringJobRead reads the given monitoring job from ns1
func MonitoringJobRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	mj, err := client.GetMonitoringJob(d.Id())
	if err != nil {
		return err
	}
	monitoringJobToResourceData(d, &mj)
	return nil
}

// MonitoringJobDelete deteltes the given monitoring job from ns1
func MonitoringJobDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	err := client.DeleteMonitoringJob(d.Id())
	d.SetId("")
	return err
}

// MonitoringJobUpdate updates the given monitoring job
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
