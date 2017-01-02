package datadog

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/zorkian/go-datadog-api"
)

func resourceDatadogDowntime() *schema.Resource {
	return &schema.Resource{
		Create: resourceDatadogDowntimeCreate,
		Read:   resourceDatadogDowntimeRead,
		Update: resourceDatadogDowntimeUpdate,
		Delete: resourceDatadogDowntimeDelete,
		Exists: resourceDatadogDowntimeExists,
		Importer: &schema.ResourceImporter{
			State: resourceDatadogDowntimeImport,
		},

		Schema: map[string]*schema.Schema{
			"active": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"disabled": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"end": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"message": {
				Type:     schema.TypeString,
				Optional: true,
				StateFunc: func(val interface{}) string {
					return strings.TrimSpace(val.(string))
				},
			},
			"recurrence": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"period": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"type": {
							Type:     schema.TypeString,
							Required: true,
						},
						"until_date": {
							Type:          schema.TypeInt,
							Optional:      true,
							ConflictsWith: []string{"recurrence.until_occurrences"},
						},
						"until_occurrences": {
							Type:          schema.TypeInt,
							Optional:      true,
							ConflictsWith: []string{"recurrence.until_date"},
						},
						"week_days": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Schema{
								Type:         schema.TypeString,
								ValidateFunc: validateWeekDay,
							},
						},
					},
				},
			},
			"scope": {
				Type:     schema.TypeList,
				Required: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"start": {
				Type:     schema.TypeInt,
				Optional: true,
			},
		},
	}
}

func buildDowntimeStruct(d *schema.ResourceData) *datadog.Downtime {
	scope := []string{}
	for _, s := range d.Get("scope").([]interface{}) {
		scope = append(scope, s.(string))
	}

	dt := datadog.Downtime{
		Scope: scope,
	}

	if attr, ok := d.GetOk("recurrence"); ok {
		r := attr.(map[string]interface{})
		period, _ := strconv.Atoi(r["period"].(string))

		dt.Recurrence = &datadog.Recurrence{
			Period: period,
			Type:   r["type"].(string),
		}

		if r["until_date"] != nil {
			dt.Recurrence.UntilDate, _ = strconv.Atoi(r["until_date"].(string))
		}
		if r["until_occurrences"] != nil {
			dt.Recurrence.UntilOccurrences, _ = strconv.Atoi(r["until_occurrences"].(string))
		}
		if r["week_days"] != nil {
			weekDays := []string{}
			fmt.Printf("%v\n", r["week_days"])
			for _, s := range r["week_days"].([]interface{}) {
				weekDays = append(weekDays, s.(string))
			}
			dt.Recurrence.WeekDays = weekDays
		}
	}

	if attr, ok := d.GetOk("active"); ok {
		dt.Active = attr.(bool)
	}
	if attr, ok := d.GetOk("disabled"); ok {
		dt.Disabled = attr.(bool)
	}
	if attr, ok := d.GetOk("end"); ok {
		dt.End = attr.(int)
	}
	if attr, ok := d.GetOk("message"); ok {
		dt.Message = strings.TrimSpace(attr.(string))
	}
	if attr, ok := d.GetOk("start"); ok {
		dt.Start = attr.(int)
	}

	return &dt
}

func resourceDatadogDowntimeExists(d *schema.ResourceData, meta interface{}) (b bool, e error) {
	// Exists - This is called to verify a resource still exists. It is called prior to Read,
	// and lowers the burden of Read to be able to assume the resource exists.
	client := meta.(*datadog.Client)

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return false, err
	}

	if _, err = client.GetDowntime(id); err != nil {
		if strings.Contains(err.Error(), "404 Not Found") {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func resourceDatadogDowntimeCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*datadog.Client)

	dts := buildDowntimeStruct(d)
	dt, err := client.CreateDowntime(dts)
	if err != nil {
		return fmt.Errorf("error updating downtime: %s", err.Error())
	}

	d.SetId(strconv.Itoa(dt.Id))

	return nil
}

func resourceDatadogDowntimeRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*datadog.Client)

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return err
	}

	dt, err := client.GetDowntime(id)
	if err != nil {
		return err
	}

	recurrence := make(map[string]interface{})
	if dt.Recurrence != nil {
		recurrence["period"] = dt.Recurrence.Period
		recurrence["type"] = dt.Recurrence.Type
		recurrence["until_date"] = dt.Recurrence.UntilDate
		recurrence["until_occurrences"] = dt.Recurrence.UntilOccurrences
		recurrence["week_days"] = dt.Recurrence.WeekDays
	}

	log.Printf("[DEBUG] downtime: %v", dt)
	d.Set("active", dt.Active)
	d.Set("disabled", dt.Disabled)
	d.Set("end", dt.End)
	d.Set("message", dt.Message)
	d.Set("recurrence", recurrence)
	d.Set("scope", dt.Scope)
	d.Set("start", dt.Start)

	return nil
}

func resourceDatadogDowntimeUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*datadog.Client)

	dt := &datadog.Downtime{}

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return err
	}

	dt.Id = id
	if attr, ok := d.GetOk("active"); ok {
		dt.Active = attr.(bool)
	}
	if attr, ok := d.GetOk("disabled"); ok {
		dt.Disabled = attr.(bool)
	}
	if attr, ok := d.GetOk("end"); ok {
		dt.End = attr.(int)
	}
	if attr, ok := d.GetOk("message"); ok {
		dt.Message = attr.(string)
	}
	recurrence := datadog.Recurrence{}
	if attr, ok := d.GetOk("recurrence"); ok {
		r := attr.(map[string]interface{})
		recurrence.Period, _ = strconv.Atoi(r["period"].(string))
		recurrence.Type = r["type"].(string)
		if r["until_date"] != nil {
			recurrence.UntilDate, _ = strconv.Atoi(r["until_date"].(string))
		}
		if r["until_occurrences"] != nil {
			recurrence.UntilOccurrences, _ = strconv.Atoi(r["until_occurrences"].(string))
		}
		if r["week_days"] != nil {
			weekDays := []string{}
			for _, s := range r["week_days"].([]interface{}) {
				weekDays = append(weekDays, s.(string))
			}
			recurrence.WeekDays = weekDays
			//recurrence.WeekDays = r["week_days"].([]string)
		}
	}
	dt.Recurrence = &recurrence
	scope := make([]string, 0)
	for _, v := range d.Get("scope").([]interface{}) {
		scope = append(scope, v.(string))
	}
	dt.Scope = scope
	if attr, ok := d.GetOk("start"); ok {
		dt.Start = attr.(int)
	}

	if err = client.UpdateDowntime(dt); err != nil {
		return fmt.Errorf("error updating downtime: %s", err.Error())
	}

	return resourceDatadogDowntimeRead(d, meta)
}

func resourceDatadogDowntimeDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*datadog.Client)

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return err
	}

	if err = client.DeleteDowntime(id); err != nil {
		return err
	}

	return nil
}

func resourceDatadogDowntimeImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	if err := resourceDatadogDowntimeRead(d, meta); err != nil {
		return nil, err
	}
	return []*schema.ResourceData{d}, nil
}

// validateWeekDay ensures that the week_days resource parameter is
// correct.
func validateWeekDay(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)

	if value != "Mon" && value != "Tue" && value != "Wed" && value != "Thu" && value != "Fri" && value != "Sat" && value != "Sun" {
		errors = append(errors, fmt.Errorf(
			"%q contains an invalid week day parameter %q. Valid parameters are %q, %q, %q, %q, %q, %q, or %q",
			k, value, "Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"))
	}
	return
}
