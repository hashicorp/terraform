package librato

import (
	"bytes"
	"fmt"
	"log"
	"math"
	"strconv"
	"time"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/henrikhodne/go-librato/librato"
)

func resourceLibratoAlert() *schema.Resource {
	return &schema.Resource{
		Create: resourceLibratoAlertCreate,
		Read:   resourceLibratoAlertRead,
		Update: resourceLibratoAlertUpdate,
		Delete: resourceLibratoAlertDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},
			"id": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"active": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"rearm_seconds": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  600,
			},
			"services": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"condition": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"metric_name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"source": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"detect_reset": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
						},
						"duration": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},
						"threshold": &schema.Schema{
							Type:     schema.TypeFloat,
							Optional: true,
						},
						"summary_function": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
				Set: resourceLibratoAlertConditionsHash,
			},
			"attributes": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"runbook_url": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
		},
	}
}

func resourceLibratoAlertConditionsHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["type"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["metric_name"].(string)))

	source, present := m["source"]
	if present {
		buf.WriteString(fmt.Sprintf("%s-", source.(string)))
	}

	detect_reset, present := m["detect_reset"]
	if present {
		buf.WriteString(fmt.Sprintf("%t-", detect_reset.(bool)))
	}

	duration, present := m["duration"]
	if present {
		buf.WriteString(fmt.Sprintf("%d-", duration.(int)))
	}

	threshold, present := m["threshold"]
	if present {
		buf.WriteString(fmt.Sprintf("%f-", threshold.(float64)))
	}

	summary_function, present := m["summary_function"]
	if present {
		buf.WriteString(fmt.Sprintf("%s-", summary_function.(string)))
	}

	return hashcode.String(buf.String())
}

func resourceLibratoAlertCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*librato.Client)

	alert := new(librato.Alert)
	if v, ok := d.GetOk("name"); ok {
		alert.Name = librato.String(v.(string))
	}
	if v, ok := d.GetOk("description"); ok {
		alert.Description = librato.String(v.(string))
	}
	// GetOK returns not OK for false boolean values, use Get
	alert.Active = librato.Bool(d.Get("active").(bool))
	if v, ok := d.GetOk("rearm_seconds"); ok {
		alert.RearmSeconds = librato.Uint(uint(v.(int)))
	}
	if v, ok := d.GetOk("services"); ok {
		vs := v.(*schema.Set)
		services := make([]*string, vs.Len())
		for i, serviceData := range vs.List() {
			services[i] = librato.String(serviceData.(string))
		}
		alert.Services = services
	}
	if v, ok := d.GetOk("condition"); ok {
		vs := v.(*schema.Set)
		conditions := make([]librato.AlertCondition, vs.Len())
		for i, conditionDataM := range vs.List() {
			conditionData := conditionDataM.(map[string]interface{})
			var condition librato.AlertCondition
			if v, ok := conditionData["type"].(string); ok && v != "" {
				condition.Type = librato.String(v)
			}
			if v, ok := conditionData["threshold"].(float64); ok && !math.IsNaN(v) {
				condition.Threshold = librato.Float(v)
			}
			if v, ok := conditionData["metric_name"].(string); ok && v != "" {
				condition.MetricName = librato.String(v)
			}
			if v, ok := conditionData["source"].(string); ok && v != "" {
				condition.Source = librato.String(v)
			}
			if v, ok := conditionData["detect_reset"].(bool); ok {
				condition.DetectReset = librato.Bool(v)
			}
			if v, ok := conditionData["duration"].(int); ok {
				condition.Duration = librato.Uint(uint(v))
			}
			if v, ok := conditionData["summary_function"].(string); ok && v != "" {
				condition.SummaryFunction = librato.String(v)
			}
			conditions[i] = condition
		}
		alert.Conditions = conditions
	}
	if v, ok := d.GetOk("attributes"); ok {
		attributeData := v.([]interface{})
		if len(attributeData) > 1 {
			return fmt.Errorf("Only one set of attributes per alert is supported")
		} else if len(attributeData) == 1 {
			if attributeData[0] == nil {
				return fmt.Errorf("No attributes found in attributes block")
			}
			attributeDataMap := attributeData[0].(map[string]interface{})
			attributes := new(librato.AlertAttributes)
			if v, ok := attributeDataMap["runbook_url"].(string); ok && v != "" {
				attributes.RunbookURL = librato.String(v)
			}
			alert.Attributes = attributes
		}
	}

	alertResult, _, err := client.Alerts.Create(alert)

	if err != nil {
		return fmt.Errorf("Error creating Librato alert %s: %s", *alert.Name, err)
	}

	resource.Retry(1*time.Minute, func() *resource.RetryError {
		_, _, err := client.Alerts.Get(*alertResult.ID)
		if err != nil {
			if errResp, ok := err.(*librato.ErrorResponse); ok && errResp.Response.StatusCode == 404 {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})

	return resourceLibratoAlertReadResult(d, alertResult)
}

func resourceLibratoAlertRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*librato.Client)
	id, err := strconv.ParseUint(d.Id(), 10, 0)
	if err != nil {
		return err
	}

	alert, _, err := client.Alerts.Get(uint(id))
	if err != nil {
		if errResp, ok := err.(*librato.ErrorResponse); ok && errResp.Response.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading Librato Alert %s: %s", d.Id(), err)
	}

	return resourceLibratoAlertReadResult(d, alert)
}

