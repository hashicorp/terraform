package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	elasticsearch "github.com/aws/aws-sdk-go/service/elasticsearchservice"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsElasticSearchDomainPolicy() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsElasticSearchDomainPolicyUpsert,
		Read:   resourceAwsElasticSearchDomainPolicyRead,
		Update: resourceAwsElasticSearchDomainPolicyUpsert,
		Delete: resourceAwsElasticSearchDomainPolicyDelete,

		Schema: map[string]*schema.Schema{
			"domain_name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"access_policies": {
				Type:             schema.TypeString,
				Required:         true,
				DiffSuppressFunc: suppressEquivalentAwsPolicyDiffs,
			},
		},
	}
}

func resourceAwsElasticSearchDomainPolicyRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).esconn
	name := d.Get("domain_name").(string)
	out, err := conn.DescribeElasticsearchDomain(&elasticsearch.DescribeElasticsearchDomainInput{
		DomainName: aws.String(name),
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "ResourceNotFoundException" {
			log.Printf("[WARN] ElasticSearch Domain %q not found, removing", name)
			d.SetId("")
			return nil
		}
		return err
	}

	log.Printf("[DEBUG] Received ElasticSearch domain: %s", out)

	ds := out.DomainStatus
	d.Set("access_policies", ds.AccessPolicies)

	return nil
}

func resourceAwsElasticSearchDomainPolicyUpsert(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).esconn
	domainName := d.Get("domain_name").(string)
	_, err := conn.UpdateElasticsearchDomainConfig(&elasticsearch.UpdateElasticsearchDomainConfigInput{
		DomainName:     aws.String(domainName),
		AccessPolicies: aws.String(d.Get("access_policies").(string)),
	})
	if err != nil {
		return err
	}

	d.SetId("esd-policy-" + domainName)

	err = resource.Retry(50*time.Minute, func() *resource.RetryError {
		out, err := conn.DescribeElasticsearchDomain(&elasticsearch.DescribeElasticsearchDomainInput{
			DomainName: aws.String(d.Get("domain_name").(string)),
		})
		if err != nil {
			return resource.NonRetryableError(err)
		}

		if *out.DomainStatus.Processing == false {
			return nil
		}

		return resource.RetryableError(
			fmt.Errorf("%q: Timeout while waiting for changes to be processed", d.Id()))
	})
	if err != nil {
		return err
	}

	return resourceAwsElasticSearchDomainPolicyRead(d, meta)
}

func resourceAwsElasticSearchDomainPolicyDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).esconn

	_, err := conn.UpdateElasticsearchDomainConfig(&elasticsearch.UpdateElasticsearchDomainConfigInput{
		DomainName:     aws.String(d.Get("domain_name").(string)),
		AccessPolicies: aws.String(""),
	})
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Waiting for ElasticSearch domain policy %q to be deleted", d.Get("domain_name").(string))
	err = resource.Retry(60*time.Minute, func() *resource.RetryError {
		out, err := conn.DescribeElasticsearchDomain(&elasticsearch.DescribeElasticsearchDomainInput{
			DomainName: aws.String(d.Get("domain_name").(string)),
		})
		if err != nil {
			return resource.NonRetryableError(err)
		}

		if *out.DomainStatus.Processing == false {
			return nil
		}

		return resource.RetryableError(
			fmt.Errorf("%q: Timeout while waiting for policy to be deleted", d.Id()))
	})
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}
