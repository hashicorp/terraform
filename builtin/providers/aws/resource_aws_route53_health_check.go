package aws

import (
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/schema"

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

		Schema: map[string]*schema.Schema{
			"type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"failure_threshold": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},
			"request_interval": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true, // todo this should be updateable but the awslabs route53 service doesnt have the ability
			},
			"ip_address": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"fqdn": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"port": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},

			"resource_path": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"search_string": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"measure_latency": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: true,
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

	if d.HasChange("search_string") {
		updateHealthCheck.SearchString = aws.String(d.Get("search_string").(string))
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
		Type:             aws.String(d.Get("type").(string)),
		FailureThreshold: aws.Int64(int64(d.Get("failure_threshold").(int))),
		RequestInterval:  aws.Int64(int64(d.Get("request_interval").(int))),
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

	if *healthConfig.Type != route53.HealthCheckTypeCalculated {
		if v, ok := d.GetOk("measure_latency"); ok {
			healthConfig.MeasureLatency = aws.Bool(v.(bool))
		}
	}

	input := &route53.CreateHealthCheckInput{
		CallerReference:   aws.String(time.Now().Format(time.RFC3339Nano)),
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
	if err != nil {
		return err
	}

	return nil
}
