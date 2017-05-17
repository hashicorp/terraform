package librato

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/henrikhodne/go-librato/librato"
)

const (
	metricTypeGauge     = "gauge"
	metricTypeCounter   = "counter"
	metricTypeComposite = "composite"
)

var metricTypes = []string{metricTypeGauge, metricTypeCounter, metricTypeComposite}

func resourceLibratoMetric() *schema.Resource {
	return &schema.Resource{
		Create: resourceLibratoMetricCreate,
		Read:   resourceLibratoMetricRead,
		Update: resourceLibratoMetricUpdate,
		Delete: resourceLibratoMetricDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},
			"type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"display_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"period": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"composite": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"attributes": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"color": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"display_max": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"display_min": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"display_units_long": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"display_units_short": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"display_stacked": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
						},
						"created_by_ua": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"gap_detection": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
						},
						"aggregate": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
						},
					},
				},
			},
		},
	}
}

func resourceLibratoMetricCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*librato.Client)

	metric := new(librato.Metric)
	if a, ok := d.GetOk("name"); ok {
		metric.Name = librato.String(a.(string))
	}
	if a, ok := d.GetOk("display_name"); ok {
		metric.DisplayName = librato.String(a.(string))
	}
	if a, ok := d.GetOk("type"); ok {
		metric.Type = librato.String(a.(string))
	}
	if a, ok := d.GetOk("description"); ok {
		metric.Description = librato.String(a.(string))
	}
	if a, ok := d.GetOk("period"); ok {
		metric.Period = librato.Uint(a.(uint))
	}

	if a, ok := d.GetOk("attributes"); ok {
		attributeData := a.([]interface{})
		if len(attributeData) > 1 {
			return fmt.Errorf("Only one set of attributes per alert is supported")
		}

		if len(attributeData) == 1 && attributeData[0] == nil {
			return fmt.Errorf("No attributes found in attributes block")
		}

		attributeDataMap := attributeData[0].(map[string]interface{})
		attributes := new(librato.MetricAttributes)

		if v, ok := attributeDataMap["color"].(string); ok && v != "" {
			attributes.Color = librato.String(v)
		}
		if v, ok := attributeDataMap["display_max"].(string); ok && v != "" {
			attributes.DisplayMax = librato.String(v)
		}
		if v, ok := attributeDataMap["display_min"].(string); ok && v != "" {
			attributes.DisplayMin = librato.String(v)
		}
		if v, ok := attributeDataMap["display_units_long"].(string); ok && v != "" {
			attributes.DisplayUnitsLong = *librato.String(v)
		}
		if v, ok := attributeDataMap["display_units_short"].(string); ok && v != "" {
			attributes.DisplayUnitsShort = *librato.String(v)
		}
		if v, ok := attributeDataMap["created_by_ua"].(string); ok && v != "" {
			attributes.CreatedByUA = *librato.String(v)
		}
		if v, ok := attributeDataMap["display_stacked"].(bool); ok {
			attributes.DisplayStacked = *librato.Bool(v)
		}
		if v, ok := attributeDataMap["gap_detection"].(bool); ok {
			attributes.GapDetection = *librato.Bool(v)
		}
		if v, ok := attributeDataMap["aggregate"].(bool); ok {
			attributes.Aggregate = *librato.Bool(v)
		}

		metric.Attributes = attributes
	}

	_, err := client.Metrics.Edit(metric)
	if err != nil {
		log.Printf("[INFO] ERROR creating Metric: %s", err)
		return fmt.Errorf("Error creating Librato service: %s", err)
	}

	resource.Retry(1*time.Minute, func() *resource.RetryError {
		_, _, err := client.Metrics.Get(*metric.Name)
		if err != nil {
			if errResp, ok := err.(*librato.ErrorResponse); ok && errResp.Response.StatusCode == 404 {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})

	return resourceLibratoMetricReadResult(d, metric)
}

func resourceLibratoMetricReadResult(d *schema.ResourceData, metric *librato.Metric) error {
	d.SetId(*metric.Name)
	d.Set("id", *metric.Name)
	d.Set("name", *metric.Name)
	d.Set("type", *metric.Type)

	if metric.Description != nil {
		d.Set("description", *metric.Description)
	}

	if metric.DisplayName != nil {
		d.Set("display_name", *metric.DisplayName)
	}

	if metric.Period != nil {
		d.Set("period", *metric.Period)
	}

	if metric.Composite != nil {
		d.Set("composite", *metric.Composite)
	}

	attributes := resourceLibratoMetricAttributesGather(d, metric.Attributes)
	d.Set("attributes", attributes)

	return nil
}

// Flattens an attributes hash into something that flatmap.Flatten() can handle
func resourceLibratoMetricAttributesGather(d *schema.ResourceData, attributes *librato.MetricAttributes) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, 1)

	if attributes != nil {
		retAttributes := make(map[string]interface{})
		if attributes.Color != nil {
			retAttributes["color"] = *attributes.Color
		}
		if attributes.DisplayMax != nil {
			retAttributes["display_max"] = attributes.DisplayMax
		}
		if attributes.DisplayMin != nil {
			retAttributes["display_min"] = attributes.DisplayMin
		}
		if attributes.DisplayUnitsLong != "" {
			retAttributes["display_units_long"] = attributes.DisplayUnitsLong
		}
		if attributes.DisplayUnitsShort != "" {
			retAttributes["display_units_short"] = attributes.DisplayUnitsShort
		}
		if attributes.CreatedByUA != "" {
			retAttributes["created_by_ua"] = attributes.CreatedByUA
		}
		retAttributes["display_stacked"] = attributes.DisplayStacked || false
		retAttributes["gap_detection"] = attributes.GapDetection || false
		retAttributes["aggregate"] = attributes.Aggregate || false

		result = append(result, retAttributes)
	}

	return result
}

func resourceLibratoMetricRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*librato.Client)

	id := d.Id()

	log.Printf("[INFO] Reading Librato Metric: %s", id)
	metric, _, err := client.Metrics.Get(id)
	if err != nil {
		if errResp, ok := err.(*librato.ErrorResponse); ok && errResp.Response.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading Librato Metric %s: %s", id, err)
	}

	log.Printf("[INFO] Read Librato Metric: %s", structToString(metric))

	return resourceLibratoMetricReadResult(d, metric)
}

func resourceLibratoMetricUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*librato.Client)

	metricID := d.Id()

	// Just to have whole object for comparison before/after update
	fullMetric, _, err := client.Metrics.Get(metricID)
	if err != nil {
		return err
	}

	metric := new(librato.Metric)
	metric.Name = librato.String(d.Get("name").(string))

	if d.HasChange("type") {
		metric.Type = librato.String(d.Get("type").(string))
		fullMetric.Type = metric.Type
	}

	if d.HasChange("description") {
		metric.Description = librato.String(d.Get("description").(string))
		fullMetric.Description = metric.Description
	}

	if d.HasChange("display_name") {
		metric.DisplayName = librato.String(d.Get("display_name").(string))
		fullMetric.DisplayName = metric.DisplayName
	}

	if d.HasChange("period") {
		metric.Period = librato.Uint(d.Get("period").(uint))
		fullMetric.Period = metric.Period
	}

	if d.HasChange("composite") {
		metric.Composite = librato.String(d.Get("composite").(string))
		fullMetric.Composite = metric.Composite
	}

	if d.HasChange("attributes") {
		attributeData := d.Get("attributes").([]interface{})
		if len(attributeData) > 1 {
			return fmt.Errorf("Only one set of attributes per alert is supported")
		}

		if len(attributeData) == 1 && attributeData[0] == nil {
			return fmt.Errorf("No attributes found in attributes block")
		}

		attributeDataMap := attributeData[0].(map[string]interface{})
		attributes := new(librato.MetricAttributes)

		if v, ok := attributeDataMap["color"].(string); ok && v != "" {
			attributes.Color = librato.String(v)
		}
		if v, ok := attributeDataMap["display_max"].(string); ok && v != "" {
			attributes.DisplayMax = librato.String(v)
		}
		if v, ok := attributeDataMap["display_min"].(string); ok && v != "" {
			attributes.DisplayMin = librato.String(v)
		}
		if v, ok := attributeDataMap["display_units_long"].(string); ok && v != "" {
			attributes.DisplayUnitsLong = *librato.String(v)
		}
		if v, ok := attributeDataMap["display_units_short"].(string); ok && v != "" {
			attributes.DisplayUnitsShort = *librato.String(v)
		}
		if v, ok := attributeDataMap["created_by_ua"].(string); ok && v != "" {
			attributes.CreatedByUA = *librato.String(v)
		}
		if v, ok := attributeDataMap["display_stacked"].(bool); ok {
			attributes.DisplayStacked = *librato.Bool(v)
		}
		if v, ok := attributeDataMap["gap_detection"].(bool); ok {
			attributes.GapDetection = *librato.Bool(v)
		}
		if v, ok := attributeDataMap["aggregate"].(bool); ok {
			attributes.Aggregate = *librato.Bool(v)
		}

		metric.Attributes = attributes
		fullMetric.Attributes = attributes
	}

	log.Printf("[INFO] Updating Librato metric: %v", metric)
	_, err = client.Metrics.Edit(metric)
	if err != nil {
		return fmt.Errorf("Error updating Librato metric: %s", err)
	}

	log.Printf("[INFO] Updated Librato metric %s", metricID)

	// Wait for propagation since Librato updates are eventually consistent
	wait := resource.StateChangeConf{
		Pending:                   []string{fmt.Sprintf("%t", false)},
		Target:                    []string{fmt.Sprintf("%t", true)},
		Timeout:                   5 * time.Minute,
		MinTimeout:                2 * time.Second,
		ContinuousTargetOccurence: 5,
		Refresh: func() (interface{}, string, error) {
			log.Printf("[DEBUG] Checking if Librato Metric %s was updated yet", metricID)
			changedMetric, _, getErr := client.Metrics.Get(metricID)
			if getErr != nil {
				return changedMetric, "", getErr
			}
			isEqual := reflect.DeepEqual(*fullMetric, *changedMetric)
			log.Printf("[DEBUG] Updated Librato Metric %s match: %t", metricID, isEqual)
			return changedMetric, fmt.Sprintf("%t", isEqual), nil
		},
	}

	_, err = wait.WaitForState()
	if err != nil {
		return fmt.Errorf("Failed updating Librato Metric %s: %s", metricID, err)
	}

	return resourceLibratoMetricRead(d, meta)
}

func resourceLibratoMetricDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*librato.Client)

	id := d.Id()

	log.Printf("[INFO] Deleting Metric: %s", id)
	_, err := client.Metrics.Delete(id)
	if err != nil {
		return fmt.Errorf("Error deleting Metric: %s", err)
	}

	resource.Retry(1*time.Minute, func() *resource.RetryError {
		_, _, err := client.Metrics.Get(id)
		if err != nil {
			if errResp, ok := err.(*librato.ErrorResponse); ok && errResp.Response.StatusCode == 404 {
				return nil
			}
			log.Printf("[INFO] non-retryable error attempting to Get metric: %s", err)
			return resource.NonRetryableError(err)
		}
		return resource.RetryableError(fmt.Errorf("metric still exists"))
	})

	d.SetId("")
	return nil
}

func structToString(i interface{}) string {
	s, _ := json.Marshal(i)
	return string(s)
}
