package aws

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsS3Bucket() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsS3BucketCreate,
		Read:   resourceAwsS3BucketRead,
		Update: resourceAwsS3BucketUpdate,
		Delete: resourceAwsS3BucketDelete,
		Importer: &schema.ResourceImporter{
			State: resourceAwsS3BucketImportState,
		},

		Schema: map[string]*schema.Schema{
			"bucket": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"bucket_prefix"},
			},
			"bucket_prefix": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"bucket_domain_name": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"arn": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"acl": {
				Type:     schema.TypeString,
				Default:  "private",
				Optional: true,
			},

			"policy": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateFunc:     validateJsonString,
				DiffSuppressFunc: suppressEquivalentAwsPolicyDiffs,
			},

			"cors_rule": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"allowed_headers": {
							Type:     schema.TypeList,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"allowed_methods": {
							Type:     schema.TypeList,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"allowed_origins": {
							Type:     schema.TypeList,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"expose_headers": {
							Type:     schema.TypeList,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"max_age_seconds": {
							Type:     schema.TypeInt,
							Optional: true,
						},
					},
				},
			},

			"website": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"index_document": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"error_document": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"redirect_all_requests_to": {
							Type: schema.TypeString,
							ConflictsWith: []string{
								"website.0.index_document",
								"website.0.error_document",
								"website.0.routing_rules",
							},
							Optional: true,
						},

						"routing_rules": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateJsonString,
							StateFunc: func(v interface{}) string {
								json, _ := normalizeJsonString(v)
								return json
							},
						},
					},
				},
			},

			"hosted_zone_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"region": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"website_endpoint": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"website_domain": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"versioning": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"mfa_delete": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
					},
				},
			},

			"logging": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"target_bucket": {
							Type:     schema.TypeString,
							Required: true,
						},
						"target_prefix": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
				Set: func(v interface{}) int {
					var buf bytes.Buffer
					m := v.(map[string]interface{})
					buf.WriteString(fmt.Sprintf("%s-", m["target_bucket"]))
					buf.WriteString(fmt.Sprintf("%s-", m["target_prefix"]))
					return hashcode.String(buf.String())
				},
			},

			"lifecycle_rule": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:         schema.TypeString,
							Optional:     true,
							Computed:     true,
							ValidateFunc: validateS3BucketLifecycleRuleId,
						},
						"prefix": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"tags": tagsSchema(),
						"enabled": {
							Type:     schema.TypeBool,
							Required: true,
						},
						"abort_incomplete_multipart_upload_days": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"expiration": {
							Type:     schema.TypeSet,
							Optional: true,
							Set:      expirationHash,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"date": {
										Type:         schema.TypeString,
										Optional:     true,
										ValidateFunc: validateS3BucketLifecycleTimestamp,
									},
									"days": {
										Type:         schema.TypeInt,
										Optional:     true,
										ValidateFunc: validateS3BucketLifecycleExpirationDays,
									},
									"expired_object_delete_marker": {
										Type:     schema.TypeBool,
										Optional: true,
									},
								},
							},
						},
						"noncurrent_version_expiration": {
							Type:     schema.TypeSet,
							Optional: true,
							Set:      expirationHash,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"days": {
										Type:         schema.TypeInt,
										Optional:     true,
										ValidateFunc: validateS3BucketLifecycleExpirationDays,
									},
								},
							},
						},
						"transition": {
							Type:     schema.TypeSet,
							Optional: true,
							Set:      transitionHash,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"date": {
										Type:         schema.TypeString,
										Optional:     true,
										ValidateFunc: validateS3BucketLifecycleTimestamp,
									},
									"days": {
										Type:         schema.TypeInt,
										Optional:     true,
										ValidateFunc: validateS3BucketLifecycleTransitionDays,
									},
									"storage_class": {
										Type:         schema.TypeString,
										Required:     true,
										ValidateFunc: validateS3BucketLifecycleStorageClass,
									},
								},
							},
						},
						"noncurrent_version_transition": {
							Type:     schema.TypeSet,
							Optional: true,
							Set:      transitionHash,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"days": {
										Type:         schema.TypeInt,
										Optional:     true,
										ValidateFunc: validateS3BucketLifecycleTransitionDays,
									},
									"storage_class": {
										Type:         schema.TypeString,
										Required:     true,
										ValidateFunc: validateS3BucketLifecycleStorageClass,
									},
								},
							},
						},
					},
				},
			},

			"force_destroy": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"acceleration_status": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validateS3BucketAccelerationStatus,
			},

			"request_payer": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validateS3BucketRequestPayerType,
			},

			"replication_configuration": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"role": {
							Type:     schema.TypeString,
							Required: true,
						},
						"rules": {
							Type:     schema.TypeSet,
							Required: true,
							Set:      rulesHash,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"id": {
										Type:         schema.TypeString,
										Optional:     true,
										ValidateFunc: validateS3BucketReplicationRuleId,
									},
									"destination": {
										Type:     schema.TypeSet,
										MaxItems: 1,
										MinItems: 1,
										Required: true,
										Set:      destinationHash,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"bucket": {
													Type:         schema.TypeString,
													Required:     true,
													ValidateFunc: validateArn,
												},
												"storage_class": {
													Type:         schema.TypeString,
													Optional:     true,
													ValidateFunc: validateS3BucketReplicationDestinationStorageClass,
												},
											},
										},
									},
									"prefix": {
										Type:         schema.TypeString,
										Required:     true,
										ValidateFunc: validateS3BucketReplicationRulePrefix,
									},
									"status": {
										Type:         schema.TypeString,
										Required:     true,
										ValidateFunc: validateS3BucketReplicationRuleStatus,
									},
								},
							},
						},
					},
				},
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceAwsS3BucketCreate(d *schema.ResourceData, meta interface{}) error {
	s3conn := meta.(*AWSClient).s3conn

	// Get the bucket and acl
	var bucket string
	if v, ok := d.GetOk("bucket"); ok {
		bucket = v.(string)
	} else if v, ok := d.GetOk("bucket_prefix"); ok {
		bucket = resource.PrefixedUniqueId(v.(string))
	} else {
		bucket = resource.UniqueId()
	}
	d.Set("bucket", bucket)
	acl := d.Get("acl").(string)

	log.Printf("[DEBUG] S3 bucket create: %s, ACL: %s", bucket, acl)

	req := &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
		ACL:    aws.String(acl),
	}

	var awsRegion string
	if region, ok := d.GetOk("region"); ok {
		awsRegion = region.(string)
	} else {
		awsRegion = meta.(*AWSClient).region
	}
	log.Printf("[DEBUG] S3 bucket create: %s, using region: %s", bucket, awsRegion)

	// Special case us-east-1 region and do not set the LocationConstraint.
	// See "Request Elements: http://docs.aws.amazon.com/AmazonS3/latest/API/RESTBucketPUT.html
	if awsRegion != "us-east-1" {
		req.CreateBucketConfiguration = &s3.CreateBucketConfiguration{
			LocationConstraint: aws.String(awsRegion),
		}
	}

	if err := validateS3BucketName(bucket, awsRegion); err != nil {
		return fmt.Errorf("Error validating S3 bucket name: %s", err)
	}

	err := resource.Retry(5*time.Minute, func() *resource.RetryError {
		log.Printf("[DEBUG] Trying to create new S3 bucket: %q", bucket)
		_, err := s3conn.CreateBucket(req)
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "OperationAborted" {
				log.Printf("[WARN] Got an error while trying to create S3 bucket %s: %s", bucket, err)
				return resource.RetryableError(
					fmt.Errorf("[WARN] Error creating S3 bucket %s, retrying: %s",
						bucket, err))
			}
		}
		if err != nil {
			return resource.NonRetryableError(err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("Error creating S3 bucket: %s", err)
	}

	// Assign the bucket name as the resource ID
	d.SetId(bucket)

	return resourceAwsS3BucketUpdate(d, meta)
}

func resourceAwsS3BucketUpdate(d *schema.ResourceData, meta interface{}) error {
	s3conn := meta.(*AWSClient).s3conn
	if err := setTagsS3(s3conn, d); err != nil {
		return fmt.Errorf("%q: %s", d.Get("bucket").(string), err)
	}

	if d.HasChange("policy") {
		if err := resourceAwsS3BucketPolicyUpdate(s3conn, d); err != nil {
			return err
		}
	}

	if d.HasChange("cors_rule") {
		if err := resourceAwsS3BucketCorsUpdate(s3conn, d); err != nil {
			return err
		}
	}

	if d.HasChange("website") {
		if err := resourceAwsS3BucketWebsiteUpdate(s3conn, d); err != nil {
			return err
		}
	}

	if d.HasChange("versioning") {
		if err := resourceAwsS3BucketVersioningUpdate(s3conn, d); err != nil {
			return err
		}
	}
	if d.HasChange("acl") {
		if err := resourceAwsS3BucketAclUpdate(s3conn, d); err != nil {
			return err
		}
	}

	if d.HasChange("logging") {
		if err := resourceAwsS3BucketLoggingUpdate(s3conn, d); err != nil {
			return err
		}
	}

	if d.HasChange("lifecycle_rule") {
		if err := resourceAwsS3BucketLifecycleUpdate(s3conn, d); err != nil {
			return err
		}
	}

	if d.HasChange("acceleration_status") {
		if err := resourceAwsS3BucketAccelerationUpdate(s3conn, d); err != nil {
			return err
		}
	}

	if d.HasChange("request_payer") {
		if err := resourceAwsS3BucketRequestPayerUpdate(s3conn, d); err != nil {
			return err
		}
	}

	if d.HasChange("replication_configuration") {
		if err := resourceAwsS3BucketReplicationConfigurationUpdate(s3conn, d); err != nil {
			return err
		}
	}

	return resourceAwsS3BucketRead(d, meta)
}

func resourceAwsS3BucketRead(d *schema.ResourceData, meta interface{}) error {
	s3conn := meta.(*AWSClient).s3conn

	var err error
	_, err = s3conn.HeadBucket(&s3.HeadBucketInput{
		Bucket: aws.String(d.Id()),
	})
	if err != nil {
		if awsError, ok := err.(awserr.RequestFailure); ok && awsError.StatusCode() == 404 {
			log.Printf("[WARN] S3 Bucket (%s) not found, error code (404)", d.Id())
			d.SetId("")
			return nil
		} else {
			// some of the AWS SDK's errors can be empty strings, so let's add
			// some additional context.
			return fmt.Errorf("error reading S3 bucket \"%s\": %s", d.Id(), err)
		}
	}

	// In the import case, we won't have this
	if _, ok := d.GetOk("bucket"); !ok {
		d.Set("bucket", d.Id())
	}

	d.Set("bucket_domain_name", bucketDomainName(d.Get("bucket").(string)))

	// Read the policy
	if _, ok := d.GetOk("policy"); ok {
		pol, err := s3conn.GetBucketPolicy(&s3.GetBucketPolicyInput{
			Bucket: aws.String(d.Id()),
		})
		log.Printf("[DEBUG] S3 bucket: %s, read policy: %v", d.Id(), pol)
		if err != nil {
			if err := d.Set("policy", ""); err != nil {
				return err
			}
		} else {
			if v := pol.Policy; v == nil {
				if err := d.Set("policy", ""); err != nil {
					return err
				}
			} else {
				policy, err := normalizeJsonString(*v)
				if err != nil {
					return errwrap.Wrapf("policy contains an invalid JSON: {{err}}", err)
				}
				d.Set("policy", policy)
			}
		}
	}

	// Read the CORS
	cors, err := s3conn.GetBucketCors(&s3.GetBucketCorsInput{
		Bucket: aws.String(d.Id()),
	})
	if err != nil {
		// An S3 Bucket might not have CORS configuration set.
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() != "NoSuchCORSConfiguration" {
			return err
		}
		log.Printf("[WARN] S3 bucket: %s, no CORS configuration could be found.", d.Id())
	}
	log.Printf("[DEBUG] S3 bucket: %s, read CORS: %v", d.Id(), cors)
	if cors.CORSRules != nil {
		rules := make([]map[string]interface{}, 0, len(cors.CORSRules))
		for _, ruleObject := range cors.CORSRules {
			rule := make(map[string]interface{})
			rule["allowed_headers"] = flattenStringList(ruleObject.AllowedHeaders)
			rule["allowed_methods"] = flattenStringList(ruleObject.AllowedMethods)
			rule["allowed_origins"] = flattenStringList(ruleObject.AllowedOrigins)
			// Both the "ExposeHeaders" and "MaxAgeSeconds" might not be set.
			if ruleObject.AllowedOrigins != nil {
				rule["expose_headers"] = flattenStringList(ruleObject.ExposeHeaders)
			}
			if ruleObject.MaxAgeSeconds != nil {
				rule["max_age_seconds"] = int(*ruleObject.MaxAgeSeconds)
			}
			rules = append(rules, rule)
		}
		if err := d.Set("cors_rule", rules); err != nil {
			return err
		}
	}

	// Read the website configuration
	ws, err := s3conn.GetBucketWebsite(&s3.GetBucketWebsiteInput{
		Bucket: aws.String(d.Id()),
	})
	var websites []map[string]interface{}
	if err == nil {
		w := make(map[string]interface{})

		if v := ws.IndexDocument; v != nil {
			w["index_document"] = *v.Suffix
		}

		if v := ws.ErrorDocument; v != nil {
			w["error_document"] = *v.Key
		}

		if v := ws.RedirectAllRequestsTo; v != nil {
			if v.Protocol == nil {
				w["redirect_all_requests_to"] = *v.HostName
			} else {
				var host string
				var path string
				parsedHostName, err := url.Parse(*v.HostName)
				if err == nil {
					host = parsedHostName.Host
					path = parsedHostName.Path
				} else {
					host = *v.HostName
					path = ""
				}

				w["redirect_all_requests_to"] = (&url.URL{
					Host:   host,
					Path:   path,
					Scheme: *v.Protocol,
				}).String()
			}
		}

		if v := ws.RoutingRules; v != nil {
			rr, err := normalizeRoutingRules(v)
			if err != nil {
				return fmt.Errorf("Error while marshaling routing rules: %s", err)
			}
			w["routing_rules"] = rr
		}

		websites = append(websites, w)
	}
	if err := d.Set("website", websites); err != nil {
		return err
	}

	// Read the versioning configuration
	versioning, err := s3conn.GetBucketVersioning(&s3.GetBucketVersioningInput{
		Bucket: aws.String(d.Id()),
	})
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] S3 Bucket: %s, versioning: %v", d.Id(), versioning)
	if versioning != nil {
		vcl := make([]map[string]interface{}, 0, 1)
		vc := make(map[string]interface{})
		if versioning.Status != nil && *versioning.Status == s3.BucketVersioningStatusEnabled {
			vc["enabled"] = true
		} else {
			vc["enabled"] = false
		}

		if versioning.MFADelete != nil && *versioning.MFADelete == s3.MFADeleteEnabled {
			vc["mfa_delete"] = true
		} else {
			vc["mfa_delete"] = false
		}
		vcl = append(vcl, vc)
		if err := d.Set("versioning", vcl); err != nil {
			return err
		}
	}

	// Read the acceleration status
	accelerate, err := s3conn.GetBucketAccelerateConfiguration(&s3.GetBucketAccelerateConfigurationInput{
		Bucket: aws.String(d.Id()),
	})
	if err != nil {
		// Amazon S3 Transfer Acceleration might not be supported in the
		// given region, for example, China (Beijing) and the Government
		// Cloud does not support this feature at the moment.
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() != "UnsupportedArgument" {
			return err
		}

		var awsRegion string
		if region, ok := d.GetOk("region"); ok {
			awsRegion = region.(string)
		} else {
			awsRegion = meta.(*AWSClient).region
		}

		log.Printf("[WARN] S3 bucket: %s, the S3 Transfer Acceleration is not supported in the region: %s", d.Id(), awsRegion)
	} else {
		log.Printf("[DEBUG] S3 bucket: %s, read Acceleration: %v", d.Id(), accelerate)
		d.Set("acceleration_status", accelerate.Status)
	}

	// Read the request payer configuration.
	payer, err := s3conn.GetBucketRequestPayment(&s3.GetBucketRequestPaymentInput{
		Bucket: aws.String(d.Id()),
	})
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] S3 Bucket: %s, read request payer: %v", d.Id(), payer)
	if payer.Payer != nil {
		if err := d.Set("request_payer", *payer.Payer); err != nil {
			return err
		}
	}

	// Read the logging configuration
	logging, err := s3conn.GetBucketLogging(&s3.GetBucketLoggingInput{
		Bucket: aws.String(d.Id()),
	})
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] S3 Bucket: %s, logging: %v", d.Id(), logging)
	lcl := make([]map[string]interface{}, 0, 1)
	if v := logging.LoggingEnabled; v != nil {
		lc := make(map[string]interface{})
		if *v.TargetBucket != "" {
			lc["target_bucket"] = *v.TargetBucket
		}
		if *v.TargetPrefix != "" {
			lc["target_prefix"] = *v.TargetPrefix
		}
		lcl = append(lcl, lc)
	}
	if err := d.Set("logging", lcl); err != nil {
		return err
	}

	// Read the lifecycle configuration
	lifecycle, err := s3conn.GetBucketLifecycleConfiguration(&s3.GetBucketLifecycleConfigurationInput{
		Bucket: aws.String(d.Id()),
	})
	if err != nil {
		if awsError, ok := err.(awserr.RequestFailure); ok && awsError.StatusCode() != 404 {
			return err
		}
	}
	log.Printf("[DEBUG] S3 Bucket: %s, lifecycle: %v", d.Id(), lifecycle)
	if len(lifecycle.Rules) > 0 {
		rules := make([]map[string]interface{}, 0, len(lifecycle.Rules))

		for _, lifecycleRule := range lifecycle.Rules {
			rule := make(map[string]interface{})

			// ID
			if lifecycleRule.ID != nil && *lifecycleRule.ID != "" {
				rule["id"] = *lifecycleRule.ID
			}
			filter := lifecycleRule.Filter
			if filter != nil {
				if filter.And != nil {
					// Prefix
					if filter.And.Prefix != nil && *filter.And.Prefix != "" {
						rule["prefix"] = *filter.And.Prefix
					}
					// Tag
					if len(filter.And.Tags) > 0 {
						rule["tags"] = tagsToMapS3(filter.And.Tags)
					}
				} else {
					// Prefix
					if filter.Prefix != nil && *filter.Prefix != "" {
						rule["prefix"] = *filter.Prefix
					}
				}
			} else {
				if lifecycleRule.Prefix != nil {
					rule["prefix"] = *lifecycleRule.Prefix
				}
			}

			// Enabled
			if lifecycleRule.Status != nil {
				if *lifecycleRule.Status == s3.ExpirationStatusEnabled {
					rule["enabled"] = true
				} else {
					rule["enabled"] = false
				}
			}

			// AbortIncompleteMultipartUploadDays
			if lifecycleRule.AbortIncompleteMultipartUpload != nil {
				if lifecycleRule.AbortIncompleteMultipartUpload.DaysAfterInitiation != nil {
					rule["abort_incomplete_multipart_upload_days"] = int(*lifecycleRule.AbortIncompleteMultipartUpload.DaysAfterInitiation)
				}
			}

			// expiration
			if lifecycleRule.Expiration != nil {
				e := make(map[string]interface{})
				if lifecycleRule.Expiration.Date != nil {
					e["date"] = (*lifecycleRule.Expiration.Date).Format("2006-01-02")
				}
				if lifecycleRule.Expiration.Days != nil {
					e["days"] = int(*lifecycleRule.Expiration.Days)
				}
				if lifecycleRule.Expiration.ExpiredObjectDeleteMarker != nil {
					e["expired_object_delete_marker"] = *lifecycleRule.Expiration.ExpiredObjectDeleteMarker
				}
				rule["expiration"] = schema.NewSet(expirationHash, []interface{}{e})
			}
			// noncurrent_version_expiration
			if lifecycleRule.NoncurrentVersionExpiration != nil {
				e := make(map[string]interface{})
				if lifecycleRule.NoncurrentVersionExpiration.NoncurrentDays != nil {
					e["days"] = int(*lifecycleRule.NoncurrentVersionExpiration.NoncurrentDays)
				}
				rule["noncurrent_version_expiration"] = schema.NewSet(expirationHash, []interface{}{e})
			}
			//// transition
			if len(lifecycleRule.Transitions) > 0 {
				transitions := make([]interface{}, 0, len(lifecycleRule.Transitions))
				for _, v := range lifecycleRule.Transitions {
					t := make(map[string]interface{})
					if v.Date != nil {
						t["date"] = (*v.Date).Format("2006-01-02")
					}
					if v.Days != nil {
						t["days"] = int(*v.Days)
					}
					if v.StorageClass != nil {
						t["storage_class"] = *v.StorageClass
					}
					transitions = append(transitions, t)
				}
				rule["transition"] = schema.NewSet(transitionHash, transitions)
			}
			// noncurrent_version_transition
			if len(lifecycleRule.NoncurrentVersionTransitions) > 0 {
				transitions := make([]interface{}, 0, len(lifecycleRule.NoncurrentVersionTransitions))
				for _, v := range lifecycleRule.NoncurrentVersionTransitions {
					t := make(map[string]interface{})
					if v.NoncurrentDays != nil {
						t["days"] = int(*v.NoncurrentDays)
					}
					if v.StorageClass != nil {
						t["storage_class"] = *v.StorageClass
					}
					transitions = append(transitions, t)
				}
				rule["noncurrent_version_transition"] = schema.NewSet(transitionHash, transitions)
			}

			rules = append(rules, rule)
		}

		if err := d.Set("lifecycle_rule", rules); err != nil {
			return err
		}
	}

	// Read the bucket replication configuration
	replication, err := s3conn.GetBucketReplication(&s3.GetBucketReplicationInput{
		Bucket: aws.String(d.Id()),
	})
	if err != nil {
		if awsError, ok := err.(awserr.RequestFailure); ok && awsError.StatusCode() != 404 {
			return err
		}
	}

	log.Printf("[DEBUG] S3 Bucket: %s, read replication configuration: %v", d.Id(), replication)
	if r := replication.ReplicationConfiguration; r != nil {
		if err := d.Set("replication_configuration", flattenAwsS3BucketReplicationConfiguration(replication.ReplicationConfiguration)); err != nil {
			log.Printf("[DEBUG] Error setting replication configuration: %s", err)
			return err
		}
	}

	// Add the region as an attribute
	location, err := s3conn.GetBucketLocation(
		&s3.GetBucketLocationInput{
			Bucket: aws.String(d.Id()),
		},
	)
	if err != nil {
		return err
	}
	var region string
	if location.LocationConstraint != nil {
		region = *location.LocationConstraint
	}
	region = normalizeRegion(region)
	if err := d.Set("region", region); err != nil {
		return err
	}

	// Add the hosted zone ID for this bucket's region as an attribute
	hostedZoneID := HostedZoneIDForRegion(region)
	if err := d.Set("hosted_zone_id", hostedZoneID); err != nil {
		return err
	}

	// Add website_endpoint as an attribute
	websiteEndpoint, err := websiteEndpoint(s3conn, d)
	if err != nil {
		return err
	}
	if websiteEndpoint != nil {
		if err := d.Set("website_endpoint", websiteEndpoint.Endpoint); err != nil {
			return err
		}
		if err := d.Set("website_domain", websiteEndpoint.Domain); err != nil {
			return err
		}
	}

	tagSet, err := getTagSetS3(s3conn, d.Id())
	if err != nil {
		return err
	}

	if err := d.Set("tags", tagsToMapS3(tagSet)); err != nil {
		return err
	}

	d.Set("arn", fmt.Sprintf("arn:%s:s3:::%s", meta.(*AWSClient).partition, d.Id()))

	return nil
}

