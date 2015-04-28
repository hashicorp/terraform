package aws

import (
	// "fmt"
	"log"
	// "strings"
	"time"

	// "github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/route53"
)

func resourceAwsRoute53HealthCheck() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsRoute53HealthCheckCreate,
		Read:   resourceAwsRoute53HealthCheckRead,
		Update: resourceAwsRoute53HealthCheckUpdate,
		Delete: resourceAwsRoute53HealthCheckDelete,

		Schema: map[string]*schema.Schema{
			"type": &schema.Schema{
				Type:     schema.TypeString, // can be {HTTP | HTTPS | HTTP_STR_MATCH | HTTPS_STR_MATCH | TCP}
				Required: true,
				ForceNew: true,
			},
			"failure_threshold": &schema.Schema{
				Type:     schema.TypeInt, // Valid Ints 1 - 10
				Required: true,
			},
			"request_interval": &schema.Schema{
				Type:     schema.TypeInt, // valid values { 10 | 30 }
				Required: true,
				ForceNew: true, // todo this should be updateable but the awslabs route53 service doesnt have the ability
			},
			"ip_address": &schema.Schema{ // if not supplied it will send a health check to FullyQualifiedDomainName
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"fully_qualified_domain_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"port": &schema.Schema{ // required for any type other than TCP
				Type:     schema.TypeInt,
				Optional: true,
			},

			"resource_path": &schema.Schema{ // must start with a '/'
				Type:     schema.TypeString, // required for everything except TCP
				Optional: true,
			},
			"search_string": &schema.Schema{ // only used for *_STR_MATCH
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceAwsRoute53HealthCheckUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).r53conn

	updateHealthCheck := &route53.UpdateHealthCheckInput{
		HealthCheckID: aws.String(d.Id()),
	}

	if d.HasChange("failure_threshold") {
		updateHealthCheck.FailureThreshold = aws.Long(int64(d.Get("failure_threshold").(int)))
	}

	if d.HasChange("fully_qualified_domain_name") {
		updateHealthCheck.FullyQualifiedDomainName = aws.String(d.Get("fully_qualified_domain_name").(string))
	}

	if d.HasChange("port") {
		updateHealthCheck.Port = aws.Long(int64(d.Get("port").(int)))
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

	return resourceAwsRoute53HealthCheckRead(d, meta)
}

func resourceAwsRoute53HealthCheckCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).r53conn

	// do we need to check if the optional fields existf before adding them?
	healthConfig := &route53.HealthCheckConfig{
		Type:             aws.String(d.Get("type").(string)),
		FailureThreshold: aws.Long(int64(d.Get("failure_threshold").(int))),
		RequestInterval:  aws.Long(int64(d.Get("request_interval").(int))),
	}

	if v, ok := d.GetOk("fully_qualified_domain_name"); ok {
		healthConfig.FullyQualifiedDomainName = aws.String(v.(string))
	}

	if v, ok := d.GetOk("search_string"); ok {
		healthConfig.SearchString = aws.String(v.(string))
	}

	if v, ok := d.GetOk("ip_address"); ok {
		healthConfig.IPAddress = aws.String(v.(string))
	}

	if v, ok := d.GetOk("port"); ok {
		healthConfig.Port = aws.Long(int64(v.(int)))
	}

	if v, ok := d.GetOk("resource_path"); ok {
		healthConfig.ResourcePath = aws.String(v.(string))
	}

	input := &route53.CreateHealthCheckInput{
		CallerReference:   aws.String(time.Now().Format(time.RFC3339Nano)),
		HealthCheckConfig: healthConfig,
	}

	wait := resource.StateChangeConf{
		Pending:    []string{"rejected"},
		Target:     "accepted",
		Timeout:    5 * time.Minute,
		MinTimeout: 1 * time.Second,
		Refresh: func() (interface{}, string, error) {
			resp, err := conn.CreateHealthCheck(input)
			if err != nil {
				return nil, "failure", err
			}
			return resp, "accepted", nil
		},
	}

	// we get the health check itself and the url of the new health check back
	respRaw, err := wait.WaitForState()
	if err != nil {
		return err
	}

	resp := respRaw.(*route53.CreateHealthCheckOutput)
	d.SetId(*resp.HealthCheck.ID) // set the id of the health check before calling the read to confirm the rest of the

	return resourceAwsRoute53HealthCheckRead(d, meta)
}

func resourceAwsRoute53HealthCheckRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).r53conn

	check, err := conn.GetHealthCheck(&route53.GetHealthCheckInput{HealthCheckID: aws.String(d.Id())})
	if err != nil {
		if r53err, ok := err.(aws.APIError); ok && r53err.Code == "NoSuchHealthCheck" {
			d.SetId("")
			return nil

		}
		return err
	}

	// todo update the internal values based on what is coming back from the get

	log.Printf("[INFO] Check coming back from amazon %s", check)
	return nil
}

func resourceAwsRoute53HealthCheckDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).r53conn

	log.Printf("[DEBUG] Deleteing Route53 health check: %s", d.Id())
	_, err := conn.DeleteHealthCheck(&route53.DeleteHealthCheckInput{HealthCheckID: aws.String(d.Id())})
	if err != nil {
		return err
	}

	return nil
}
