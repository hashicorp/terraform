package aws

import (
	"bytes"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/aws/awserr"
	"github.com/awslabs/aws-sdk-go/aws/awsutil"
	"github.com/awslabs/aws-sdk-go/service/cloudfront"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsCloudFrontWebDistribution() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsCloudFrontWebDistributionCreate,
		Read:   resourceAwsCloudFrontWebDistributionRead,
		Update: resourceAwsCloudFrontWebDistributionUpdate,
		Delete: resourceAwsCloudFrontWebDistributionDelete,

		Schema: map[string]*schema.Schema{
			"enabled": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},

			"comment": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"price_class": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "PriceClass_All",
			},

			"default_root_object": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"domain_name": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"status": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"default_viewer_protocol_policy": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "allow-all",
			},

			"default_forward_cookie": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "none",
			},

			"default_whitelisted_cookies": &schema.Schema{
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
			},

			"default_forward_query_string": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"default_minimum_ttl": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},

			"default_smooth_streaming": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},

			"default_allowed_methods": &schema.Schema{
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
				Computed: true,
			},

			"default_cached_methods": &schema.Schema{
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
				Computed: true,
			},

			"default_forwarded_headers": &schema.Schema{
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
			},

			"logging_enabled": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"logging_include_cookies": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"logging_prefix": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"logging_bucket": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"default_origin": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"minimum_ssl": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "SSLv3",
			},

			"certificate_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"ssl_support_method": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "sni-only",
			},

			"aliases": &schema.Schema{
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
			},

			"geo_restriction_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "none",
			},

			"geo_restrictions": &schema.Schema{
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
			},

			"zone_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"origin": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"domain_name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"id": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"http_port": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Default:  80,
						},

						"https_port": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Default:  443,
						},

						"origin_protocol_policy": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "http-only",
						},

						"origin_path": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "",
						},
					},
				},
				Set: resourceAwsCloudFrontWebDistributionOriginHash,
			},

			"behavior": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"pattern": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"origin": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"smooth_streaming": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
						},

						"viewer_protocol_policy": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "allow-all",
						},

						"minimum_ttl": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},

						"allowed_methods": &schema.Schema{
							Type:     schema.TypeList,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Optional: true,
						},

						"cached_methods": &schema.Schema{
							Type:     schema.TypeList,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Optional: true,
						},

						"forwarded_headers": &schema.Schema{
							Type:     schema.TypeList,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Optional: true,
						},

						"forward_cookie": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "none",
						},

						"whitelisted_cookies": &schema.Schema{
							Type:     schema.TypeList,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Optional: true,
						},
					},
				},
				Set: resourceAwsCloudFrontWebDistributionBehaviorHash,
			},
		},
	}
}

func resourceAwsCloudFrontWebDistributionCreate(d *schema.ResourceData, meta interface{}) error {
	cloudfrontconn := meta.(*AWSClient).cloudfrontconn

	// CloudFront distribution configurations require a unique Caller Reference
	callerReference := time.Now().Format(time.RFC3339Nano)
	c, err := resourceAwsCloudFrontWebDistributionDistributionConfig(d, meta, &callerReference)
	if err != nil {
		return err
	}

	res, err := cloudfrontconn.CreateDistribution(&cloudfront.CreateDistributionInput{
		DistributionConfig: c,
	})

	if err != nil {
		return fmt.Errorf("Error creating CloudFront distribution: %s", err)
	}

	d.SetId(*res.Distribution.ID)

	err = resourceAwsCloudFrontWebDistributionWaitUntilDeployed(d, meta)
	if err != nil {
		return err
	}

	return resourceAwsCloudFrontWebDistributionRead(d, meta)
}

