package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsCloudFrontDistribution() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsCloudFrontDistributionCreate,
		Read:   resourceAwsCloudFrontDistributionRead,
		Update: resourceAwsCloudFrontDistributionUpdate,
		Delete: resourceAwsCloudFrontDistributionDelete,
		Importer: &schema.ResourceImporter{
			State: resourceAwsCloudFrontDistributionImport,
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"aliases": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      aliasesHash,
			},
			"cache_behavior": {
				Type:     schema.TypeSet,
				Optional: true,
				Set:      cacheBehaviorHash,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"allowed_methods": {
							Type:     schema.TypeList,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"cached_methods": {
							Type:     schema.TypeList,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"compress": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"default_ttl": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"forwarded_values": {
							Type:     schema.TypeSet,
							Required: true,
							Set:      forwardedValuesHash,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"cookies": {
										Type:     schema.TypeSet,
										Required: true,
										Set:      cookiePreferenceHash,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"forward": {
													Type:     schema.TypeString,
													Required: true,
												},
												"whitelisted_names": {
													Type:     schema.TypeList,
													Optional: true,
													Elem:     &schema.Schema{Type: schema.TypeString},
												},
											},
										},
									},
									"headers": {
										Type:     schema.TypeList,
										Optional: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
									"query_string": {
										Type:     schema.TypeBool,
										Required: true,
									},
									"query_string_cache_keys": {
										Type:     schema.TypeList,
										Optional: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
								},
							},
						},
						"lambda_function_association": {
							Type:     schema.TypeSet,
							Optional: true,
							MaxItems: 4,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"event_type": {
										Type:     schema.TypeString,
										Required: true,
									},
									"lambda_arn": {
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
							Set: lambdaFunctionAssociationHash,
						},
						"max_ttl": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"min_ttl": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"path_pattern": {
							Type:     schema.TypeString,
							Required: true,
						},
						"smooth_streaming": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"target_origin_id": {
							Type:     schema.TypeString,
							Required: true,
						},
						"trusted_signers": {
							Type:     schema.TypeList,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"viewer_protocol_policy": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"comment": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"custom_error_response": {
				Type:     schema.TypeSet,
				Optional: true,
				Set:      customErrorResponseHash,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"error_caching_min_ttl": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"error_code": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"response_code": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"response_page_path": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"default_cache_behavior": {
				Type:     schema.TypeSet,
				Required: true,
				Set:      defaultCacheBehaviorHash,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"allowed_methods": {
							Type:     schema.TypeList,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"cached_methods": {
							Type:     schema.TypeList,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"compress": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"default_ttl": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"forwarded_values": {
							Type:     schema.TypeSet,
							Required: true,
							Set:      forwardedValuesHash,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"cookies": {
										Type:     schema.TypeSet,
										Required: true,
										Set:      cookiePreferenceHash,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"forward": {
													Type:     schema.TypeString,
													Required: true,
												},
												"whitelisted_names": {
													Type:     schema.TypeList,
													Optional: true,
													Elem:     &schema.Schema{Type: schema.TypeString},
												},
											},
										},
									},
									"headers": {
										Type:     schema.TypeList,
										Optional: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
									"query_string": {
										Type:     schema.TypeBool,
										Required: true,
									},
									"query_string_cache_keys": {
										Type:     schema.TypeList,
										Optional: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
								},
							},
						},
						"lambda_function_association": {
							Type:     schema.TypeSet,
							Optional: true,
							MaxItems: 4,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"event_type": {
										Type:     schema.TypeString,
										Required: true,
									},
									"lambda_arn": {
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
							Set: lambdaFunctionAssociationHash,
						},
						"max_ttl": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"min_ttl": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"smooth_streaming": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"target_origin_id": {
							Type:     schema.TypeString,
							Required: true,
						},
						"trusted_signers": {
							Type:     schema.TypeList,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"viewer_protocol_policy": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"default_root_object": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"enabled": {
				Type:     schema.TypeBool,
				Required: true,
			},
			"http_version": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "http2",
				ValidateFunc: validateHTTP,
			},
			"logging_config": {
				Type:     schema.TypeSet,
				Optional: true,
				Set:      loggingConfigHash,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"bucket": {
							Type:     schema.TypeString,
							Required: true,
						},
						"include_cookies": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"prefix": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "",
						},
					},
				},
			},
			"origin": {
				Type:     schema.TypeSet,
				Required: true,
				Set:      originHash,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"custom_origin_config": {
							Type:          schema.TypeSet,
							Optional:      true,
							ConflictsWith: []string{"origin.s3_origin_config"},
							Set:           customOriginConfigHash,
							MaxItems:      1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"http_port": {
										Type:     schema.TypeInt,
										Required: true,
									},
									"https_port": {
										Type:     schema.TypeInt,
										Required: true,
									},
									"origin_keepalive_timeout": {
										Type:     schema.TypeInt,
										Optional: true,
										Default:  5,
									},
									"origin_read_timeout": {
										Type:     schema.TypeInt,
										Optional: true,
										Default:  30,
									},
									"origin_protocol_policy": {
										Type:     schema.TypeString,
										Required: true,
									},
									"origin_ssl_protocols": {
										Type:     schema.TypeList,
										Required: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
								},
							},
						},
						"domain_name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"custom_header": {
							Type:     schema.TypeSet,
							Optional: true,
							Set:      originCustomHeaderHash,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:     schema.TypeString,
										Required: true,
									},
									"value": {
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
						"origin_id": {
							Type:     schema.TypeString,
							Required: true,
						},
						"origin_path": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"s3_origin_config": {
							Type:          schema.TypeSet,
							Optional:      true,
							ConflictsWith: []string{"origin.custom_origin_config"},
							Set:           s3OriginConfigHash,
							MaxItems:      1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"origin_access_identity": {
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
					},
				},
			},
			"price_class": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "PriceClass_All",
			},
			"restrictions": {
				Type:     schema.TypeSet,
				Required: true,
				Set:      restrictionsHash,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"geo_restriction": {
							Type:     schema.TypeSet,
							Required: true,
							Set:      geoRestrictionHash,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"locations": {
										Type:     schema.TypeList,
										Optional: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
									"restriction_type": {
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
					},
				},
			},
			"viewer_certificate": {
				Type:     schema.TypeSet,
				Required: true,
				Set:      viewerCertificateHash,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"acm_certificate_arn": {
							Type:          schema.TypeString,
							Optional:      true,
							ConflictsWith: []string{"viewer_certificate.cloudfront_default_certificate", "viewer_certificate.iam_certificate_id"},
						},
						"cloudfront_default_certificate": {
							Type:          schema.TypeBool,
							Optional:      true,
							ConflictsWith: []string{"viewer_certificate.acm_certificate_arn", "viewer_certificate.iam_certificate_id"},
						},
						"iam_certificate_id": {
							Type:          schema.TypeString,
							Optional:      true,
							ConflictsWith: []string{"viewer_certificate.acm_certificate_arn", "viewer_certificate.cloudfront_default_certificate"},
						},
						"minimum_protocol_version": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "TLSv1",
						},
						"ssl_support_method": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"web_acl_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"caller_reference": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"active_trusted_signers": {
				Type:     schema.TypeMap,
				Computed: true,
			},
			"domain_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"last_modified_time": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"in_progress_validation_batches": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"etag": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"hosted_zone_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			// retain_on_delete is a non-API attribute that may help facilitate speedy
			// deletion of a resoruce. It's mainly here for testing purposes, so
			// enable at your own risk.
			"retain_on_delete": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"is_ipv6_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceAwsCloudFrontDistributionCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudfrontconn

	params := &cloudfront.CreateDistributionWithTagsInput{
		DistributionConfigWithTags: &cloudfront.DistributionConfigWithTags{
			DistributionConfig: expandDistributionConfig(d),
			Tags:               tagsFromMapCloudFront(d.Get("tags").(map[string]interface{})),
		},
	}

	resp, err := conn.CreateDistributionWithTags(params)
	if err != nil {
		return err
	}
	d.SetId(*resp.Distribution.Id)
	return resourceAwsCloudFrontDistributionRead(d, meta)
}

func resourceAwsCloudFrontDistributionRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudfrontconn
	params := &cloudfront.GetDistributionInput{
		Id: aws.String(d.Id()),
	}

	resp, err := conn.GetDistribution(params)
	if err != nil {
		if errcode, ok := err.(awserr.Error); ok && errcode.Code() == "NoSuchDistribution" {
			log.Printf("[WARN] No Distribution found: %s", d.Id())
			d.SetId("")
			return nil
		}

		return err
	}

	// Update attributes from DistributionConfig
	err = flattenDistributionConfig(d, resp.Distribution.DistributionConfig)
	if err != nil {
		return err
	}
	// Update other attributes outside of DistributionConfig
	d.SetId(*resp.Distribution.Id)
	err = d.Set("active_trusted_signers", flattenActiveTrustedSigners(resp.Distribution.ActiveTrustedSigners))
	if err != nil {
		return err
	}
	d.Set("status", resp.Distribution.Status)
	d.Set("domain_name", resp.Distribution.DomainName)
	d.Set("last_modified_time", aws.String(resp.Distribution.LastModifiedTime.String()))
	d.Set("in_progress_validation_batches", resp.Distribution.InProgressInvalidationBatches)
	d.Set("etag", resp.ETag)
	d.Set("arn", resp.Distribution.ARN)

	tagResp, err := conn.ListTagsForResource(&cloudfront.ListTagsForResourceInput{
		Resource: aws.String(d.Get("arn").(string)),
	})

	if err != nil {
		return errwrap.Wrapf(fmt.Sprintf(
			"Error retrieving EC2 tags for CloudFront Distribution %q (ARN: %q): {{err}}",
			d.Id(), d.Get("arn").(string)), err)
	}

	if err := d.Set("tags", tagsToMapCloudFront(tagResp.Tags)); err != nil {
		return err
	}

	return nil
}

func resourceAwsCloudFrontDistributionUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudfrontconn
	params := &cloudfront.UpdateDistributionInput{
		Id:                 aws.String(d.Id()),
		DistributionConfig: expandDistributionConfig(d),
		IfMatch:            aws.String(d.Get("etag").(string)),
	}
	_, err := conn.UpdateDistribution(params)
	if err != nil {
		return err
	}

	if err := setTagsCloudFront(conn, d, d.Get("arn").(string)); err != nil {
		return err
	}

	return resourceAwsCloudFrontDistributionRead(d, meta)
}

func resourceAwsCloudFrontDistributionDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudfrontconn

	// manually disable the distribution first
	d.Set("enabled", false)
	err := resourceAwsCloudFrontDistributionUpdate(d, meta)
	if err != nil {
		return err
	}

	// skip delete if retain_on_delete is enabled
	if d.Get("retain_on_delete").(bool) {
		log.Printf("[WARN] Removing CloudFront Distribution ID %q with `retain_on_delete` set. Please delete this distribution manually.", d.Id())
		d.SetId("")
		return nil
	}

	// Distribution needs to be in deployed state again before it can be deleted.
	err = resourceAwsCloudFrontDistributionWaitUntilDeployed(d.Id(), meta)
	if err != nil {
		return err
	}

	// now delete
	params := &cloudfront.DeleteDistributionInput{
		Id:      aws.String(d.Id()),
		IfMatch: aws.String(d.Get("etag").(string)),
	}

	_, err = conn.DeleteDistribution(params)
	if err != nil {
		return err
	}

	// Done
	d.SetId("")
	return nil
}

// resourceAwsCloudFrontWebDistributionWaitUntilDeployed blocks until the
// distribution is deployed. It currently takes exactly 15 minutes to deploy
// but that might change in the future.
func resourceAwsCloudFrontDistributionWaitUntilDeployed(id string, meta interface{}) error {
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"InProgress"},
		Target:     []string{"Deployed"},
		Refresh:    resourceAwsCloudFrontWebDistributionStateRefreshFunc(id, meta),
		Timeout:    70 * time.Minute,
		MinTimeout: 15 * time.Second,
		Delay:      10 * time.Minute,
	}

	_, err := stateConf.WaitForState()
	return err
}

// The refresh function for resourceAwsCloudFrontWebDistributionWaitUntilDeployed.
func resourceAwsCloudFrontWebDistributionStateRefreshFunc(id string, meta interface{}) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		conn := meta.(*AWSClient).cloudfrontconn
		params := &cloudfront.GetDistributionInput{
			Id: aws.String(id),
		}

		resp, err := conn.GetDistribution(params)
		if err != nil {
			log.Printf("[WARN] Error retrieving CloudFront Distribution %q details: %s", id, err)
			return nil, "", err
		}

		if resp == nil {
			return nil, "", nil
		}

		return resp.Distribution, *resp.Distribution.Status, nil
	}
}

// validateHTTP ensures that the http_version resource parameter is
// correct.
func validateHTTP(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)

	if value != "http1.1" && value != "http2" {
		errors = append(errors, fmt.Errorf(
			"%q contains an invalid HTTP version parameter %q. Valid parameters are either %q or %q.",
			k, value, "http1.1", "http2"))
	}
	return
}
