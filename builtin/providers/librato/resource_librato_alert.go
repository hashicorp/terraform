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
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"active": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"rearm_seconds": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  600,
			},
			"services": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"condition": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:     schema.TypeString,
							Required: true,
						},
						"metric_name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"source": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"detect_reset": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"duration": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"threshold": {
							Type:     schema.TypeFloat,
							Optional: true,
						},
						"summary_function": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
				Set: resourceLibratoAlertConditionsHash,
			},
			"attributes": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"runbook_url": {
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

	detectReset, present := m["detect_reset"]
	if present {
		buf.WriteString(fmt.Sprintf("%t-", detectReset.(bool)))
	}

	duration, present := m["duration"]
	if present {
		buf.WriteString(fmt.Sprintf("%d-", duration.(int)))
	}

	threshold, present := m["threshold"]
	if present {
		buf.WriteString(fmt.Sprintf("%f-", threshold.(float64)))
	}

	summaryFunction, present := m["summary_function"]
	if present {
		buf.WriteString(fmt.Sprintf("%s-", summaryFunction.(string)))
	}

	return hashcode.String(buf.String())
}

func resourceLibratoAlertCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*librato.Client)

	alert := librato.Alert{
		Name: librato.String(d.Get("name").(string)),
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

	alertResult, _, err := client.Alerts.Create(&alert)

	if err != nil {
		return fmt.Errorf("Error creating Librato alert %s: %s", *alert.Name, err)
	}
	log.Printf("[INFO] Created Librato alert: %s", *alertResult)

	retryErr := resource.Retry(1*time.Minute, func() *resource.RetryError {
		_, _, err := client.Alerts.Get(*alertResult.ID)
		if err != nil {
			if errResp, ok := err.(*librato.ErrorResponse); ok && errResp.Response.StatusCode == 404 {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})
	if retryErr != nil {
		return fmt.Errorf("Error creating librato alert: %s", err)
	}

	d.SetId(strconv.FormatUint(uint64(*alertResult.ID), 10))

	return resourceLibratoAlertRead(d, meta)
}

func resourceLibratoAlertRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*librato.Client)
	id, err := strconv.ParseUint(d.Id(), 10, 0)
	if err != nil {
		return err
	}

	log.Printf("[INFO] Reading Librato Alert: %d", id)
	alert, _, err := client.Alerts.Get(uint(id))
	if err != nil {
		if errResp, ok := err.(*librato.ErrorResponse); ok && errResp.Response.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading Librato Alert %s: %s", d.Id(), err)
	}
	log.Printf("[INFO] Received Librato Alert: %s", *alert)

	d.Set("name", alert.Name)

	if alert.Description != nil {
		if err := d.Set("description", alert.Description); err != nil {
			return err
		}
	}
	if alert.Active != nil {
		if err := d.Set("active", alert.Active); err != nil {
			return err
		}
	}
	if alert.RearmSeconds != nil {
		if err := d.Set("rearm_seconds", alert.RearmSeconds); err != nil {
			return err
		}
	}

	// Since the following aren't simple terraform types (TypeList), it's best to
	// catch the error returned from the d.Set() function, and handle accordingly.
	services := resourceLibratoAlertServicesGather(d, alert.Services.([]interface{}))
	if err := d.Set("services", schema.NewSet(schema.HashString, services)); err != nil {
		return err
	}

	conditions := resourceLibratoAlertConditionsGather(d, alert.Conditions)
	if err := d.Set("condition", schema.NewSet(resourceLibratoAlertConditionsHash, conditions)); err != nil {
		return err
	}

	attributes := resourceLibratoAlertAttributesGather(d, alert.Attributes)
	if err := d.Set("attributes", attributes); err != nil {
		return err
	}

	return nil
}

func resourceLibratoAlertServicesGather(d *schema.ResourceData, services []interface{}) []interface{} {
	retServices := make([]interface{}, 0, len(services))

	for _, s := range services {
		serviceData := s.(map[string]interface{})
		// ID field is returned as float64, for whatever reason
		retServices = append(retServices, fmt.Sprintf("%.f", serviceData["id"]))
	}

	return retServices
}

func resourceLibratoAlertConditionsGather(d *schema.ResourceData, conditions []librato.AlertCondition) []interface{} {
	retConditions := make([]interface{}, 0, len(conditions))
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
			condition["duration"] = int(*c.Duration)
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

	id, err := strconv.ParseUint(d.Id(), 10, 0)
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

	log.Printf("[INFO] Updating Librato alert: %s", alert)
	_, updErr := client.Alerts.Update(uint(id), alert)
	if updErr != nil {
		return fmt.Errorf("Error updating Librato alert: %s", updErr)
	}

	log.Printf("[INFO] Updated Librato alert %d", id)

	// Wait for propagation since Librato updates are eventually consistent
	wait := resource.StateChangeConf{
		Pending:                   []string{fmt.Sprintf("%t", false)},
		Target:                    []string{fmt.Sprintf("%t", true)},
		Timeout:                   5 * time.Minute,
		MinTimeout:                2 * time.Second,
		ContinuousTargetOccurence: 5,
		Refresh: func() (interface{}, string, error) {
			log.Printf("[DEBUG] Checking if Librato Alert %d was updated yet", id)
			changedAlert, _, getErr := client.Alerts.Get(uint(id))
			if getErr != nil {
				return changedAlert, "", getErr
			}
			return changedAlert, "true", nil
		},
	}

	_, err = wait.WaitForState()
	if err != nil {
		return fmt.Errorf("Failed updating Librato Alert %d: %s", id, err)
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

	retryErr := resource.Retry(1*time.Minute, func() *resource.RetryError {
		_, _, err := client.Alerts.Get(uint(id))
		if err != nil {
			if errResp, ok := err.(*librato.ErrorResponse); ok && errResp.Response.StatusCode == 404 {
				return nil
			}
			return resource.NonRetryableError(err)
		}
		return resource.RetryableError(fmt.Errorf("alert still exists"))
	})
	if retryErr != nil {
		return fmt.Errorf("Error deleting librato alert: %s", err)
	}

	return nil
}