func resourceAwsCloudFrontWebDistributionRead(d *schema.ResourceData, meta interface{}) error {
	v, err := resourceAwsCloudFrontWebDistributionDistributionRetrieve(d, meta)
	if err != nil {
		if cferr, ok := err.(awserr.Error); ok && cferr.Code() == "NoSuchDistribution" {
			// Fail quietly if resource no longer exists
			d.SetId("")
			return nil
		}
		return err
	}

	c := v.Distribution.DistributionConfig
	d.Set("enabled", c.Enabled)
	d.Set("comment", c.Comment)
	d.Set("price_class", c.PriceClass)
	d.Set("default_root_object", c.DefaultRootObject)
	d.Set("domain_name", v.Distribution.DomainName)
	d.Set("status", v.Distribution.Status)
	d.Set("default_viewer_protocol_policy", c.DefaultCacheBehavior.ViewerProtocolPolicy)
	d.Set("default_forward_cookie", c.DefaultCacheBehavior.ForwardedValues.Cookies)
	d.Set("default_whitelisted_cookies", resourceAwsCloudFrontCopyItems(c.DefaultCacheBehavior.ForwardedValues.Cookies.WhitelistedNames.Items))
	d.Set("default_forward_query_string", c.DefaultCacheBehavior.ForwardedValues.QueryString)
	d.Set("default_minimum_ttl", c.DefaultCacheBehavior.MinTTL)
	d.Set("default_smooth_streaming", c.DefaultCacheBehavior.SmoothStreaming)
	d.Set("default_allowed_methods", resourceAwsCloudFrontCopyItems(c.DefaultCacheBehavior.AllowedMethods.Items))
	d.Set("default_cached_methods", resourceAwsCloudFrontCopyItems(c.DefaultCacheBehavior.AllowedMethods.CachedMethods.Items))
	d.Set("default_forwarded_headers", resourceAwsCloudFrontCopyItems(c.DefaultCacheBehavior.ForwardedValues.Headers.Items))
	d.Set("logging_enabled", c.Logging.Enabled)
	d.Set("logging_include_cookies", c.Logging.IncludeCookies)
	d.Set("logging_prefix", c.Logging.Prefix)
	d.Set("logging_bucket", c.Logging.Bucket)
	d.Set("default_origin", c.DefaultCacheBehavior.TargetOriginID)
	d.Set("aliases", c.Aliases.Items)
	d.Set("geo_restriction_type", c.Restrictions.GeoRestriction.RestrictionType)
	d.Set("geo_restrictions", resourceAwsCloudFrontCopyItems(c.Restrictions.GeoRestriction.Items))
	d.Set("zone_id", "Z2FDTNDATAQYW2")

	d.Set("minimum_ssl", c.ViewerCertificate.MinimumProtocolVersion)
	d.Set("ssl_support_method", c.ViewerCertificate.SSLSupportMethod)
	if *c.ViewerCertificate.CloudFrontDefaultCertificate == true {
		d.Set("certificate_id", "")
	} else {
		d.Set("certificate_id", c.ViewerCertificate.IAMCertificateID)
	}

	if err := d.Set("origin", resourceAwsCloudFrontOriginGather(c.Origins)); err != nil {
		return err
	}

	if err := d.Set("behavior", resourceAwsCloudFrontBehaviorGather(c.CacheBehaviors)); err != nil {
		return err
	}

	return nil
}

func resourceAwsCloudFrontWebDistributionUpdate(d *schema.ResourceData, meta interface{}) error {
	cloudfrontconn := meta.(*AWSClient).cloudfrontconn

	// CloudFront configuration changes requires the ETag of the latest changeset
	v, err := resourceAwsCloudFrontWebDistributionDistributionRetrieve(d, meta)
	if err != nil {
		return err
	}

	c, err := resourceAwsCloudFrontWebDistributionDistributionConfig(d, meta, v.Distribution.DistributionConfig.CallerReference)
	if err != nil {
		return err
	}

	params := &cloudfront.UpdateDistributionInput{
		DistributionConfig: c,
		ID:                 aws.String(string(d.Id())),
		IfMatch:            v.ETag,
	}

	_, err = cloudfrontconn.UpdateDistribution(params)

	if err != nil {
		return fmt.Errorf("Error updating CloudFront distribution: %s", err)
	}

	err = resourceAwsCloudFrontWebDistributionWaitUntilDeployed(d, meta)
	if err != nil {
		return err
	}

	return resourceAwsCloudFrontWebDistributionRead(d, meta)
}