func resourceLibratoAlertReadResult(d *schema.ResourceData, alert *librato.Alert) error {
	d.SetId(strconv.FormatUint(uint64(*alert.ID), 10))
	d.Set("id", *alert.ID)
	d.Set("name", *alert.Name)
	d.Set("description", *alert.Description)
	d.Set("active", *alert.Active)
	d.Set("rearm_seconds", *alert.RearmSeconds)

	services := resourceLibratoAlertServicesGather(d, alert.Services.([]interface{}))
	d.Set("services", services)

	conditions := resourceLibratoAlertConditionsGather(d, alert.Conditions)
	d.Set("condition", conditions)

	attributes := resourceLibratoAlertAttributesGather(d, alert.Attributes)
	d.Set("attributes", attributes)

	return nil
}

func resourceLibratoAlertServicesGather(d *schema.ResourceData, services []interface{}) []string {
	retServices := make([]string, 0, len(services))

	for _, s := range services {
		serviceData := s.(map[string]interface{})
		// ID field is returned as float64, for whatever reason
		retServices = append(retServices, fmt.Sprintf("%.f", serviceData["id"]))
	}

	return retServices
}

func resourceLibratoAlertConditionsGather(d *schema.ResourceData, conditions []librato.AlertCondition) []map[string]interface{} {
	retConditions := make([]map[string]interface{}, 0, len(conditions))
	for _, c := range conditions {
		condition := make(map[string]interface{})
		if c.Type != nil {
			condition["type"] = *c.Type
		}
		if c.Threshold != nil {
			condition["threshold"] = *c.Threshold
		}
		if c.MetricName != nil {
			condition["metric_name"] = *c.MetricName
		}
		if c.Source != nil {
			condition["source"] = *c.Source
		}
		if c.DetectReset != nil {
			condition["detect_reset"] = *c.MetricName
		}
		if c.Duration != nil {
			condition["duration"] = *c.Duration
		}
		if c.SummaryFunction != nil {
			condition["summary_function"] = *c.SummaryFunction
		}
		retConditions = append(retConditions, condition)
	}

	return retConditions
}

// Flattens an attributes hash into something that flatmap.Flatten() can handle
func resourceLibratoAlertAttributesGather(d *schema.ResourceData, attributes *librato.AlertAttributes) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, 1)

	if attributes != nil {
		retAttributes := make(map[string]interface{})
		if attributes.RunbookURL != nil {
			retAttributes["runbook_url"] = *attributes.RunbookURL
		}
		result = append(result, retAttributes)
	}

	return result
}

func resourceLibratoAlertUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*librato.Client)

	alertID, err := strconv.ParseUint(d.Id(), 10, 0)
	if err != nil {
		return err
	}

	alert := new(librato.Alert)
	alert.Name = librato.String(d.Get("name").(string))
	if d.HasChange("description") {
		alert.Description = librato.String(d.Get("description").(string))
	}
	if d.HasChange("active") {
		alert.Active = librato.Bool(d.Get("active").(bool))
	}
	if d.HasChange("rearm_seconds") {
		alert.RearmSeconds = librato.Uint(uint(d.Get("rearm_seconds").(int)))
	}
	if d.HasChange("services") {
		vs := d.Get("services").(*schema.Set)
		services := make([]*string, vs.Len())
		for i, serviceData := range vs.List() {
			services[i] = librato.String(serviceData.(string))
		}
		alert.Services = services
	}

	vs := d.Get("condition").(*schema.Set)
	conditions := make([]librato.AlertCondition, vs.Len())
	for i, conditionDataM := range vs.List() {
		conditionData := conditionDataM.(map[string]interface{})
		var condition librato.AlertCondition
		if v, ok := conditionData["type"].(string); ok && v != "" {
			condition.Type = librato.String(v)
		}
		if v, ok := conditionData["threshold"].(float64); ok && !math.IsNaN(v) {
			condition.Threshold = librato.Float(v)
		}
		if v, ok := conditionData["metric_name"].(string); ok && v != "" {
			condition.MetricName = librato.String(v)
		}
		if v, ok := conditionData["source"].(string); ok && v != "" {
			condition.Source = librato.String(v)
		}
		if v, ok := conditionData["detect_reset"].(bool); ok {
			condition.DetectReset = librato.Bool(v)
		}
		if v, ok := conditionData["duration"].(int); ok {
			condition.Duration = librato.Uint(uint(v))
		}
		if v, ok := conditionData["summary_function"].(string); ok && v != "" {
			condition.SummaryFunction = librato.String(v)
		}
		conditions[i] = condition
		alert.Conditions = conditions
	}
	if d.HasChange("attributes") {
		attributeData := d.Get("attributes").([]interface{})
		if len(attributeData) > 1 {
			return fmt.Errorf("Only one set of attributes per alert is supported")
		} else if len(attributeData) == 1 {
			if attributeData[0] == nil {
				return fmt.Errorf("No attributes found in attributes block")
			}
			attributeDataMap := attributeData[0].(map[string]interface{})
			attributes := new(librato.AlertAttributes)
			if v, ok := attributeDataMap["runbook_url"].(string); ok && v != "" {
				attributes.RunbookURL = librato.String(v)
			}
			alert.Attributes = attributes
		}
	}

	_, err = client.Alerts.Edit(uint(alertID), alert)
	if err != nil {
		return fmt.Errorf("Error updating Librato alert: %s", err)
	}

	return resourceLibratoAlertRead(d, meta)
}

func resourceLibratoAlertDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*librato.Client)
	id, err := strconv.ParseUint(d.Id(), 10, 0)
	if err != nil {
		return err
	}

	log.Printf("[INFO] Deleting Alert: %d", id)
	_, err = client.Alerts.Delete(uint(id))
	if err != nil {
		return fmt.Errorf("Error deleting Alert: %s", err)
	}

	resource.Retry(1*time.Minute, func() *resource.RetryError {
		_, _, err := client.Alerts.Get(uint(id))
		if err != nil {
			if errResp, ok := err.(*librato.ErrorResponse); ok && errResp.Response.StatusCode == 404 {
				return nil
			}
			return resource.NonRetryableError(err)
		}
		return resource.RetryableError(fmt.Errorf("alert still exists"))
	})

	d.SetId("")
	return nil
}