func resourceAwsS3BucketDelete(d *schema.ResourceData, meta interface{}) error {
	s3conn := meta.(*AWSClient).s3conn

	log.Printf("[DEBUG] S3 Delete Bucket: %s", d.Id())
	_, err := s3conn.DeleteBucket(&s3.DeleteBucketInput{
		Bucket: aws.String(d.Id()),
	})
	if err != nil {
		ec2err, ok := err.(awserr.Error)
		if ok && ec2err.Code() == "BucketNotEmpty" {
			if d.Get("force_destroy").(bool) {
				// bucket may have things delete them
				log.Printf("[DEBUG] S3 Bucket attempting to forceDestroy %+v", err)

				bucket := d.Get("bucket").(string)
				resp, err := s3conn.ListObjectVersions(
					&s3.ListObjectVersionsInput{
						Bucket: aws.String(bucket),
					},
				)

				if err != nil {
					return fmt.Errorf("Error S3 Bucket list Object Versions err: %s", err)
				}

				objectsToDelete := make([]*s3.ObjectIdentifier, 0)

				if len(resp.DeleteMarkers) != 0 {

					for _, v := range resp.DeleteMarkers {
						objectsToDelete = append(objectsToDelete, &s3.ObjectIdentifier{
							Key:       v.Key,
							VersionId: v.VersionId,
						})
					}
				}

				if len(resp.Versions) != 0 {
					for _, v := range resp.Versions {
						objectsToDelete = append(objectsToDelete, &s3.ObjectIdentifier{
							Key:       v.Key,
							VersionId: v.VersionId,
						})
					}
				}

				params := &s3.DeleteObjectsInput{
					Bucket: aws.String(bucket),
					Delete: &s3.Delete{
						Objects: objectsToDelete,
					},
				}

				_, err = s3conn.DeleteObjects(params)

				if err != nil {
					return fmt.Errorf("Error S3 Bucket force_destroy error deleting: %s", err)
				}

				// this line recurses until all objects are deleted or an error is returned
				return resourceAwsS3BucketDelete(d, meta)
			}
		}
		return fmt.Errorf("Error deleting S3 Bucket: %s %q", err, d.Get("bucket").(string))
	}
	return nil
}

