package aws

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/route53"
)

func resourceAwsRoute53HealthCheck() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsRoute53HealthCheckCreate,
		Read:   resourceAwsRoute53HealthCheckRead,
		Update: resourceAwsRoute53HealthCheckUpdate,
		Delete: resourceAwsRoute53HealthCheckDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"type": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				StateFunc: func(val interface{}) string {
					return strings.ToUpper(val.(string))
				},
				ValidateFunc: validation.StringInSlice([]string{
					route53.HealthCheckTypeCalculated,
					route53.HealthCheckTypeCloudwatchMetric,
					route53.HealthCheckTypeHttp,
					route53.HealthCheckTypeHttpStrMatch,
					route53.HealthCheckTypeHttps,
					route53.HealthCheckTypeHttpsStrMatch,
					route53.HealthCheckTypeTcp,
				}, true),
			},
			"failure_threshold": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"request_interval": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true, // todo this should be updateable but the awslabs route53 service doesnt have the ability
			},
			"ip_address": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"fqdn": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"port": {
				Type:     schema.TypeInt,
				Optional: true,
			},

			"invert_healthcheck": {
				Type:     schema.TypeBool,
				Optional: true,
			},

			"resource_path": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"search_string": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"measure_latency": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: true,
			},

			"child_healthchecks": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
				Set:      schema.HashString,
			},
			"child_health_threshold": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntAtMost(256),
			},

			"cloudwatch_alarm_name": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"cloudwatch_alarm_region": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"insufficient_data_health_status": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"reference_name": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"enable_sni": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},

			"regions": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
				Set:      schema.HashString,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceAwsRoute53HealthCheckUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).r53conn

	updateHealthCheck := &route53.UpdateHealthCheckInput{
		HealthCheckId: aws.String(d.Id()),
	}

	if d.HasChange("failure_threshold") {
		updateHealthCheck.FailureThreshold = aws.Int64(int64(d.Get("failure_threshold").(int)))
	}

	if d.HasChange("fqdn") {
		updateHealthCheck.FullyQualifiedDomainName = aws.String(d.Get("fqdn").(string))
	}

	if d.HasChange("port") {
		updateHealthCheck.Port = aws.Int64(int64(d.Get("port").(int)))
	}

	if d.HasChange("resource_path") {
		updateHealthCheck.ResourcePath = aws.String(d.Get("resource_path").(string))
	}

	if d.HasChange("invert_healthcheck") {
		updateHealthCheck.Inverted = aws.Bool(d.Get("invert_healthcheck").(bool))
	}

	if d.HasChange("child_healthchecks") {
		updateHealthCheck.ChildHealthChecks = expandStringList(d.Get("child_healthchecks").(*schema.Set).List())

	}
	if d.HasChange("child_health_threshold") {
		updateHealthCheck.HealthThreshold = aws.Int64(int64(d.Get("child_health_threshold").(int)))
	}

	if d.HasChange("search_string") {
		updateHealthCheck.SearchString = aws.String(d.Get("search_string").(string))
	}

	if d.HasChange("cloudwatch_alarm_name") || d.HasChange("cloudwatch_alarm_region") {
		cloudwatchAlarm := &route53.AlarmIdentifier{
			Name:   aws.String(d.Get("cloudwatch_alarm_name").(string)),
			Region: aws.String(d.Get("cloudwatch_alarm_region").(string)),
		}

		updateHealthCheck.AlarmIdentifier = cloudwatchAlarm
	}

	if d.HasChange("insufficient_data_health_status") {
		updateHealthCheck.InsufficientDataHealthStatus = aws.String(d.Get("insufficient_data_health_status").(string))
	}

	if d.HasChange("enable_sni") {
		updateHealthCheck.EnableSNI = aws.Bool(d.Get("enable_sni").(bool))
	}

	if d.HasChange("regions") {
		updateHealthCheck.Regions = expandStringList(d.Get("regions").(*schema.Set).List())
	}

	_, err := conn.UpdateHealthCheck(updateHealthCheck)
	if err != nil {
		return err
	}

	if err := setTagsR53(conn, d, "healthcheck"); err != nil {
		return err
	}

	return resourceAwsRoute53HealthCheckRead(d, meta)
}

