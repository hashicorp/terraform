package aws

import (
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsCloudFrontDistribution() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsCloudFrontDistributionCreate,
		Read:   resourceAwsCloudFrontDistributionRead,
		Update: resourceAwsCloudFrontDistributionUpdate,
		Delete: resourceAwsCloudFrontDistributionDelete,

		Schema: map[string]*schema.Schema{
			"aliases": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      aliasesHash,
			},
			"cache_behavior": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Set:      cacheBehaviorHash,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"allowed_methods": &schema.Schema{
							Type:     schema.TypeList,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"cached_methods": &schema.Schema{
							Type:     schema.TypeList,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"compress": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"default_ttl": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						"forwarded_values": &schema.Schema{
							Type:     schema.TypeSet,
							Required: true,
							Set:      forwardedValuesHash,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"cookies": &schema.Schema{
										Type:     schema.TypeSet,
										Required: true,
										Set:      cookiePreferenceHash,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"forward": &schema.Schema{
													Type:     schema.TypeString,
													Required: true,
												},
												"whitelisted_names": &schema.Schema{
													Type:     schema.TypeList,
													Optional: true,
													Elem:     &schema.Schema{Type: schema.TypeString},
												},
											},
										},
									},
									"headers": &schema.Schema{
										Type:     schema.TypeList,
										Optional: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
									"query_string": &schema.Schema{
										Type:     schema.TypeBool,
										Required: true,
									},
								},
							},
						},
						"max_ttl": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						"min_ttl": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						"path_pattern": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"smooth_streaming": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
						},
						"target_origin_id": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"trusted_signers": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"viewer_protocol_policy": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"comment": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"custom_error_response": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Set:      customErrorResponseHash,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"error_caching_min_ttl": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},
						"error_code": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						"response_code": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},
						"response_page_path": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"default_cache_behavior": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Set:      defaultCacheBehaviorHash,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"allowed_methods": &schema.Schema{
							Type:     schema.TypeList,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"cached_methods": &schema.Schema{
							Type:     schema.TypeList,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"compress": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"default_ttl": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						"forwarded_values": &schema.Schema{
							Type:     schema.TypeSet,
							Required: true,
							Set:      forwardedValuesHash,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"cookies": &schema.Schema{
										Type:     schema.TypeSet,
										Optional: true,
										Set:      cookiePreferenceHash,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"forward": &schema.Schema{
													Type:     schema.TypeString,
													Required: true,
												},
												"whitelisted_names": &schema.Schema{
													Type:     schema.TypeList,
													Optional: true,
													Elem:     &schema.Schema{Type: schema.TypeString},
												},
											},
										},
									},
									"headers": &schema.Schema{
										Type:     schema.TypeList,
										Optional: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
									"query_string": &schema.Schema{
										Type:     schema.TypeBool,
										Required: true,
									},
								},
							},
						},
						"max_ttl": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						"min_ttl": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						"smooth_streaming": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
						},
						"target_origin_id": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"trusted_signers": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"viewer_protocol_policy": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"default_root_object": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"enabled": &schema.Schema{
				Type:     schema.TypeBool,
				Required: true,
			},
			"logging_config": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Set:      loggingConfigHash,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"bucket": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"include_cookies": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"prefix": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "",
						},
					},
				},
			},
			"origin": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Set:      originHash,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"custom_origin_config": &schema.Schema{
							Type:          schema.TypeSet,
							Optional:      true,
							ConflictsWith: []string{"origin.s3_origin_config"},
							Set:           customOriginConfigHash,
							MaxItems:      1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"http_port": &schema.Schema{
										Type:     schema.TypeInt,
										Required: true,
									},
									"https_port": &schema.Schema{
										Type:     schema.TypeInt,
										Required: true,
									},
									"origin_protocol_policy": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
									"origin_ssl_protocols": &schema.Schema{
										Type:     schema.TypeList,
										Required: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
								},
							},
						},
						"domain_name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"custom_header": &schema.Schema{
							Type:     schema.TypeSet,
							Optional: true,
							Set:      originCustomHeaderHash,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
									"value": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
						"origin_id": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"origin_path": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"s3_origin_config": &schema.Schema{
							Type:          schema.TypeSet,
							Optional:      true,
							ConflictsWith: []string{"origin.custom_origin_config"},
							Set:           s3OriginConfigHash,
							MaxItems:      1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"origin_access_identity": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
										Default:  "",
									},
								},
							},
						},
					},
				},
			},
			"price_class": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "PriceClass_All",
			},
			"restrictions": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Set:      restrictionsHash,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"geo_restriction": &schema.Schema{
							Type:     schema.TypeSet,
							Required: true,
							Set:      geoRestrictionHash,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"locations": &schema.Schema{
										Type:     schema.TypeList,
										Optional: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
									"restriction_type": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
					},
				},
			},
			"viewer_certificate": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Set:      viewerCertificateHash,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"acm_certificate_arn": &schema.Schema{
							Type:          schema.TypeString,
							Optional:      true,
							ConflictsWith: []string{"viewer_certificate.cloudfront_default_certificate", "viewer_certificate.iam_certificate_id"},
						},
						"cloudfront_default_certificate": &schema.Schema{
							Type:          schema.TypeBool,
							Optional:      true,
							ConflictsWith: []string{"viewer_certificate.acm_certificate_arn", "viewer_certificate.iam_certificate_id"},
						},
						"iam_certificate_id": &schema.Schema{
							Type:          schema.TypeString,
							Optional:      true,
							ConflictsWith: []string{"viewer_certificate.acm_certificate_arn", "viewer_certificate.cloudfront_default_certificate"},
						},
						"minimum_protocol_version": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "SSLv3",
						},
						"ssl_support_method": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"web_acl_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"caller_reference": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"status": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"active_trusted_signers": &schema.Schema{
				Type:     schema.TypeMap,
				Computed: true,
			},
			"domain_name": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"last_modified_time": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"in_progress_validation_batches": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},
			"etag": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"hosted_zone_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			// retain_on_delete is a non-API attribute that may help facilitate speedy
			// deletion of a resoruce. It's mainly here for testing purposes, so
			// enable at your own risk.
			"retain_on_delete": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
		},
	}
}

func resourceAwsCloudFrontDistributionCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudfrontconn
	params := &cloudfront.CreateDistributionInput{
		DistributionConfig: expandDistributionConfig(d),
	}

	resp, err := conn.CreateDistribution(params)
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
		log.Printf("[WARN] Removing Distribtuion ID %s with retain_on_delete set. Please delete this distribution manually.", d.Id())
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
		Pending:    []string{"InProgress", "Deployed"},
		Target:     []string{"Deployed"},
		Refresh:    resourceAwsCloudFrontWebDistributionStateRefreshFunc(id, meta),
		Timeout:    40 * time.Minute,
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
			log.Printf("Error on retrieving CloudFront distribution when waiting: %s", err)
			return nil, "", err
		}

		if resp == nil {
			return nil, "", nil
		}

		return resp.Distribution, *resp.Distribution.Status, nil
	}
}