func resourceAwsS3BucketPolicyUpdate(s3conn *s3.S3, d *schema.ResourceData) error {
	bucket := d.Get("bucket").(string)
	policy := d.Get("policy").(string)

	if policy != "" {
		log.Printf("[DEBUG] S3 bucket: %s, put policy: %s", bucket, policy)

		params := &s3.PutBucketPolicyInput{
			Bucket: aws.String(bucket),
			Policy: aws.String(policy),
		}

		err := resource.Retry(1*time.Minute, func() *resource.RetryError {
			if _, err := s3conn.PutBucketPolicy(params); err != nil {
				if awserr, ok := err.(awserr.Error); ok {
					if awserr.Code() == "MalformedPolicy" {
						return resource.RetryableError(awserr)
					}
				}
				return resource.NonRetryableError(err)
			}
			return nil
		})

		if err != nil {
			return fmt.Errorf("Error putting S3 policy: %s", err)
		}
	} else {
		log.Printf("[DEBUG] S3 bucket: %s, delete policy: %s", bucket, policy)
		_, err := s3conn.DeleteBucketPolicy(&s3.DeleteBucketPolicyInput{
			Bucket: aws.String(bucket),
		})

		if err != nil {
			return fmt.Errorf("Error deleting S3 policy: %s", err)
		}
	}

	return nil
}