func resourceAwsCloudFrontWebDistributionDelete(d *schema.ResourceData, meta interface{}) error {
	cloudfrontconn := meta.(*AWSClient).cloudfrontconn

	v, err := resourceAwsCloudFrontWebDistributionDistributionRetrieve(d, meta)
	if err != nil {
		return err
	}

	// Do nothing if resource no longer exists
	if v == nil {
		return nil
	}

	// CloudFront distributions must be disabled in order to be deleted
	if d.Get("enabled") == true {
		d.Set("enabled", false)

		err := resourceAwsCloudFrontWebDistributionUpdate(d, meta)
		if err != nil {
			return err
		}

		// Retrieve the latest ETag
		v, err = resourceAwsCloudFrontWebDistributionDistributionRetrieve(d, meta)
		if err != nil {
			return err
		}
	}

	params := &cloudfront.DeleteDistributionInput{
		ID:      aws.String(string(d.Id())),
		IfMatch: v.ETag,
	}

	_, err = cloudfrontconn.DeleteDistribution(params)

	if err != nil {
		return fmt.Errorf("Error deleting CloudFront distribution: %s", err)
	}

	return nil
}

func resourceAwsCloudFrontWebDistributionDistributionConfig(
	d *schema.ResourceData, meta interface{},
	callerReference *string) (*cloudfront.DistributionConfig, error) {

	aliases := resourceAwsCloudFrontWebDistributionAwsStringLists(
		d.Get("aliases"))
	geoRestrictions := resourceAwsCloudFrontWebDistributionAwsStringLists(
		d.Get("geo_restrictions"))
	defaultAllowedMethods := resourceAwsCloudFrontWebDistributionHandleMethods(
		d.Get("default_allowed_methods"))
	defaultCachedMethods := resourceAwsCloudFrontWebDistributionHandleMethods(
		d.Get("default_cached_methods"))
	defaultForwardedHeaders := resourceAwsCloudFrontWebDistributionAwsStringLists(
		d.Get("default_forwarded_headers"))

	originsList := d.Get("origin").(*schema.Set).List()
	origins := resourceAwsCloudFrontWebDistributionExpandOrigins(originsList)
	behaviorsList := d.Get("behavior").(*schema.Set).List()
	behaviors := resourceAwsCloudFrontWebDistributionExpandBehaviors(behaviorsList)
	cookies := resourceAwsCloudFrontWebDistributionCookies(
		d.Get("default_forward_cookie"), d.Get("default_whitelisted_cookies"))

	viewerCertificate := &cloudfront.ViewerCertificate{
		MinimumProtocolVersion: aws.String(d.Get("minimum_ssl").(string)),
		SSLSupportMethod:       aws.String(d.Get("ssl_support_method").(string)),
	}
	if d.Get("certificate_id") == "" {
		viewerCertificate.CloudFrontDefaultCertificate = aws.Boolean(true)
	} else {
		viewerCertificate.IAMCertificateID = aws.String(d.Get("certificate_id").(string))
	}

	// PUT DistributionConfig requires, unlike POST, EVERY possible option to be set.
	// Except for the configurable options, these are the defaults options.
	x := &cloudfront.DistributionConfig{
		CallerReference:   callerReference,
		Enabled:           aws.Boolean(d.Get("enabled").(bool)),
		Comment:           aws.String(d.Get("comment").(string)),
		PriceClass:        aws.String(d.Get("price_class").(string)),
		DefaultRootObject: aws.String(d.Get("default_root_object").(string)),
		Aliases: &cloudfront.Aliases{
			Quantity: aws.Long(int64(len(aliases))),
			Items:    aliases,
		},
		Origins: &cloudfront.Origins{
			Quantity: aws.Long(int64(len(origins))),
			Items:    origins,
		},
		ViewerCertificate: viewerCertificate,
		Logging: &cloudfront.LoggingConfig{
			Enabled:        aws.Boolean(d.Get("logging_enabled").(bool)),
			IncludeCookies: aws.Boolean(d.Get("logging_include_cookies").(bool)),
			Prefix:         aws.String(d.Get("logging_prefix").(string)),
			Bucket:         aws.String(d.Get("logging_bucket").(string)),
		},
		Restrictions: &cloudfront.Restrictions{
			GeoRestriction: &cloudfront.GeoRestriction{
				Quantity:        aws.Long(int64(len(geoRestrictions))),
				RestrictionType: aws.String(d.Get("geo_restriction_type").(string)),
				Items:           geoRestrictions,
			},
		},
		DefaultCacheBehavior: &cloudfront.DefaultCacheBehavior{
			ForwardedValues: &cloudfront.ForwardedValues{
				Cookies:     cookies,
				QueryString: aws.Boolean(d.Get("default_forward_query_string").(bool)),
				Headers: &cloudfront.Headers{
					Quantity: aws.Long(int64(len(defaultForwardedHeaders))),
					Items:    defaultForwardedHeaders,
				},
			},
			TargetOriginID:       aws.String(d.Get("default_origin").(string)),
			ViewerProtocolPolicy: aws.String(d.Get("default_viewer_protocol_policy").(string)),
			MinTTL:               aws.Long(int64(d.Get("default_minimum_ttl").(int))),
			TrustedSigners: &cloudfront.TrustedSigners{
				Enabled:  aws.Boolean(false),
				Quantity: aws.Long(0),
			},
			SmoothStreaming: aws.Boolean(d.Get("default_smooth_streaming").(bool)),
			AllowedMethods: &cloudfront.AllowedMethods{
				Quantity: aws.Long(int64(len(defaultAllowedMethods))),
				Items:    defaultAllowedMethods,
				CachedMethods: &cloudfront.CachedMethods{
					Quantity: aws.Long(int64(len(defaultCachedMethods))),
					Items:    defaultCachedMethods,
				},
			},
		},
		CacheBehaviors: &cloudfront.CacheBehaviors{
			Quantity: aws.Long(int64(len(behaviors))),
			Items:    behaviors,
		},
		CustomErrorResponses: &cloudfront.CustomErrorResponses{
			Quantity: aws.Long(0),
		},
	}

	log.Println(awsutil.StringValue(x))

	return x, nil
}

