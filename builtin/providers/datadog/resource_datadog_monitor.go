package datadog

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/zorkian/go-datadog-api"
)

func resourceDatadogMonitor() *schema.Resource {
	return &schema.Resource{
		Create: resourceDatadogMonitorCreate,
		Read:   resourceDatadogMonitorRead,
		Update: resourceDatadogMonitorUpdate,
		Delete: resourceDatadogMonitorDelete,
		Exists: resourceDatadogMonitorExists,
		Importer: &schema.ResourceImporter{
			State: resourceDatadogImport,
		},

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"message": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				StateFunc: func(val interface{}) string {
					return strings.TrimSpace(val.(string))
				},
			},
			"escalation_message": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				StateFunc: func(val interface{}) string {
					return strings.TrimSpace(val.(string))
				},
			},
			"query": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				StateFunc: func(val interface{}) string {
					return strings.TrimSpace(val.(string))
				},
			},
			"type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			// Options
			"thresholds": &schema.Schema{
				Type:     schema.TypeMap,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ok": &schema.Schema{
							Type:     schema.TypeFloat,
							Optional: true,
						},
						"warning": &schema.Schema{
							Type:     schema.TypeFloat,
							Optional: true,
						},
						"critical": &schema.Schema{
							Type:     schema.TypeFloat,
							Required: true,
						},
					},
				},
				DiffSuppressFunc: supressDataDogFloatIntDiff,
			},
			"notify_no_data": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"no_data_timeframe": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"renotify_interval": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"notify_audit": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"timeout_h": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"require_full_window": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"locked": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			// TODO should actually be map[string]int
			"silenced": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					Elem: &schema.Schema{
						Type: schema.TypeInt},
				},
			},
			"include_tags": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"tags": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					Elem: &schema.Schema{
						Type: schema.TypeString},
				},
			},
		},
	}
}

func buildMonitorStruct(d *schema.ResourceData) *datadog.Monitor {

	var thresholds datadog.ThresholdCount

	if r, ok := d.GetOk("thresholds.ok"); ok {
		thresholds.Ok = json.Number(r.(string))
	}
	if r, ok := d.GetOk("thresholds.warning"); ok {
		thresholds.Warning = json.Number(r.(string))
	}
	if r, ok := d.GetOk("thresholds.critical"); ok {
		thresholds.Critical = json.Number(r.(string))
	}

	o := datadog.Options{
		Thresholds: thresholds,
	}
	if attr, ok := d.GetOk("silenced"); ok {
		s := make(map[string]int)
		// TODO: this is not very defensive, test if we can fail on non int input
		for k, v := range attr.(map[string]interface{}) {
			s[k], _ = strconv.Atoi(v.(string))
		}
		o.Silenced = s
	}
	if attr, ok := d.GetOk("notify_no_data"); ok {
		o.NotifyNoData = attr.(bool)
	}
	if attr, ok := d.GetOk("no_data_timeframe"); ok {
		o.NoDataTimeframe = datadog.NoDataTimeframe(attr.(int))
	}
	if attr, ok := d.GetOk("renotify_interval"); ok {
		o.RenotifyInterval = attr.(int)
	}
	if attr, ok := d.GetOk("notify_audit"); ok {
		o.NotifyAudit = attr.(bool)
	}
	if attr, ok := d.GetOk("timeout_h"); ok {
		o.TimeoutH = attr.(int)
	}
	if attr, ok := d.GetOk("escalation_message"); ok {
		o.EscalationMessage = attr.(string)
	}
	if attr, ok := d.GetOk("include_tags"); ok {
		o.IncludeTags = attr.(bool)
	}
	if attr, ok := d.GetOk("require_full_window"); ok {
		o.RequireFullWindow = attr.(bool)
	}
	if attr, ok := d.GetOk("locked"); ok {
		o.Locked = attr.(bool)
	}

	m := datadog.Monitor{
		Type:    d.Get("type").(string),
		Query:   d.Get("query").(string),
		Name:    d.Get("name").(string),
		Message: d.Get("message").(string),
		Options: o,
	}

	if attr, ok := d.GetOk("tags"); ok {
		s := make([]string, 0)
		for k, v := range attr.(map[string]interface{}) {
			s = append(s, fmt.Sprintf("%s:%s", k, v.(string)))
		}
		m.Tags = s
	}

	return &m
}