func resourceAwsS3BucketCorsUpdate(s3conn *s3.S3, d *schema.ResourceData) error {
	bucket := d.Get("bucket").(string)
	rawCors := d.Get("cors_rule").([]interface{})

	if len(rawCors) == 0 {
		// Delete CORS
		log.Printf("[DEBUG] S3 bucket: %s, delete CORS", bucket)
		_, err := s3conn.DeleteBucketCors(&s3.DeleteBucketCorsInput{
			Bucket: aws.String(bucket),
		})
		if err != nil {
			return fmt.Errorf("Error deleting S3 CORS: %s", err)
		}
	} else {
		// Put CORS
		rules := make([]*s3.CORSRule, 0, len(rawCors))
		for _, cors := range rawCors {
			corsMap := cors.(map[string]interface{})
			r := &s3.CORSRule{}
			for k, v := range corsMap {
				log.Printf("[DEBUG] S3 bucket: %s, put CORS: %#v, %#v", bucket, k, v)
				if k == "max_age_seconds" {
					r.MaxAgeSeconds = aws.Int64(int64(v.(int)))
				} else {
					vMap := make([]*string, len(v.([]interface{})))
					for i, vv := range v.([]interface{}) {
						str := vv.(string)
						vMap[i] = aws.String(str)
					}
					switch k {
					case "allowed_headers":
						r.AllowedHeaders = vMap
					case "allowed_methods":
						r.AllowedMethods = vMap
					case "allowed_origins":
						r.AllowedOrigins = vMap
					case "expose_headers":
						r.ExposeHeaders = vMap
					}
				}
			}
			rules = append(rules, r)
		}
		corsInput := &s3.PutBucketCorsInput{
			Bucket: aws.String(bucket),
			CORSConfiguration: &s3.CORSConfiguration{
				CORSRules: rules,
			},
		}
		log.Printf("[DEBUG] S3 bucket: %s, put CORS: %#v", bucket, corsInput)
		_, err := s3conn.PutBucketCors(corsInput)
		if err != nil {
			return fmt.Errorf("Error putting S3 CORS: %s", err)
		}
	}

	return nil
}