func resourceAwsRoute53HealthCheckCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).r53conn

	healthConfig := &route53.HealthCheckConfig{
		Type: aws.String(d.Get("type").(string)),
	}

	if v, ok := d.GetOk("request_interval"); ok {
		healthConfig.RequestInterval = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("failure_threshold"); ok {
		healthConfig.FailureThreshold = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("fqdn"); ok {
		healthConfig.FullyQualifiedDomainName = aws.String(v.(string))
	}

	if v, ok := d.GetOk("search_string"); ok {
		healthConfig.SearchString = aws.String(v.(string))
	}

	if v, ok := d.GetOk("ip_address"); ok {
		healthConfig.IPAddress = aws.String(v.(string))
	}

	if v, ok := d.GetOk("port"); ok {
		healthConfig.Port = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("resource_path"); ok {
		healthConfig.ResourcePath = aws.String(v.(string))
	}

	if *healthConfig.Type != route53.HealthCheckTypeCalculated && *healthConfig.Type != route53.HealthCheckTypeCloudwatchMetric {
		if v, ok := d.GetOk("measure_latency"); ok {
			healthConfig.MeasureLatency = aws.Bool(v.(bool))
		}
	}

	if v, ok := d.GetOk("invert_healthcheck"); ok {
		healthConfig.Inverted = aws.Bool(v.(bool))
	}

	if v, ok := d.GetOk("enable_sni"); ok {
		healthConfig.EnableSNI = aws.Bool(v.(bool))
	}

	if *healthConfig.Type == route53.HealthCheckTypeCalculated {
		if v, ok := d.GetOk("child_healthchecks"); ok {
			healthConfig.ChildHealthChecks = expandStringList(v.(*schema.Set).List())
		}

		if v, ok := d.GetOk("child_health_threshold"); ok {
			healthConfig.HealthThreshold = aws.Int64(int64(v.(int)))
		}
	}

	if *healthConfig.Type == route53.HealthCheckTypeCloudwatchMetric {
		cloudwatchAlarmIdentifier := &route53.AlarmIdentifier{}

		if v, ok := d.GetOk("cloudwatch_alarm_name"); ok {
			cloudwatchAlarmIdentifier.Name = aws.String(v.(string))
		}

		if v, ok := d.GetOk("cloudwatch_alarm_region"); ok {
			cloudwatchAlarmIdentifier.Region = aws.String(v.(string))
		}

		healthConfig.AlarmIdentifier = cloudwatchAlarmIdentifier

		if v, ok := d.GetOk("insufficient_data_health_status"); ok {
			healthConfig.InsufficientDataHealthStatus = aws.String(v.(string))
		}
	}

	if v, ok := d.GetOk("regions"); ok {
		healthConfig.Regions = expandStringList(v.(*schema.Set).List())
	}

	callerRef := resource.UniqueId()
	if v, ok := d.GetOk("reference_name"); ok {
		callerRef = fmt.Sprintf("%s-%s", v.(string), callerRef)
	}

	input := &route53.CreateHealthCheckInput{
		CallerReference:   aws.String(callerRef),
		HealthCheckConfig: healthConfig,
	}

	resp, err := conn.CreateHealthCheck(input)

	if err != nil {
		return err
	}

	d.SetId(*resp.HealthCheck.Id)

	if err := setTagsR53(conn, d, "healthcheck"); err != nil {
		return err
	}

	return resourceAwsRoute53HealthCheckRead(d, meta)
}

func resourceAwsRoute53HealthCheckRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).r53conn

	read, err := conn.GetHealthCheck(&route53.GetHealthCheckInput{HealthCheckId: aws.String(d.Id())})
	if err != nil {
		if r53err, ok := err.(awserr.Error); ok && r53err.Code() == "NoSuchHealthCheck" {
			d.SetId("")
			return nil

		}
		return err
	}

	if read == nil {
		return nil
	}

	updated := read.HealthCheck.HealthCheckConfig
	d.Set("type", updated.Type)
	d.Set("failure_threshold", updated.FailureThreshold)
	d.Set("request_interval", updated.RequestInterval)
	d.Set("fqdn", updated.FullyQualifiedDomainName)
	d.Set("search_string", updated.SearchString)
	d.Set("ip_address", updated.IPAddress)
	d.Set("port", updated.Port)
	d.Set("resource_path", updated.ResourcePath)
	d.Set("measure_latency", updated.MeasureLatency)
	d.Set("invert_healthcheck", updated.Inverted)

	if err := d.Set("child_healthchecks", flattenStringList(updated.ChildHealthChecks)); err != nil {
		return fmt.Errorf("error setting child_healthchecks: %s", err)
	}

	d.Set("child_health_threshold", updated.HealthThreshold)
	d.Set("insufficient_data_health_status", updated.InsufficientDataHealthStatus)
	d.Set("enable_sni", updated.EnableSNI)

	d.Set("regions", flattenStringList(updated.Regions))

	if updated.AlarmIdentifier != nil {
		d.Set("cloudwatch_alarm_name", updated.AlarmIdentifier.Name)
		d.Set("cloudwatch_alarm_region", updated.AlarmIdentifier.Region)
	}

	// read the tags
	req := &route53.ListTagsForResourceInput{
		ResourceId:   aws.String(d.Id()),
		ResourceType: aws.String("healthcheck"),
	}

	resp, err := conn.ListTagsForResource(req)
	if err != nil {
		return err
	}

	var tags []*route53.Tag
	if resp.ResourceTagSet != nil {
		tags = resp.ResourceTagSet.Tags
	}

	if err := d.Set("tags", tagsToMapR53(tags)); err != nil {
		return err
	}

	return nil
}

func resourceAwsRoute53HealthCheckDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).r53conn

	log.Printf("[DEBUG] Deleteing Route53 health check: %s", d.Id())
	_, err := conn.DeleteHealthCheck(&route53.DeleteHealthCheckInput{HealthCheckId: aws.String(d.Id())})
	return err
}