func resourceAwsCloudFrontWebDistributionDistributionRetrieve(
	d *schema.ResourceData, meta interface{}) (*cloudfront.GetDistributionOutput, error) {
	cloudfrontconn := meta.(*AWSClient).cloudfrontconn

	req := &cloudfront.GetDistributionInput{
		ID: aws.String(d.Id()),
	}

	res, err := cloudfrontconn.GetDistribution(req)
	if err != nil {
		return nil, fmt.Errorf("Error retrieving CloudFront distribution: %s", err)
	}

	return res, nil
}

// resourceAwsCloudFrontWebDistributionWaitUntilDeployed blocks until the distribution is deployed.
// It currently takes exactly 15 minutes to deploy but that might change in the
// future.
func resourceAwsCloudFrontWebDistributionWaitUntilDeployed(
	d *schema.ResourceData, meta interface{}) error {
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"InProgress", "Deployed"},
		Target:     "Deployed",
		Refresh:    resourceAwsCloudFrontWebDistributionStateRefreshFunc(d, meta),
		Timeout:    40 * time.Minute,
		MinTimeout: 15 * time.Second,
		Delay:      10 * time.Minute,
	}

	_, err := stateConf.WaitForState()
	return err
}

func resourceAwsCloudFrontWebDistributionStateRefreshFunc(
	d *schema.ResourceData, meta interface{}) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		v, err := resourceAwsCloudFrontWebDistributionDistributionRetrieve(d, meta)

		if err != nil {
			log.Printf("Error on retrieving CloudFront distribution when waiting: %s", err)
			return nil, "", err
		}

		if v == nil {
			return nil, "", nil
		}

		return v.Distribution, *v.Distribution.Status, nil
	}
}

func resourceAwsCloudFrontWebDistributionOriginHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["domain_name"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["id"].(string)))
	buf.WriteString(fmt.Sprintf("%d-", m["http_port"].(int)))
	buf.WriteString(fmt.Sprintf("%d-", m["https_port"].(int)))
	buf.WriteString(fmt.Sprintf("%s-", m["origin_protocol_policy"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["origin_path"].(string)))

	return hashcode.String(buf.String())
}