func resourceAwsS3BucketWebsiteUpdate(s3conn *s3.S3, d *schema.ResourceData) error {
	ws := d.Get("website").([]interface{})

	if len(ws) == 1 {
		var w map[string]interface{}
		if ws[0] != nil {
			w = ws[0].(map[string]interface{})
		} else {
			w = make(map[string]interface{})
		}
		return resourceAwsS3BucketWebsitePut(s3conn, d, w)
	} else if len(ws) == 0 {
		return resourceAwsS3BucketWebsiteDelete(s3conn, d)
	} else {
		return fmt.Errorf("Cannot specify more than one website.")
	}
}

func resourceAwsS3BucketWebsitePut(s3conn *s3.S3, d *schema.ResourceData, website map[string]interface{}) error {
	bucket := d.Get("bucket").(string)

	var indexDocument, errorDocument, redirectAllRequestsTo, routingRules string
	if v, ok := website["index_document"]; ok {
		indexDocument = v.(string)
	}
	if v, ok := website["error_document"]; ok {
		errorDocument = v.(string)
	}
	if v, ok := website["redirect_all_requests_to"]; ok {
		redirectAllRequestsTo = v.(string)
	}
	if v, ok := website["routing_rules"]; ok {
		routingRules = v.(string)
	}

	if indexDocument == "" && redirectAllRequestsTo == "" {
		return fmt.Errorf("Must specify either index_document or redirect_all_requests_to.")
	}

	websiteConfiguration := &s3.WebsiteConfiguration{}

	if indexDocument != "" {
		websiteConfiguration.IndexDocument = &s3.IndexDocument{Suffix: aws.String(indexDocument)}
	}

	if errorDocument != "" {
		websiteConfiguration.ErrorDocument = &s3.ErrorDocument{Key: aws.String(errorDocument)}
	}

	if redirectAllRequestsTo != "" {
		redirect, err := url.Parse(redirectAllRequestsTo)
		if err == nil && redirect.Scheme != "" {
			var redirectHostBuf bytes.Buffer
			redirectHostBuf.WriteString(redirect.Host)
			if redirect.Path != "" {
				redirectHostBuf.WriteString(redirect.Path)
			}
			websiteConfiguration.RedirectAllRequestsTo = &s3.RedirectAllRequestsTo{HostName: aws.String(redirectHostBuf.String()), Protocol: aws.String(redirect.Scheme)}
		} else {
			websiteConfiguration.RedirectAllRequestsTo = &s3.RedirectAllRequestsTo{HostName: aws.String(redirectAllRequestsTo)}
		}
	}

	if routingRules != "" {
		var unmarshaledRules []*s3.RoutingRule
		if err := json.Unmarshal([]byte(routingRules), &unmarshaledRules); err != nil {
			return err
		}
		websiteConfiguration.RoutingRules = unmarshaledRules
	}

	putInput := &s3.PutBucketWebsiteInput{
		Bucket:               aws.String(bucket),
		WebsiteConfiguration: websiteConfiguration,
	}

	log.Printf("[DEBUG] S3 put bucket website: %#v", putInput)

	_, err := s3conn.PutBucketWebsite(putInput)
	if err != nil {
		return fmt.Errorf("Error putting S3 website: %s", err)
	}

	return nil
}

func resourceAwsS3BucketWebsiteDelete(s3conn *s3.S3, d *schema.ResourceData) error {
	bucket := d.Get("bucket").(string)
	deleteInput := &s3.DeleteBucketWebsiteInput{Bucket: aws.String(bucket)}

	log.Printf("[DEBUG] S3 delete bucket website: %#v", deleteInput)

	_, err := s3conn.DeleteBucketWebsite(deleteInput)
	if err != nil {
		return fmt.Errorf("Error deleting S3 website: %s", err)
	}

	d.Set("website_endpoint", "")
	d.Set("website_domain", "")

	return nil
}

func websiteEndpoint(s3conn *s3.S3, d *schema.ResourceData) (*S3Website, error) {
	// If the bucket doesn't have a website configuration, return an empty
	// endpoint
	if _, ok := d.GetOk("website"); !ok {
		return nil, nil
	}

	bucket := d.Get("bucket").(string)

	// Lookup the region for this bucket
	location, err := s3conn.GetBucketLocation(
		&s3.GetBucketLocationInput{
			Bucket: aws.String(bucket),
		},
	)
	if err != nil {
		return nil, err
	}
	var region string
	if location.LocationConstraint != nil {
		region = *location.LocationConstraint
	}

	return WebsiteEndpoint(bucket, region), nil
}

