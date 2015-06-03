package datadog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
)

const (
	monitorEndpoint = "https://app.datadoghq.com/api/v1/monitor"
)

func datadogMonitorResource() *schema.Resource {
	return &schema.Resource{
		Create: resourceMonitorCreate,
		Read:   resourceMonitorRead,
		Update: resourceMonitorUpdate,
		Delete: resourceMonitorDelete,
		Exists: resourceMonitorExists,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			// Metric and Monitor settings
			"metric": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"metric_tags": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "*",
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

			// Alert Settings
			"warning": &schema.Schema{
				Type:     schema.TypeMap,
				Required: true,
			},
			"critical": &schema.Schema{
				Type:     schema.TypeMap,
				Required: true,
			},

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
		},
	}
}

func getIDFromResponse(h *http.Response) (string, error) {
	body, err := ioutil.ReadAll(h.Body)
	if err != nil {
		return "", err
	}
	h.Body.Close()
	log.Println(h)
	log.Println(string(body))
	v := map[string]interface{}{}
	err = json.Unmarshal(body, &v)
	if err != nil {
		return "", err
	}
	if id, ok := v["id"]; ok {
		return strconv.Itoa(int(id.(float64))), nil
	}
	return "", fmt.Errorf("error getting ID from response %s", h.Status)
}

func marshalMetric(d *schema.ResourceData, typeStr string) ([]byte, error) {
	name := d.Get("name").(string)
	message := d.Get("message").(string)
	timeAggr := d.Get("time_aggr").(string)
	timeWindow := d.Get("time_window").(string)
	spaceAggr := d.Get("space_aggr").(string)
	metric := d.Get("metric").(string)
	tags := d.Get("metric_tags").(string)
	operator := d.Get("operator").(string)
	query := fmt.Sprintf("%s(%s):%s:%s{%s} %s %s", timeAggr, timeWindow, spaceAggr, metric, tags, operator, d.Get(fmt.Sprintf("%s.threshold", typeStr)))

	log.Println(query)
	m := map[string]interface{}{
		"type":    "metric alert",
		"query":   query,
		"name":    fmt.Sprintf("[%s] %s", typeStr, name),
		"message": fmt.Sprintf("%s %s", message, d.Get(fmt.Sprintf("%s.notify", typeStr))),
		"options": map[string]interface{}{
			"notify_no_data":    d.Get("notify_no_data").(bool),
			"no_data_timeframe": d.Get("no_data_timeframe").(int),
		},
	}
	return json.Marshal(m)
}

func authSuffix(meta interface{}) string {
	m := meta.(map[string]string)
	return fmt.Sprintf("?api_key=%s&application_key=%s", m["api_key"], m["app_key"])
}

func resourceMonitorCreate(d *schema.ResourceData, meta interface{}) error {
	warningBody, err := marshalMetric(d, "warning")
	if err != nil {
		return err
	}
	criticalBody, err := marshalMetric(d, "critical")
	if err != nil {
		return err
	}

	resW, err := http.Post(fmt.Sprintf("%s%s", monitorEndpoint, authSuffix(meta)), "application/json", bytes.NewReader(warningBody))
	if err != nil {
		return fmt.Errorf("error creating warning: %s", err.Error())
	}

	resC, err := http.Post(fmt.Sprintf("%s%s", monitorEndpoint, authSuffix(meta)), "application/json", bytes.NewReader(criticalBody))
	if err != nil {
		return fmt.Errorf("error creating critical: %s", err.Error())
	}

	warningMonitorID, err := getIDFromResponse(resW)
	if err != nil {
		return err
	}
	criticalMonitorID, err := getIDFromResponse(resC)
	if err != nil {
		return err
	}

	d.SetId(fmt.Sprintf("%s__%s", warningMonitorID, criticalMonitorID))

	return nil
}

func resourceMonitorDelete(d *schema.ResourceData, meta interface{}) (e error) {
	for _, v := range strings.Split(d.Id(), "__") {
		client := http.Client{}
		req, _ := http.NewRequest("DELETE", fmt.Sprintf("%s/%s%s", monitorEndpoint, v, authSuffix(meta)), nil)
		_, err := client.Do(req)
		e = err
	}
	return
}

func resourceMonitorExists(d *schema.ResourceData, meta interface{}) (b bool, e error) {
	b = true
	for _, v := range strings.Split(d.Id(), "__") {
		res, err := http.Get(fmt.Sprintf("%s/%s%s", monitorEndpoint, v, authSuffix(meta)))
		if err != nil {
			e = err
			continue
		}
		if res.StatusCode > 400 {
			b = false
			continue
		}
		b = b && true
	}
	if !b {
		e = resourceMonitorDelete(d, meta)
	}
	return
}

func resourceMonitorRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceMonitorUpdate(d *schema.ResourceData, meta interface{}) error {
	split := strings.Split(d.Id(), "__")
	warningID, criticalID := split[0], split[1]

	warningBody, _ := marshalMetric(d, "warning")
	criticalBody, _ := marshalMetric(d, "critical")

	client := http.Client{}

	reqW, _ := http.NewRequest("PUT", fmt.Sprintf("%s/%s%s", monitorEndpoint, warningID, authSuffix(meta)), bytes.NewReader(warningBody))
	resW, err := client.Do(reqW)
	if err != nil {
		return fmt.Errorf("error updating warning: %s", err.Error())
	}
	resW.Body.Close()
	if resW.StatusCode > 400 {
		return fmt.Errorf("error updating warning monitor: %s", resW.Status)
	}

	reqC, _ := http.NewRequest("PUT", fmt.Sprintf("%s/%s%s", monitorEndpoint, criticalID, authSuffix(meta)), bytes.NewReader(criticalBody))
	resC, err := client.Do(reqC)
	if err != nil {
		return fmt.Errorf("error updating critical: %s", err.Error())
	}
	resW.Body.Close()
	if resW.StatusCode > 400 {
		return fmt.Errorf("error updating critical monitor: %s", resC.Status)
	}
	return nil
}