func resourceAwsCloudFrontWebDistributionBehaviorHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["pattern"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["origin"].(string)))
	buf.WriteString(fmt.Sprintf("%v-", m["smooth_streaming"].(bool)))
	buf.WriteString(fmt.Sprintf("%s-", m["viewer_protocol_policy"].(string)))
	buf.WriteString(fmt.Sprintf("%d-", m["minimum_ttl"].(int)))
	buf.WriteString(fmt.Sprintf("%v-", m["forward_cookie"].(string)))
	resourceAwsCloudFrontWebDistributionHashStringList(m, "forwarded_headers", &buf)
	resourceAwsCloudFrontWebDistributionHashStringList(m, "whitelisted_cookies", &buf)

	// Since defaults does not work with lists and the following values does not
	// necessarily match the actual values the following is enough to uniquely
	// distinguish the accepted values.
	methods := len(m["allowed_methods"].([]interface{}))
	if methods == 0 {
		methods = 2
	}
	buf.WriteString(fmt.Sprintf("%d-", methods))
	methods = len(m["cached_methods"].([]interface{}))
	if methods == 0 {
		methods = 2
	}
	buf.WriteString(fmt.Sprintf("%d-", methods))

	return hashcode.String(buf.String())
}

func resourceAwsCloudFrontWebDistributionHashStringList(m map[string]interface{}, key string, buf *bytes.Buffer) {
	if v, ok := m[key]; ok {
		vs := v.([]interface{})
		s := make([]string, len(vs))
		for i, raw := range vs {
			s[i] = raw.(string)
		}
		sort.Strings(s)

		for _, v := range s {
			buf.WriteString(fmt.Sprintf("%s-", v))
		}
	}
}

func resourceAwsCloudFrontWebDistributionExpandOrigins(configured []interface{}) []*cloudfront.Origin {
	origins := make([]*cloudfront.Origin, 0, len(configured))

	for _, raw := range configured {
		data := raw.(map[string]interface{})

		o := &cloudfront.Origin{
			DomainName: aws.String(data["domain_name"].(string)),
			ID:         aws.String(data["id"].(string)),
			OriginPath: aws.String(data["origin_path"].(string)),
			CustomOriginConfig: &cloudfront.CustomOriginConfig{
				HTTPPort:             aws.Long(int64(data["http_port"].(int))),
				HTTPSPort:            aws.Long(int64(data["https_port"].(int))),
				OriginProtocolPolicy: aws.String(data["origin_protocol_policy"].(string)),
			},
		}

		origins = append(origins, o)
	}

	return origins
}

func resourceAwsCloudFrontWebDistributionExpandBehaviors(configured []interface{}) []*cloudfront.CacheBehavior {
	behaviors := make([]*cloudfront.CacheBehavior, 0, len(configured))

	for _, raw := range configured {
		data := raw.(map[string]interface{})

		allowedMethods := resourceAwsCloudFrontWebDistributionHandleMethods(data["allowed_methods"])
		cachedMethods := resourceAwsCloudFrontWebDistributionHandleMethods(data["cached_methods"])
		forwardedHeaders := resourceAwsCloudFrontWebDistributionAwsStringLists(data["forwarded_headers"])
		cookies := resourceAwsCloudFrontWebDistributionCookies(
			data["forward_cookie"], data["whitelisted_cookies"])

		o := &cloudfront.CacheBehavior{
			PathPattern:    aws.String(data["pattern"].(string)),
			TargetOriginID: aws.String(data["origin"].(string)),
			ForwardedValues: &cloudfront.ForwardedValues{
				Cookies:     cookies,
				QueryString: aws.Boolean(false),
				Headers: &cloudfront.Headers{
					Quantity: aws.Long(int64(len(forwardedHeaders))),
					Items:    forwardedHeaders,
				},
			},
			MinTTL: aws.Long(int64(data["minimum_ttl"].(int))),
			TrustedSigners: &cloudfront.TrustedSigners{
				Enabled:  aws.Boolean(false),
				Quantity: aws.Long(0),
			},
			ViewerProtocolPolicy: aws.String(data["viewer_protocol_policy"].(string)),
			SmoothStreaming:      aws.Boolean(data["smooth_streaming"].(bool)),
			AllowedMethods: &cloudfront.AllowedMethods{
				Quantity: aws.Long(int64(len(allowedMethods))),
				Items:    allowedMethods,
				CachedMethods: &cloudfront.CachedMethods{
					Quantity: aws.Long(int64(len(cachedMethods))),
					Items:    cachedMethods,
				},
			},
		}

		behaviors = append(behaviors, o)
	}

	return behaviors
}