func resourceDatadogMonitorExists(d *schema.ResourceData, meta interface{}) (b bool, e error) {
	// Exists - This is called to verify a resource still exists. It is called prior to Read,
	// and lowers the burden of Read to be able to assume the resource exists.
	client := meta.(*datadog.Client)

	i, err := strconv.Atoi(d.Id())
	if err != nil {
		return false, err
	}

	if _, err = client.GetMonitor(i); err != nil {
		if strings.Contains(err.Error(), "404 Not Found") {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func resourceDatadogMonitorCreate(d *schema.ResourceData, meta interface{}) error {

	client := meta.(*datadog.Client)

	m := buildMonitorStruct(d)
	m, err := client.CreateMonitor(m)
	if err != nil {
		return fmt.Errorf("error updating montor: %s", err.Error())
	}

	d.SetId(strconv.Itoa(m.Id))

	return nil
}

func resourceDatadogMonitorRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*datadog.Client)

	i, err := strconv.Atoi(d.Id())
	if err != nil {
		return err
	}

	m, err := client.GetMonitor(i)
	if err != nil {
		return err
	}

	thresholds := make(map[string]string)
	for k, v := range map[string]json.Number{
		"ok":       m.Options.Thresholds.Ok,
		"warning":  m.Options.Thresholds.Warning,
		"critical": m.Options.Thresholds.Critical,
	} {
		s := v.String()
		if s != "" {
			thresholds[k] = s
		}
	}

	tags := make(map[string]string)
	for _, s := range m.Tags {
		tag := strings.Split(s, ":")
		tags[tag[0]] = tag[1]
	}

	log.Printf("[DEBUG] monitor: %v", m)
	d.Set("name", m.Name)
	d.Set("message", m.Message)
	d.Set("query", m.Query)
	d.Set("type", m.Type)
	d.Set("thresholds", thresholds)
	d.Set("notify_no_data", m.Options.NotifyNoData)
	d.Set("no_data_timeframe", m.Options.NoDataTimeframe)
	d.Set("renotify_interval", m.Options.RenotifyInterval)
	d.Set("notify_audit", m.Options.NotifyAudit)
	d.Set("timeout_h", m.Options.TimeoutH)
	d.Set("escalation_message", m.Options.EscalationMessage)
	d.Set("silenced", m.Options.Silenced)
	d.Set("include_tags", m.Options.IncludeTags)
	d.Set("tags", tags)
	d.Set("require_full_window", m.Options.RequireFullWindow)
	d.Set("locked", m.Options.Locked)

	return nil
}

func resourceDatadogMonitorUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*datadog.Client)

	m := &datadog.Monitor{}

	i, err := strconv.Atoi(d.Id())
	if err != nil {
		return err
	}

	m.Id = i
	if attr, ok := d.GetOk("name"); ok {
		m.Name = attr.(string)
	}
	if attr, ok := d.GetOk("message"); ok {
		m.Message = attr.(string)
	}
	if attr, ok := d.GetOk("query"); ok {
		m.Query = attr.(string)
	}

	if attr, ok := d.GetOk("tags"); ok {
		s := make([]string, 0)
		for k, v := range attr.(map[string]interface{}) {
			s = append(s, fmt.Sprintf("%s:%s", k, v.(string)))
		}
		m.Tags = s
	}

	o := datadog.Options{}
	if attr, ok := d.GetOk("thresholds"); ok {
		thresholds := attr.(map[string]interface{})
		if thresholds["ok"] != nil {
			o.Thresholds.Ok = json.Number(thresholds["ok"].(string))
		}
		if thresholds["warning"] != nil {
			o.Thresholds.Warning = json.Number(thresholds["warning"].(string))
		}
		if thresholds["critical"] != nil {
			o.Thresholds.Critical = json.Number(thresholds["critical"].(string))
		}
	}

	if attr, ok := d.GetOk("notify_no_data"); ok {
		o.NotifyNoData = attr.(bool)
	}
	if attr, ok := d.GetOk("no_data_timeframe"); ok {
		o.NoDataTimeframe = datadog.NoDataTimeframe(attr.(int))
	}
	if attr, ok := d.GetOk("renotify_interval"); ok {
		o.RenotifyInterval = attr.(int)
	}
	if attr, ok := d.GetOk("notify_audit"); ok {
		o.NotifyAudit = attr.(bool)
	}
	if attr, ok := d.GetOk("timeout_h"); ok {
		o.TimeoutH = attr.(int)
	}
	if attr, ok := d.GetOk("escalation_message"); ok {
		o.EscalationMessage = attr.(string)
	}
	if attr, ok := d.GetOk("silenced"); ok {
		// TODO: this is not very defensive, test if we can fail non int input
		s := make(map[string]int)
		for k, v := range attr.(map[string]interface{}) {
			s[k], _ = strconv.Atoi(v.(string))
		}
		o.Silenced = s
	}
	if attr, ok := d.GetOk("include_tags"); ok {
		o.IncludeTags = attr.(bool)
	}
	if attr, ok := d.GetOk("require_full_window"); ok {
		o.RequireFullWindow = attr.(bool)
	}
	if attr, ok := d.GetOk("locked"); ok {
		o.Locked = attr.(bool)
	}

	m.Options = o

	if err = client.UpdateMonitor(m); err != nil {
		return fmt.Errorf("error updating monitor: %s", err.Error())
	}

	return resourceDatadogMonitorRead(d, meta)
}

func resourceDatadogMonitorDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*datadog.Client)

	i, err := strconv.Atoi(d.Id())
	if err != nil {
		return err
	}

	if err = client.DeleteMonitor(i); err != nil {
		return err
	}

	return nil
}

func resourceDatadogImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	if err := resourceDatadogMonitorRead(d, meta); err != nil {
		return nil, err
	}
	return []*schema.ResourceData{d}, nil
}

// Ignore any diff that results from the mix of ints or floats returned from the
// DataDog API.
func supressDataDogFloatIntDiff(k, old, new string, d *schema.ResourceData) bool {
	oF, err := strconv.ParseFloat(old, 64)
	if err != nil {
		log.Printf("Error parsing float of old value (%s): %s", old, err)
		return false
	}

	nF, err := strconv.ParseFloat(new, 64)
	if err != nil {
		log.Printf("Error parsing float of new value (%s): %s", new, err)
		return false
	}

	// if the float values of these attributes are equivalent, ignore this
	// diff
	if oF == nF {
		return true
	}
	return false
}
