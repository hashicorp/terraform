package datadog

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"gopkg.in/zorkian/go-datadog-api.v2"
)

func resourceDatadogMonitor() *schema.Resource {
	return &schema.Resource{
		Create: resourceDatadogMonitorCreate,
		Read:   resourceDatadogMonitorRead,
		Update: resourceDatadogMonitorUpdate,
		Delete: resourceDatadogMonitorDelete,
		Exists: resourceDatadogMonitorExists,
		Importer: &schema.ResourceImporter{
			State: resourceDatadogMonitorImport,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"message": {
				Type:     schema.TypeString,
				Required: true,
				StateFunc: func(val interface{}) string {
					return strings.TrimSpace(val.(string))
				},
			},
			"escalation_message": {
				Type:     schema.TypeString,
				Optional: true,
				StateFunc: func(val interface{}) string {
					return strings.TrimSpace(val.(string))
				},
			},
			"query": {
				Type:     schema.TypeString,
				Required: true,
				StateFunc: func(val interface{}) string {
					return strings.TrimSpace(val.(string))
				},
			},
			"type": {
				Type:     schema.TypeString,
				Required: true,
			},

			// Options
			"thresholds": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ok": {
							Type:     schema.TypeFloat,
							Optional: true,
						},
						"warning": {
							Type:     schema.TypeFloat,
							Optional: true,
						},
						"critical": {
							Type:     schema.TypeFloat,
							Optional: true,
						},
					},
				},
				DiffSuppressFunc: suppressDataDogFloatIntDiff,
			},
			"notify_no_data": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"new_host_delay": {
				Type:     schema.TypeInt,
				Computed: true,
				Optional: true,
			},
			"evaluation_delay": {
				Type:     schema.TypeInt,
				Computed: true,
				Optional: true,
			},
			"no_data_timeframe": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"renotify_interval": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"notify_audit": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"timeout_h": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"require_full_window": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"locked": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"silenced": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem:     schema.TypeInt,
			},
			"include_tags": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"tags": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func buildMonitorStruct(d *schema.ResourceData) *datadog.Monitor {

	var thresholds datadog.ThresholdCount

	if r, ok := d.GetOk("thresholds.ok"); ok {
		thresholds.SetOk(json.Number(r.(string)))
	}
	if r, ok := d.GetOk("thresholds.warning"); ok {
		thresholds.SetWarning(json.Number(r.(string)))
	}
	if r, ok := d.GetOk("thresholds.critical"); ok {
		thresholds.SetCritical(json.Number(r.(string)))
	}

	o := datadog.Options{
		Thresholds:        &thresholds,
		NotifyNoData:      datadog.Bool(d.Get("notify_no_data").(bool)),
		RequireFullWindow: datadog.Bool(d.Get("require_full_window").(bool)),
		IncludeTags:       datadog.Bool(d.Get("include_tags").(bool)),
	}
	if attr, ok := d.GetOk("silenced"); ok {
		s := make(map[string]int)
		// TODO: this is not very defensive, test if we can fail on non int input
		for k, v := range attr.(map[string]interface{}) {
			s[k] = v.(int)
		}
		o.Silenced = s
	}
	if attr, ok := d.GetOk("notify_no_data"); ok {
		o.SetNotifyNoData(attr.(bool))
	}
	if attr, ok := d.GetOk("new_host_delay"); ok {
		o.SetNewHostDelay(attr.(int))
	}
	if attr, ok := d.GetOk("evaluation_delay"); ok {
		o.SetEvaluationDelay(attr.(int))
	}
	if attr, ok := d.GetOk("no_data_timeframe"); ok {
		o.NoDataTimeframe = datadog.NoDataTimeframe(attr.(int))
	}
	if attr, ok := d.GetOk("renotify_interval"); ok {
		o.SetRenotifyInterval(attr.(int))
	}
	if attr, ok := d.GetOk("notify_audit"); ok {
		o.SetNotifyAudit(attr.(bool))
	}
	if attr, ok := d.GetOk("timeout_h"); ok {
		o.SetTimeoutH(attr.(int))
	}
	if attr, ok := d.GetOk("escalation_message"); ok {
		o.SetEscalationMessage(attr.(string))
	}
	if attr, ok := d.GetOk("locked"); ok {
		o.SetLocked(attr.(bool))
	}

	m := datadog.Monitor{
		Type:    datadog.String(d.Get("type").(string)),
		Query:   datadog.String(d.Get("query").(string)),
		Name:    datadog.String(d.Get("name").(string)),
		Message: datadog.String(d.Get("message").(string)),
		Options: &o,
	}

	if attr, ok := d.GetOk("tags"); ok {
		tags := []string{}
		for _, s := range attr.([]interface{}) {
			tags = append(tags, s.(string))
		}
		m.Tags = tags
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
		return fmt.Errorf("error updating monitor: %s", err.Error())
	}

	d.SetId(strconv.Itoa(m.GetId()))

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
		"ok":       m.Options.Thresholds.GetOk(),
		"warning":  m.Options.Thresholds.GetWarning(),
		"critical": m.Options.Thresholds.GetCritical(),
	} {
		s := v.String()
		if s != "" {
			thresholds[k] = s
		}
	}

	tags := []string{}
	for _, s := range m.Tags {
		tags = append(tags, s)
	}

	log.Printf("[DEBUG] monitor: %v", m)
	d.Set("name", m.GetName())
	d.Set("message", m.GetMessage())
	d.Set("query", m.GetQuery())
	d.Set("type", m.GetType())
	d.Set("thresholds", thresholds)

	d.Set("new_host_delay", m.Options.GetNewHostDelay())
	d.Set("evaluation_delay", m.Options.GetEvaluationDelay())
	d.Set("notify_no_data", m.Options.GetNotifyNoData())
	d.Set("no_data_timeframe", m.Options.NoDataTimeframe)
	d.Set("renotify_interval", m.Options.GetRenotifyInterval())
	d.Set("notify_audit", m.Options.GetNotifyAudit())
	d.Set("timeout_h", m.Options.GetTimeoutH())
	d.Set("escalation_message", m.Options.GetEscalationMessage())
	d.Set("silenced", m.Options.Silenced)
	d.Set("include_tags", m.Options.GetIncludeTags())
	d.Set("tags", tags)
	d.Set("require_full_window", m.Options.GetRequireFullWindow()) // TODO Is this one of those options that we neeed to check?
	d.Set("locked", m.Options.GetLocked())

	return nil
}

func resourceDatadogMonitorUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*datadog.Client)

	m := &datadog.Monitor{}

	i, err := strconv.Atoi(d.Id())
	if err != nil {
		return err
	}

	m.Id = datadog.Int(i)
	if attr, ok := d.GetOk("name"); ok {
		m.SetName(attr.(string))
	}
	if attr, ok := d.GetOk("message"); ok {
		m.SetMessage(attr.(string))
	}
	if attr, ok := d.GetOk("query"); ok {
		m.SetQuery(attr.(string))
	}

	if attr, ok := d.GetOk("tags"); ok {
		s := make([]string, 0)
		for _, v := range attr.([]interface{}) {
			s = append(s, v.(string))
		}
		m.Tags = s
	}

	o := datadog.Options{
		NotifyNoData:      datadog.Bool(d.Get("notify_no_data").(bool)),
		RequireFullWindow: datadog.Bool(d.Get("require_full_window").(bool)),
		IncludeTags:       datadog.Bool(d.Get("include_tags").(bool)),
	}
	if attr, ok := d.GetOk("thresholds"); ok {
		thresholds := attr.(map[string]interface{})
		o.Thresholds = &datadog.ThresholdCount{} // TODO: This is a little annoying..
		if thresholds["ok"] != nil {
			o.Thresholds.SetOk(json.Number(thresholds["ok"].(string)))
		}
		if thresholds["warning"] != nil {
			o.Thresholds.SetWarning(json.Number(thresholds["warning"].(string)))
		}
		if thresholds["critical"] != nil {
			o.Thresholds.SetCritical(json.Number(thresholds["critical"].(string)))
		}
	}

	if attr, ok := d.GetOk("new_host_delay"); ok {
		o.SetNewHostDelay(attr.(int))
	}
	if attr, ok := d.GetOk("evaluation_delay"); ok {
		o.SetEvaluationDelay(attr.(int))
	}
	if attr, ok := d.GetOk("no_data_timeframe"); ok {
		o.NoDataTimeframe = datadog.NoDataTimeframe(attr.(int))
	}
	if attr, ok := d.GetOk("renotify_interval"); ok {
		o.SetRenotifyInterval(attr.(int))
	}
	if attr, ok := d.GetOk("notify_audit"); ok {
		o.SetNotifyAudit(attr.(bool))
	}
	if attr, ok := d.GetOk("timeout_h"); ok {
		o.SetTimeoutH(attr.(int))
	}
	if attr, ok := d.GetOk("escalation_message"); ok {
		o.SetEscalationMessage(attr.(string))
	}
	if attr, ok := d.GetOk("silenced"); ok {
		// TODO: this is not very defensive, test if we can fail non int input
		s := make(map[string]int)
		for k, v := range attr.(map[string]interface{}) {
			s[k] = v.(int)
		}
		o.Silenced = s
	}
	if attr, ok := d.GetOk("locked"); ok {
		o.SetLocked(attr.(bool))
	}

	m.Options = &o

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

func resourceDatadogMonitorImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	if err := resourceDatadogMonitorRead(d, meta); err != nil {
		return nil, err
	}
	return []*schema.ResourceData{d}, nil
}

// Ignore any diff that results from the mix of ints or floats returned from the
// DataDog API.
func suppressDataDogFloatIntDiff(k, old, new string, d *schema.ResourceData) bool {
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