func resourceAwsCloudFrontWebDistributionHandleMethods(in interface{}) []*string {
	if len(in.([]interface{})) == 0 {
		return []*string{
			aws.String("GET"),
			aws.String("HEAD"),
		}
	}

	return resourceAwsCloudFrontWebDistributionAwsStringLists(in)
}

func resourceAwsCloudFrontWebDistributionAwsStringLists(in interface{}) []*string {
	list := in.([]interface{})
	out := make([]*string, 0, len(list))
	for _, r := range list {
		s := r.(string)
		out = append(out, aws.String(s))
	}
	return out
}

func resourceAwsCloudFrontWebDistributionCookies(a, b interface{}) *cloudfront.CookiePreference {
	forwardCookie := a.(string)

	if forwardCookie != "whitelist" {
		return &cloudfront.CookiePreference{
			Forward: aws.String(forwardCookie),
		}
	}

	whitelist := resourceAwsCloudFrontWebDistributionAwsStringLists(b)

	return &cloudfront.CookiePreference{
		Forward: aws.String(forwardCookie),
		WhitelistedNames: &cloudfront.CookieNames{
			Quantity: aws.Long(int64(len(whitelist))),
			Items:    whitelist,
		},
	}
}

func resourceAwsCloudFrontOriginGather(d *cloudfront.Origins) []map[string]interface{} {
	origins := make([]map[string]interface{}, *d.Quantity)

	for i, cd := range d.Items {
		m := make(map[string]interface{})

		m["domain_name"] = *cd.DomainName
		m["id"] = *cd.ID
		if cd.CustomOriginConfig != nil {
			m["http_port"] = *cd.CustomOriginConfig.HTTPPort
			m["https_port"] = *cd.CustomOriginConfig.HTTPSPort
			m["origin_protocol_policy"] = *cd.CustomOriginConfig.OriginProtocolPolicy
		}
		m["origin_path"] = *cd.OriginPath

		origins[i] = m
	}

	return origins
}

func resourceAwsCloudFrontBehaviorGather(d *cloudfront.CacheBehaviors) []map[string]interface{} {
	behaviors := make([]map[string]interface{}, *d.Quantity)

	log.Println(awsutil.StringValue(d))

	for i, cd := range d.Items {
		m := make(map[string]interface{})

		m["pattern"] = *cd.PathPattern
		m["origin"] = *cd.TargetOriginID
		m["smooth_streaming"] = *cd.SmoothStreaming
		m["viewer_protocol_policy"] = *cd.ViewerProtocolPolicy
		m["minimum_ttl"] = int(*cd.MinTTL)
		m["allowed_methods"] = resourceAwsCloudFrontCopyItems(cd.AllowedMethods.Items)
		m["cached_methods"] = resourceAwsCloudFrontCopyItems(cd.AllowedMethods.CachedMethods.Items)
		m["forwarded_headers"] = resourceAwsCloudFrontCopyItems(cd.ForwardedValues.Headers.Items)
		m["forward_cookie"] = *cd.ForwardedValues.Cookies.Forward
		if cd.ForwardedValues.Cookies.WhitelistedNames != nil {
			m["whitelisted_cookies"] = resourceAwsCloudFrontCopyItems(cd.ForwardedValues.Cookies.WhitelistedNames.Items)
		}

		behaviors[i] = m
	}

	log.Println("resourceAwsCloudFrontBehaviorGather:", behaviors)

	return behaviors
}

func resourceAwsCloudFrontCopyItems(d []*string) []string {
	list := make([]string, 0, len(d))
	for _, item := range d {
		list = append(list, *item)
	}
	return list
}
