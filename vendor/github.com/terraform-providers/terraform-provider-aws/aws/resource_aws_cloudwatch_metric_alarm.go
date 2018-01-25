package aws

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsCloudWatchMetricAlarm() *schema.Resource {
	return &schema.Resource{
		Create:        resourceAwsCloudWatchMetricAlarmCreate,
		Read:          resourceAwsCloudWatchMetricAlarmRead,
		Update:        resourceAwsCloudWatchMetricAlarmUpdate,
		Delete:        resourceAwsCloudWatchMetricAlarmDelete,
		SchemaVersion: 1,
		MigrateState:  resourceAwsCloudWatchMetricAlarmMigrateState,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"alarm_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"comparison_operator": {
				Type:     schema.TypeString,
				Required: true,
			},
			"evaluation_periods": {
				Type:     schema.TypeInt,
				Required: true,
			},
			"metric_name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"namespace": {
				Type:     schema.TypeString,
				Required: true,
			},
			"period": {
				Type:     schema.TypeInt,
				Required: true,
			},
			"statistic": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"extended_statistic"},
			},
			"threshold": {
				Type:     schema.TypeFloat,
				Required: true,
			},
			"actions_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"alarm_actions": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"alarm_description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"datapoints_to_alarm": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"dimensions": {
				Type:     schema.TypeMap,
				Optional: true,
			},
			"insufficient_data_actions": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"ok_actions": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"unit": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"extended_statistic": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"statistic"},
			},
			"treat_missing_data": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "missing",
				ValidateFunc: validation.StringInSlice([]string{"breaching", "notBreaching", "ignore", "missing"}, true),
			},
			"evaluate_low_sample_count_percentiles": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringInSlice([]string{"evaluate", "ignore"}, true),
			},
		},
	}
}

func resourceAwsCloudWatchMetricAlarmCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudwatchconn

	_, statisticOk := d.GetOk("statistic")
	_, extendedStatisticOk := d.GetOk("extended_statistic")

	if !statisticOk && !extendedStatisticOk {
		return fmt.Errorf("One of `statistic` or `extended_statistic` must be set for a cloudwatch metric alarm")
	}

	params := getAwsCloudWatchPutMetricAlarmInput(d)

	log.Printf("[DEBUG] Creating CloudWatch Metric Alarm: %#v", params)
	_, err := conn.PutMetricAlarm(&params)
	if err != nil {
		return fmt.Errorf("Creating metric alarm failed: %s", err)
	}
	d.SetId(d.Get("alarm_name").(string))
	log.Println("[INFO] CloudWatch Metric Alarm created")

	return resourceAwsCloudWatchMetricAlarmRead(d, meta)
}

func resourceAwsCloudWatchMetricAlarmRead(d *schema.ResourceData, meta interface{}) error {
	a, err := getAwsCloudWatchMetricAlarm(d, meta)
	if err != nil {
		return err
	}
	if a == nil {
		d.SetId("")
		return nil
	}

	log.Printf("[DEBUG] Reading CloudWatch Metric Alarm: %s", d.Get("alarm_name"))

	d.Set("actions_enabled", a.ActionsEnabled)

	if err := d.Set("alarm_actions", _strArrPtrToList(a.AlarmActions)); err != nil {
		log.Printf("[WARN] Error setting Alarm Actions: %s", err)
	}
	d.Set("alarm_description", a.AlarmDescription)
	d.Set("alarm_name", a.AlarmName)
	d.Set("comparison_operator", a.ComparisonOperator)
	d.Set("datapoints_to_alarm", a.DatapointsToAlarm)
	if err := d.Set("dimensions", flattenDimensions(a.Dimensions)); err != nil {
		return err
	}
	d.Set("evaluation_periods", a.EvaluationPeriods)

	if err := d.Set("insufficient_data_actions", _strArrPtrToList(a.InsufficientDataActions)); err != nil {
		log.Printf("[WARN] Error setting Insufficient Data Actions: %s", err)
	}
	d.Set("metric_name", a.MetricName)
	d.Set("namespace", a.Namespace)

	if err := d.Set("ok_actions", _strArrPtrToList(a.OKActions)); err != nil {
		log.Printf("[WARN] Error setting OK Actions: %s", err)
	}
	d.Set("period", a.Period)
	d.Set("statistic", a.Statistic)
	d.Set("threshold", a.Threshold)
	d.Set("unit", a.Unit)
	d.Set("extended_statistic", a.ExtendedStatistic)
	d.Set("treat_missing_data", a.TreatMissingData)
	d.Set("evaluate_low_sample_count_percentiles", a.EvaluateLowSampleCountPercentile)

	return nil
}

func resourceAwsCloudWatchMetricAlarmUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudwatchconn
	params := getAwsCloudWatchPutMetricAlarmInput(d)

	log.Printf("[DEBUG] Updating CloudWatch Metric Alarm: %#v", params)
	_, err := conn.PutMetricAlarm(&params)
	if err != nil {
		return fmt.Errorf("Updating metric alarm failed: %s", err)
	}
	log.Println("[INFO] CloudWatch Metric Alarm updated")

	return resourceAwsCloudWatchMetricAlarmRead(d, meta)
}

