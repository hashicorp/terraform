package datadog

import (
	"bytes"
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/zorkian/go-datadog-api"
)

// resourceDatadogMetricAlert is a Datadog monitor resource
func resourceDatadogMetricAlert() *schema.Resource {
	return &schema.Resource{
		Create: resourceDatadogMetricAlertCreate,
		Read:   resourceDatadogGenericRead,
		Update: resourceDatadogMetricAlertUpdate,
		Delete: resourceDatadogGenericDelete,
		Exists: resourceDatadogGenericExists,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"metric": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"tags": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"keys": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"time_aggr": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"time_window": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"space_aggr": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"operator": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"message": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"thresholds": thresholdSchema(),

			// Additional Settings
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
				Default:  0,
			},
		},
	}
}

// buildMonitorStruct returns a monitor struct
func buildMetricAlertStruct(d *schema.ResourceData) *datadog.Monitor {
	name := d.Get("name").(string)
	message := d.Get("message").(string)
	timeAggr := d.Get("time_aggr").(string)
	timeWindow := d.Get("time_window").(string)
	spaceAggr := d.Get("space_aggr").(string)
	metric := d.Get("metric").(string)

	// Tags are are no separate resource/gettable, so some trickery is needed
	var buffer bytes.Buffer
	if raw, ok := d.GetOk("tags"); ok {
		list := raw.([]interface{})
		length := (len(list) - 1)
		for i, v := range list {
			buffer.WriteString(fmt.Sprintf("%s", v))
			if i != length {
				buffer.WriteString(",")
			}

		}
	}

	tagsParsed := buffer.String()

	// Keys are used for multi alerts
	var b bytes.Buffer
	if raw, ok := d.GetOk("keys"); ok {
		list := raw.([]interface{})
		b.WriteString("by {")
		length := (len(list) - 1)
		for i, v := range list {
			b.WriteString(fmt.Sprintf("%s", v))
			if i != length {
				b.WriteString(",")
			}

		}
		b.WriteString("}")
	}

	keys := b.String()

	threshold, thresholds := getThresholds(d)

	operator := d.Get("operator").(string)
	query := fmt.Sprintf("%s(%s):%s:%s{%s} %s %s %s", timeAggr,
		timeWindow,
		spaceAggr,
		metric,
		tagsParsed,
		keys,
		operator,
		threshold)

	log.Print(fmt.Sprintf("[DEBUG] submitting query: %s", query))

	o := datadog.Options{
		NotifyNoData:     d.Get("notify_no_data").(bool),
		NoDataTimeframe:  d.Get("no_data_timeframe").(int),
		RenotifyInterval: d.Get("renotify_interval").(int),
		Thresholds:       thresholds,
	}

	m := datadog.Monitor{
		Type:    "metric alert",
		Query:   query,
		Name:    name,
		Message: message,
		Options: o,
	}

	return &m
}

// resourceDatadogMetricAlertCreate creates a monitor.
func resourceDatadogMetricAlertCreate(d *schema.ResourceData, meta interface{}) error {

	m := buildMetricAlertStruct(d)
	if err := monitorCreator(d, meta, m); err != nil {
		return err
	}

	return nil
}

// resourceDatadogMetricAlertUpdate updates a monitor.
func resourceDatadogMetricAlertUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] running update.")

	m := buildMetricAlertStruct(d)
	if err := monitorUpdater(d, meta, m); err != nil {
		return err
	}

	return nil
}