func bucketDomainName(bucket string) string {
	return fmt.Sprintf("%s.s3.amazonaws.com", bucket)
}

func WebsiteEndpoint(bucket string, region string) *S3Website {
	domain := WebsiteDomainUrl(region)
	return &S3Website{Endpoint: fmt.Sprintf("%s.%s", bucket, domain), Domain: domain}
}

func WebsiteDomainUrl(region string) string {
	region = normalizeRegion(region)

	// New regions uses different syntax for website endpoints
	// http://docs.aws.amazon.com/AmazonS3/latest/dev/WebsiteEndpoints.html
	if isOldRegion(region) {
		return fmt.Sprintf("s3-website-%s.amazonaws.com", region)
	}
	return fmt.Sprintf("s3-website.%s.amazonaws.com", region)
}

func isOldRegion(region string) bool {
	oldRegions := []string{
		"ap-northeast-1",
		"ap-southeast-1",
		"ap-southeast-2",
		"eu-west-1",
		"sa-east-1",
		"us-east-1",
		"us-gov-west-1",
		"us-west-1",
		"us-west-2",
	}
	for _, r := range oldRegions {
		if region == r {
			return true
		}
	}
	return false
}

func resourceAwsS3BucketAclUpdate(s3conn *s3.S3, d *schema.ResourceData) error {
	acl := d.Get("acl").(string)
	bucket := d.Get("bucket").(string)

	i := &s3.PutBucketAclInput{
		Bucket: aws.String(bucket),
		ACL:    aws.String(acl),
	}
	log.Printf("[DEBUG] S3 put bucket ACL: %#v", i)

	_, err := s3conn.PutBucketAcl(i)
	if err != nil {
		return fmt.Errorf("Error putting S3 ACL: %s", err)
	}

	return nil
}

func resourceAwsS3BucketVersioningUpdate(s3conn *s3.S3, d *schema.ResourceData) error {
	v := d.Get("versioning").([]interface{})
	bucket := d.Get("bucket").(string)
	vc := &s3.VersioningConfiguration{}

	if len(v) > 0 {
		c := v[0].(map[string]interface{})

		if c["enabled"].(bool) {
			vc.Status = aws.String(s3.BucketVersioningStatusEnabled)
		} else {
			vc.Status = aws.String(s3.BucketVersioningStatusSuspended)
		}

		if c["mfa_delete"].(bool) {
			vc.MFADelete = aws.String(s3.MFADeleteEnabled)
		} else {
			vc.MFADelete = aws.String(s3.MFADeleteDisabled)
		}

	} else {
		vc.Status = aws.String(s3.BucketVersioningStatusSuspended)
	}

	i := &s3.PutBucketVersioningInput{
		Bucket:                  aws.String(bucket),
		VersioningConfiguration: vc,
	}
	log.Printf("[DEBUG] S3 put bucket versioning: %#v", i)

	_, err := s3conn.PutBucketVersioning(i)
	if err != nil {
		return fmt.Errorf("Error putting S3 versioning: %s", err)
	}

	return nil
}

func resourceAwsS3BucketLoggingUpdate(s3conn *s3.S3, d *schema.ResourceData) error {
	logging := d.Get("logging").(*schema.Set).List()
	bucket := d.Get("bucket").(string)
	loggingStatus := &s3.BucketLoggingStatus{}

	if len(logging) > 0 {
		c := logging[0].(map[string]interface{})

		loggingEnabled := &s3.LoggingEnabled{}
		if val, ok := c["target_bucket"]; ok {
			loggingEnabled.TargetBucket = aws.String(val.(string))
		}
		if val, ok := c["target_prefix"]; ok {
			loggingEnabled.TargetPrefix = aws.String(val.(string))
		}

		loggingStatus.LoggingEnabled = loggingEnabled
	}

	i := &s3.PutBucketLoggingInput{
		Bucket:              aws.String(bucket),
		BucketLoggingStatus: loggingStatus,
	}
	log.Printf("[DEBUG] S3 put bucket logging: %#v", i)

	_, err := s3conn.PutBucketLogging(i)
	if err != nil {
		return fmt.Errorf("Error putting S3 logging: %s", err)
	}

	return nil
}

func resourceAwsS3BucketAccelerationUpdate(s3conn *s3.S3, d *schema.ResourceData) error {
	bucket := d.Get("bucket").(string)
	enableAcceleration := d.Get("acceleration_status").(string)

	i := &s3.PutBucketAccelerateConfigurationInput{
		Bucket: aws.String(bucket),
		AccelerateConfiguration: &s3.AccelerateConfiguration{
			Status: aws.String(enableAcceleration),
		},
	}
	log.Printf("[DEBUG] S3 put bucket acceleration: %#v", i)

	_, err := s3conn.PutBucketAccelerateConfiguration(i)
	if err != nil {
		return fmt.Errorf("Error putting S3 acceleration: %s", err)
	}

	return nil
}

func resourceAwsS3BucketRequestPayerUpdate(s3conn *s3.S3, d *schema.ResourceData) error {
	bucket := d.Get("bucket").(string)
	payer := d.Get("request_payer").(string)

	i := &s3.PutBucketRequestPaymentInput{
		Bucket: aws.String(bucket),
		RequestPaymentConfiguration: &s3.RequestPaymentConfiguration{
			Payer: aws.String(payer),
		},
	}
	log.Printf("[DEBUG] S3 put bucket request payer: %#v", i)

	_, err := s3conn.PutBucketRequestPayment(i)
	if err != nil {
		return fmt.Errorf("Error putting S3 request payer: %s", err)
	}

	return nil
}