func resourceAwsCloudWatchMetricAlarmDelete(d *schema.ResourceData, meta interface{}) error {
	p, err := getAwsCloudWatchMetricAlarm(d, meta)
	if err != nil {
		return err
	}
	if p == nil {
		log.Printf("[DEBUG] CloudWatch Metric Alarm %s is already gone", d.Id())
		return nil
	}

	log.Printf("[INFO] Deleting CloudWatch Metric Alarm: %s", d.Id())

	conn := meta.(*AWSClient).cloudwatchconn
	params := cloudwatch.DeleteAlarmsInput{
		AlarmNames: []*string{aws.String(d.Id())},
	}

	if _, err := conn.DeleteAlarms(&params); err != nil {
		return fmt.Errorf("Error deleting CloudWatch Metric Alarm: %s", err)
	}
	log.Println("[INFO] CloudWatch Metric Alarm deleted")

	d.SetId("")
	return nil
}

func getAwsCloudWatchPutMetricAlarmInput(d *schema.ResourceData) cloudwatch.PutMetricAlarmInput {
	params := cloudwatch.PutMetricAlarmInput{
		AlarmName:          aws.String(d.Get("alarm_name").(string)),
		ComparisonOperator: aws.String(d.Get("comparison_operator").(string)),
		EvaluationPeriods:  aws.Int64(int64(d.Get("evaluation_periods").(int))),
		MetricName:         aws.String(d.Get("metric_name").(string)),
		Namespace:          aws.String(d.Get("namespace").(string)),
		Period:             aws.Int64(int64(d.Get("period").(int))),
		Threshold:          aws.Float64(d.Get("threshold").(float64)),
		TreatMissingData:   aws.String(d.Get("treat_missing_data").(string)),
	}

	if v := d.Get("actions_enabled"); v != nil {
		params.ActionsEnabled = aws.Bool(v.(bool))
	}

	if v, ok := d.GetOk("alarm_description"); ok {
		params.AlarmDescription = aws.String(v.(string))
	}

	if v, ok := d.GetOk("datapoints_to_alarm"); ok {
		params.DatapointsToAlarm = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("unit"); ok {
		params.Unit = aws.String(v.(string))
	}

	if v, ok := d.GetOk("statistic"); ok {
		params.Statistic = aws.String(v.(string))
	}

	if v, ok := d.GetOk("extended_statistic"); ok {
		params.ExtendedStatistic = aws.String(v.(string))
	}

	if v, ok := d.GetOk("evaluate_low_sample_count_percentiles"); ok {
		params.EvaluateLowSampleCountPercentile = aws.String(v.(string))
	}

	var alarmActions []*string
	if v := d.Get("alarm_actions"); v != nil {
		for _, v := range v.(*schema.Set).List() {
			str := v.(string)
			alarmActions = append(alarmActions, aws.String(str))
		}
		params.AlarmActions = alarmActions
	}

	var insufficientDataActions []*string
	if v := d.Get("insufficient_data_actions"); v != nil {
		for _, v := range v.(*schema.Set).List() {
			str := v.(string)
			insufficientDataActions = append(insufficientDataActions, aws.String(str))
		}
		params.InsufficientDataActions = insufficientDataActions
	}

	var okActions []*string
	if v := d.Get("ok_actions"); v != nil {
		for _, v := range v.(*schema.Set).List() {
			str := v.(string)
			okActions = append(okActions, aws.String(str))
		}
		params.OKActions = okActions
	}

	a := d.Get("dimensions").(map[string]interface{})
	dimensions := make([]*cloudwatch.Dimension, 0, len(a))
	for k, v := range a {
		dimensions = append(dimensions, &cloudwatch.Dimension{
			Name:  aws.String(k),
			Value: aws.String(v.(string)),
		})
	}
	params.Dimensions = dimensions

	return params
}

func getAwsCloudWatchMetricAlarm(d *schema.ResourceData, meta interface{}) (*cloudwatch.MetricAlarm, error) {
	conn := meta.(*AWSClient).cloudwatchconn

	params := cloudwatch.DescribeAlarmsInput{
		AlarmNames: []*string{aws.String(d.Id())},
	}

	resp, err := conn.DescribeAlarms(&params)
	if err != nil {
		return nil, err
	}

	// Find it and return it
	for idx, ma := range resp.MetricAlarms {
		if *ma.AlarmName == d.Id() {
			return resp.MetricAlarms[idx], nil
		}
	}

	return nil, nil
}

func _strArrPtrToList(strArrPtr []*string) []string {
	var result []string
	for _, elem := range strArrPtr {
		result = append(result, *elem)
	}
	return result
}

func flattenDimensions(dims []*cloudwatch.Dimension) map[string]interface{} {
	flatDims := make(map[string]interface{})
	for _, d := range dims {
		flatDims[*d.Name] = *d.Value
	}
	return flatDims
}
