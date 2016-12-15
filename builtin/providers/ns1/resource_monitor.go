package ns1

import (
	"log"

	"github.com/hashicorp/terraform/helper/schema"

	ns1 "gopkg.in/ns1/ns1-go.v2/rest"
	"gopkg.in/ns1/ns1-go.v2/rest/model/monitor"
)

func monitorJobResource() *schema.Resource {
	return &schema.Resource{
		Create: resourceNS1MonitorJobCreate,
		Read:   resourceNS1MonitorJobRead,
		Update: resourceNS1MonitorJobUpdate,
		Delete: resourceNS1MonitorJobDelete,
		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"type": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateJob,
			},
			"active": &schema.Schema{
				Type:     schema.TypeBool,
				Required: true,
			},
			"regions": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
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
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validatePolicy,
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
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validateNotifyRepeat,
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
						"key": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"value": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"comparison": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func resourceNS1MonitorJobCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)

	j := buildNS1JobStruct(d)

	log.Printf("[INFO] Creating NS1 monitoring job: %s \n", j.Name)

	if _, err := client.Jobs.Create(j); err != nil {
		return err
	}

	d.SetId(j.ID)

	return resourceNS1MonitorJobRead(d, meta)
}

func resourceNS1MonitorJobRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)

	log.Printf("[INFO] Reading NS1 monitoring job: %s \n", d.Id())

	j, _, err := client.Jobs.Get(d.Id())
	if err != nil {
		return err
	}

	d.Set("name", j.Name)
	d.Set("type", j.Type)
	d.Set("active", j.Active)
	d.Set("regions", j.Regions)
	d.Set("frequency", j.Frequency)
	d.Set("rapid_recheck", j.RapidRecheck)
	d.Set("policy", j.Policy)
	d.Set("notes", j.Notes)
	d.Set("frequency", j.Frequency)
	d.Set("notify_delay", j.NotifyDelay)
	d.Set("notify_repeat", j.NotifyRepeat)
	d.Set("notify_regional", j.NotifyRegional)
	d.Set("notify_failback", j.NotifyFailback)
	d.Set("notify_list", j.NotifyListID)
	d.Set("config", j.Config)
	d.Set("rules", flattenNS1JobRules(j.Rules))

	return nil
}

func resourceNS1MonitorJobUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)

	j := buildNS1JobStruct(d)
	j.ID = d.Id()

	log.Printf("[INFO] Updating NS1 monitoring job: %s \n", j.ID)

	if _, err := client.Jobs.Update(j); err != nil {
		return err
	}

	return nil
}

func resourceNS1MonitorJobDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)

	log.Printf("[INFO] Updating NS1 monitoring job: %s \n", d.Id())

	if _, err := client.Jobs.Delete(d.Id()); err != nil {
		return err
	}

	d.SetId("")

	return nil
}

func buildNS1JobStruct(d *schema.ResourceData) *monitor.Job {
	j := &monitor.Job{}

	j.Name = d.Get("name").(string)
	j.Type = d.Get("type").(string)
	j.Active = d.Get("active").(bool)
	j.Frequency = d.Get("frequency").(int)
	j.RapidRecheck = d.Get("rapid_recheck").(bool)
	j.RegionScope = "fixed"
	j.Policy = d.Get("policy").(string)
	j.Frequency = d.Get("frequency").(int)

	if v, ok := d.GetOk("notes"); ok {
		j.Notes = v.(string)
	}
	if v, ok := d.GetOk("notify_delay"); ok {
		j.NotifyDelay = v.(int)
	}
	if v, ok := d.GetOk("notify_repeat"); ok {
		j.NotifyRepeat = v.(int)
	}
	if v, ok := d.GetOk("notify_regional"); ok {
		j.NotifyRegional = v.(bool)
	}
	if v, ok := d.GetOk("notify_failback"); ok {
		j.NotifyFailback = v.(bool)
	}
	if v, ok := d.GetOk("notify_list"); ok {
		j.NotifyListID = v.(string)
	}

	j.Config = d.Get("config").(map[string]interface{})

	regions := d.Get("regions").([]interface{})
	j.Regions = make([]string, len(regions))
	for i, v := range regions {
		j.Regions[i] = v.(string)
	}

	j.Rules = expandNS1JobRules(d)

	return j
}