func resourceAwsS3BucketReplicationConfigurationUpdate(s3conn *s3.S3, d *schema.ResourceData) error {
	bucket := d.Get("bucket").(string)
	replicationConfiguration := d.Get("replication_configuration").([]interface{})

	if len(replicationConfiguration) == 0 {
		i := &s3.DeleteBucketReplicationInput{
			Bucket: aws.String(bucket),
		}

		err := resource.Retry(1*time.Minute, func() *resource.RetryError {
			if _, err := s3conn.DeleteBucketReplication(i); err != nil {
				return resource.NonRetryableError(err)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("Error removing S3 bucket replication: %s", err)
		}
		return nil
	}

	hasVersioning := false
	// Validate that bucket versioning is enabled
	if versioning, ok := d.GetOk("versioning"); ok {
		v := versioning.([]interface{})

		if v[0].(map[string]interface{})["enabled"].(bool) {
			hasVersioning = true
		}
	}

	if !hasVersioning {
		return fmt.Errorf("versioning must be enabled to allow S3 bucket replication")
	}

	c := replicationConfiguration[0].(map[string]interface{})

	rc := &s3.ReplicationConfiguration{}
	if val, ok := c["role"]; ok {
		rc.Role = aws.String(val.(string))
	}

	rcRules := c["rules"].(*schema.Set).List()
	rules := []*s3.ReplicationRule{}
	for _, v := range rcRules {
		rr := v.(map[string]interface{})
		rcRule := &s3.ReplicationRule{
			Prefix: aws.String(rr["prefix"].(string)),
			Status: aws.String(rr["status"].(string)),
		}

		if rrid, ok := rr["id"]; ok {
			rcRule.ID = aws.String(rrid.(string))
		}

		ruleDestination := &s3.Destination{}
		if destination, ok := rr["destination"]; ok {
			dest := destination.(*schema.Set).List()

			bd := dest[0].(map[string]interface{})
			ruleDestination.Bucket = aws.String(bd["bucket"].(string))

			if storageClass, ok := bd["storage_class"]; ok && storageClass != "" {
				ruleDestination.StorageClass = aws.String(storageClass.(string))
			}
		}
		rcRule.Destination = ruleDestination
		rules = append(rules, rcRule)
	}

	rc.Rules = rules
	i := &s3.PutBucketReplicationInput{
		Bucket: aws.String(bucket),
		ReplicationConfiguration: rc,
	}
	log.Printf("[DEBUG] S3 put bucket replication configuration: %#v", i)

	_, err := s3conn.PutBucketReplication(i)
	if err != nil {
		return fmt.Errorf("Error putting S3 replication configuration: %s", err)
	}

	return nil
}

func resourceAwsS3BucketLifecycleUpdate(s3conn *s3.S3, d *schema.ResourceData) error {
	bucket := d.Get("bucket").(string)

	lifecycleRules := d.Get("lifecycle_rule").([]interface{})

	if len(lifecycleRules) == 0 {
		i := &s3.DeleteBucketLifecycleInput{
			Bucket: aws.String(bucket),
		}

		err := resource.Retry(1*time.Minute, func() *resource.RetryError {
			if _, err := s3conn.DeleteBucketLifecycle(i); err != nil {
				return resource.NonRetryableError(err)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("Error removing S3 lifecycle: %s", err)
		}
		return nil
	}

	rules := make([]*s3.LifecycleRule, 0, len(lifecycleRules))

	for i, lifecycleRule := range lifecycleRules {
		r := lifecycleRule.(map[string]interface{})

		rule := &s3.LifecycleRule{}

		// Filter
		tags := r["tags"].(map[string]interface{})
		filter := &s3.LifecycleRuleFilter{}
		if len(tags) > 0 {
			lifecycleRuleAndOp := &s3.LifecycleRuleAndOperator{}
			lifecycleRuleAndOp.SetPrefix(r["prefix"].(string))
			lifecycleRuleAndOp.SetTags(tagsFromMapS3(tags))
			filter.SetAnd(lifecycleRuleAndOp)
		} else {
			filter.SetPrefix(r["prefix"].(string))
		}
		rule.SetFilter(filter)

		// ID
		if val, ok := r["id"].(string); ok && val != "" {
			rule.ID = aws.String(val)
		} else {
			rule.ID = aws.String(resource.PrefixedUniqueId("tf-s3-lifecycle-"))
		}

		// Enabled
		if val, ok := r["enabled"].(bool); ok && val {
			rule.Status = aws.String(s3.ExpirationStatusEnabled)
		} else {
			rule.Status = aws.String(s3.ExpirationStatusDisabled)
		}

		// AbortIncompleteMultipartUpload
		if val, ok := r["abort_incomplete_multipart_upload_days"].(int); ok && val > 0 {
			rule.AbortIncompleteMultipartUpload = &s3.AbortIncompleteMultipartUpload{
				DaysAfterInitiation: aws.Int64(int64(val)),
			}
		}

		// Expiration
		expiration := d.Get(fmt.Sprintf("lifecycle_rule.%d.expiration", i)).(*schema.Set).List()
		if len(expiration) > 0 {
			e := expiration[0].(map[string]interface{})
			i := &s3.LifecycleExpiration{}

			if val, ok := e["date"].(string); ok && val != "" {
				t, err := time.Parse(time.RFC3339, fmt.Sprintf("%sT00:00:00Z", val))
				if err != nil {
					return fmt.Errorf("Error Parsing AWS S3 Bucket Lifecycle Expiration Date: %s", err.Error())
				}
				i.Date = aws.Time(t)
			} else if val, ok := e["days"].(int); ok && val > 0 {
				i.Days = aws.Int64(int64(val))
			} else if val, ok := e["expired_object_delete_marker"].(bool); ok {
				i.ExpiredObjectDeleteMarker = aws.Bool(val)
			}
			rule.Expiration = i
		}

		// NoncurrentVersionExpiration
		nc_expiration := d.Get(fmt.Sprintf("lifecycle_rule.%d.noncurrent_version_expiration", i)).(*schema.Set).List()
		if len(nc_expiration) > 0 {
			e := nc_expiration[0].(map[string]interface{})

			if val, ok := e["days"].(int); ok && val > 0 {
				rule.NoncurrentVersionExpiration = &s3.NoncurrentVersionExpiration{
					NoncurrentDays: aws.Int64(int64(val)),
				}
			}
		}

		// Transitions
		transitions := d.Get(fmt.Sprintf("lifecycle_rule.%d.transition", i)).(*schema.Set).List()
		if len(transitions) > 0 {
			rule.Transitions = make([]*s3.Transition, 0, len(transitions))
			for _, transition := range transitions {
				transition := transition.(map[string]interface{})
				i := &s3.Transition{}
				if val, ok := transition["date"].(string); ok && val != "" {
					t, err := time.Parse(time.RFC3339, fmt.Sprintf("%sT00:00:00Z", val))
					if err != nil {
						return fmt.Errorf("Error Parsing AWS S3 Bucket Lifecycle Expiration Date: %s", err.Error())
					}
					i.Date = aws.Time(t)
				} else if val, ok := transition["days"].(int); ok && val >= 0 {
					i.Days = aws.Int64(int64(val))
				}
				if val, ok := transition["storage_class"].(string); ok && val != "" {
					i.StorageClass = aws.String(val)
				}

				rule.Transitions = append(rule.Transitions, i)
			}
		}
		// NoncurrentVersionTransitions
		nc_transitions := d.Get(fmt.Sprintf("lifecycle_rule.%d.noncurrent_version_transition", i)).(*schema.Set).List()
		if len(nc_transitions) > 0 {
			rule.NoncurrentVersionTransitions = make([]*s3.NoncurrentVersionTransition, 0, len(nc_transitions))
			for _, transition := range nc_transitions {
				transition := transition.(map[string]interface{})
				i := &s3.NoncurrentVersionTransition{}
				if val, ok := transition["days"].(int); ok && val >= 0 {
					i.NoncurrentDays = aws.Int64(int64(val))
				}
				if val, ok := transition["storage_class"].(string); ok && val != "" {
					i.StorageClass = aws.String(val)
				}

				rule.NoncurrentVersionTransitions = append(rule.NoncurrentVersionTransitions, i)
			}
		}

		rules = append(rules, rule)
	}

	i := &s3.PutBucketLifecycleConfigurationInput{
		Bucket: aws.String(bucket),
		LifecycleConfiguration: &s3.BucketLifecycleConfiguration{
			Rules: rules,
		},
	}

	err := resource.Retry(1*time.Minute, func() *resource.RetryError {
		if _, err := s3conn.PutBucketLifecycleConfiguration(i); err != nil {
			return resource.NonRetryableError(err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("Error putting S3 lifecycle: %s", err)
	}

	return nil
}

func flattenAwsS3BucketReplicationConfiguration(r *s3.ReplicationConfiguration) []map[string]interface{} {
	replication_configuration := make([]map[string]interface{}, 0, 1)
	m := make(map[string]interface{})

	if r.Role != nil && *r.Role != "" {
		m["role"] = *r.Role
	}

	rules := make([]interface{}, 0, len(r.Rules))
	for _, v := range r.Rules {
		t := make(map[string]interface{})
		if v.Destination != nil {
			rd := make(map[string]interface{})
			if v.Destination.Bucket != nil {
				rd["bucket"] = *v.Destination.Bucket
			}
			if v.Destination.StorageClass != nil {
				rd["storage_class"] = *v.Destination.StorageClass
			}
			t["destination"] = schema.NewSet(destinationHash, []interface{}{rd})
		}

		if v.ID != nil {
			t["id"] = *v.ID
		}
		if v.Prefix != nil {
			t["prefix"] = *v.Prefix
		}
		if v.Status != nil {
			t["status"] = *v.Status
		}
		rules = append(rules, t)
	}
	m["rules"] = schema.NewSet(rulesHash, rules)

	replication_configuration = append(replication_configuration, m)

	return replication_configuration
}

func normalizeRoutingRules(w []*s3.RoutingRule) (string, error) {
	withNulls, err := json.Marshal(w)
	if err != nil {
		return "", err
	}

	var rules []map[string]interface{}
	if err := json.Unmarshal(withNulls, &rules); err != nil {
		return "", err
	}

	var cleanRules []map[string]interface{}
	for _, rule := range rules {
		cleanRules = append(cleanRules, removeNil(rule))
	}

	withoutNulls, err := json.Marshal(cleanRules)
	if err != nil {
		return "", err
	}

	return string(withoutNulls), nil
}

func removeNil(data map[string]interface{}) map[string]interface{} {
	withoutNil := make(map[string]interface{})

	for k, v := range data {
		if v == nil {
			continue
		}

		switch v.(type) {
		case map[string]interface{}:
			withoutNil[k] = removeNil(v.(map[string]interface{}))
		default:
			withoutNil[k] = v
		}
	}

	return withoutNil
}

// DEPRECATED. Please consider using `normalizeJsonString` function instead.
func normalizeJson(jsonString interface{}) string {
	if jsonString == nil || jsonString == "" {
		return ""
	}
	var j interface{}
	err := json.Unmarshal([]byte(jsonString.(string)), &j)
	if err != nil {
		return fmt.Sprintf("Error parsing JSON: %s", err)
	}
	b, _ := json.Marshal(j)
	return string(b[:])
}

func normalizeRegion(region string) string {
	// Default to us-east-1 if the bucket doesn't have a region:
	// http://docs.aws.amazon.com/AmazonS3/latest/API/RESTBucketGETlocation.html
	if region == "" {
		region = "us-east-1"
	}

	return region
}

func validateS3BucketAccelerationStatus(v interface{}, k string) (ws []string, errors []error) {
	validTypes := map[string]struct{}{
		"Enabled":   struct{}{},
		"Suspended": struct{}{},
	}

	if _, ok := validTypes[v.(string)]; !ok {
		errors = append(errors, fmt.Errorf("S3 Bucket Acceleration Status %q is invalid, must be %q or %q", v.(string), "Enabled", "Suspended"))
	}
	return
}

func validateS3BucketRequestPayerType(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if value != s3.PayerRequester && value != s3.PayerBucketOwner {
		errors = append(errors, fmt.Errorf(
			"%q contains an invalid Request Payer type %q. Valid types are either %q or %q",
			k, value, s3.PayerRequester, s3.PayerBucketOwner))
	}
	return
}

// validateS3BucketName validates any S3 bucket name that is not inside the us-east-1 region.
// Buckets outside of this region have to be DNS-compliant. After the same restrictions are
// applied to buckets in the us-east-1 region, this function can be refactored as a SchemaValidateFunc
func validateS3BucketName(value string, region string) error {
	if region != "us-east-1" {
		if (len(value) < 3) || (len(value) > 63) {
			return fmt.Errorf("%q must contain from 3 to 63 characters", value)
		}
		if !regexp.MustCompile(`^[0-9a-z-.]+$`).MatchString(value) {
			return fmt.Errorf("only lowercase alphanumeric characters and hyphens allowed in %q", value)
		}
		if regexp.MustCompile(`^(?:[0-9]{1,3}\.){3}[0-9]{1,3}$`).MatchString(value) {
			return fmt.Errorf("%q must not be formatted as an IP address", value)
		}
		if strings.HasPrefix(value, `.`) {
			return fmt.Errorf("%q cannot start with a period", value)
		}
		if strings.HasSuffix(value, `.`) {
			return fmt.Errorf("%q cannot end with a period", value)
		}
		if strings.Contains(value, `..`) {
			return fmt.Errorf("%q can be only one period between labels", value)
		}
	} else {
		if len(value) > 255 {
			return fmt.Errorf("%q must contain less than 256 characters", value)
		}
		if !regexp.MustCompile(`^[0-9a-zA-Z-._]+$`).MatchString(value) {
			return fmt.Errorf("only alphanumeric characters, hyphens, periods, and underscores allowed in %q", value)
		}
	}
	return nil
}

func expirationHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	if v, ok := m["date"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}
	if v, ok := m["days"]; ok {
		buf.WriteString(fmt.Sprintf("%d-", v.(int)))
	}
	if v, ok := m["expired_object_delete_marker"]; ok {
		buf.WriteString(fmt.Sprintf("%t-", v.(bool)))
	}
	return hashcode.String(buf.String())
}

func transitionHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	if v, ok := m["date"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}
	if v, ok := m["days"]; ok {
		buf.WriteString(fmt.Sprintf("%d-", v.(int)))
	}
	if v, ok := m["storage_class"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}
	return hashcode.String(buf.String())
}

func rulesHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	if v, ok := m["id"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}
	if v, ok := m["prefix"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}
	if v, ok := m["status"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}
	return hashcode.String(buf.String())
}

func destinationHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	if v, ok := m["bucket"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}
	if v, ok := m["storage_class"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}
	return hashcode.String(buf.String())
}

type S3Website struct {
	Endpoint, Domain string
}
