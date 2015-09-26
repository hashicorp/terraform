package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudfront"
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
			"origin_domain_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"origin_http_port": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  80,
			},

			"origin_https_port": &schema.Schema{
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

			"viewer_protocol_policy": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "allow-all",
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

			"forward_query_string": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"minimum_ttl": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  0,
			},

			"maximum_ttl": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  31536000,
			},

			"default_ttl": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  86400,
			},

			"smooth_streaming": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},

			"allowed_methods": &schema.Schema{
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
				Computed: true,
			},

			"cached_methods": &schema.Schema{
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
				Computed: true,
			},

			"forwarded_headers": &schema.Schema{
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
				Default:  "vip",
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

	d.SetId(*res.Distribution.Id)

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
	d.Set("viewer_protocol_policy", c.DefaultCacheBehavior.ViewerProtocolPolicy)
	d.Set("forward_cookie", c.DefaultCacheBehavior.ForwardedValues.Cookies)
	if c.DefaultCacheBehavior.ForwardedValues.Cookies.WhitelistedNames != nil {
		d.Set("whitelisted_cookies", resourceAwsCloudFrontCopyItems(c.DefaultCacheBehavior.ForwardedValues.Cookies.WhitelistedNames.Items))
	}
	d.Set("forward_query_string", c.DefaultCacheBehavior.ForwardedValues.QueryString)
	d.Set("minimum_ttl", c.DefaultCacheBehavior.MinTTL)
	d.Set("maximum_ttl", c.DefaultCacheBehavior.MaxTTL)
	d.Set("default_ttl", c.DefaultCacheBehavior.DefaultTTL)
	d.Set("smooth_streaming", c.DefaultCacheBehavior.SmoothStreaming)
	d.Set("allowed_methods", resourceAwsCloudFrontCopyItems(c.DefaultCacheBehavior.AllowedMethods.Items))
	d.Set("cached_methods", resourceAwsCloudFrontCopyItems(c.DefaultCacheBehavior.AllowedMethods.CachedMethods.Items))
	d.Set("forwarded_headers", resourceAwsCloudFrontCopyItems(c.DefaultCacheBehavior.ForwardedValues.Headers.Items))
	d.Set("logging_enabled", c.Logging.Enabled)
	d.Set("logging_include_cookies", c.Logging.IncludeCookies)
	d.Set("logging_prefix", c.Logging.Prefix)
	d.Set("logging_bucket", c.Logging.Bucket)
	d.Set("aliases", c.Aliases.Items)
	d.Set("geo_restriction_type", c.Restrictions.GeoRestriction.RestrictionType)
	d.Set("geo_restrictions", resourceAwsCloudFrontCopyItems(c.Restrictions.GeoRestriction.Items))
	d.Set("zone_id", "Z2FDTNDATAQYW2")

	d.Set("minimum_ssl", c.ViewerCertificate.MinimumProtocolVersion)
	d.Set("ssl_support_method", c.ViewerCertificate.SSLSupportMethod)
	if *c.ViewerCertificate.CloudFrontDefaultCertificate == true {
		d.Set("certificate_id", "")
	} else {
		d.Set("certificate_id", c.ViewerCertificate.IAMCertificateId)
	}

	// CloudFront distributions supports multiple origins. However most of the above
	// configuration options also apply to a single origin which would result in
	// an overwhelming API
	o := c.Origins.Items[0]
	d.Set("origin_domain_name", o.DomainName)
	d.Set("origin_path", o.OriginPath)
	d.Set("origin_http_port", o.CustomOriginConfig.HTTPPort)
	d.Set("origin_https_port", o.CustomOriginConfig.HTTPSPort)
	d.Set("origin_protocol_policy", o.CustomOriginConfig.OriginProtocolPolicy)

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
		Id:                 aws.String(string(d.Id())),
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

		// Retrieve the latest ETag again
		v, err = resourceAwsCloudFrontWebDistributionDistributionRetrieve(d, meta)
		if err != nil {
			return err
		}
	}

	params := &cloudfront.DeleteDistributionInput{
		Id:      aws.String(string(d.Id())),
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

	originId := fmt.Sprintf("%s-origin", d.Get("origin_domain_name"))
	aliases := resourceAwsCloudFrontWebDistributionAwsStringLists(
		d.Get("aliases"))
	geoRestrictions := resourceAwsCloudFrontWebDistributionAwsStringLists(
		d.Get("geo_restrictions"))
	allowedMethods := resourceAwsCloudFrontWebDistributionHandleMethods(
		d.Get("allowed_methods"))
	cachedMethods := resourceAwsCloudFrontWebDistributionHandleMethods(
		d.Get("cached_methods"))
	forwardedHeaders := resourceAwsCloudFrontWebDistributionAwsStringLists(
		d.Get("forwarded_headers"))
	cookies := resourceAwsCloudFrontWebDistributionCookies(
		d.Get("forward_cookie"), d.Get("whitelisted_cookies"))
	viewerCertificate := &cloudfront.ViewerCertificate{
		MinimumProtocolVersion: aws.String(d.Get("minimum_ssl").(string)),
		SSLSupportMethod:       aws.String(d.Get("ssl_support_method").(string)),
	}
	if d.Get("certificate_id") == "" {
		viewerCertificate.CloudFrontDefaultCertificate = aws.Bool(true)
	} else {
		viewerCertificate.IAMCertificateId = aws.String(d.Get("certificate_id").(string))
	}

	// PUT DistributionConfig requires, unlike POST, EVERY possible option to be set.
	// Except for the configurable options, these are the defaults options.
	return &cloudfront.DistributionConfig{
		CallerReference:   callerReference,
		Enabled:           aws.Bool(d.Get("enabled").(bool)),
		Comment:           aws.String(d.Get("comment").(string)),
		PriceClass:        aws.String(d.Get("price_class").(string)),
		DefaultRootObject: aws.String(d.Get("default_root_object").(string)),
		Aliases: &cloudfront.Aliases{
			Quantity: aws.Int64(int64(len(aliases))),
			Items:    aliases,
		},
		Origins: &cloudfront.Origins{
			Quantity: aws.Int64(1),
			Items: []*cloudfront.Origin{
				&cloudfront.Origin{
					DomainName: aws.String(d.Get("origin_domain_name").(string)),
					Id:         aws.String(originId),
					OriginPath: aws.String(d.Get("origin_path").(string)),
					CustomOriginConfig: &cloudfront.CustomOriginConfig{
						HTTPPort:             aws.Int64(int64(d.Get("origin_http_port").(int))),
						HTTPSPort:            aws.Int64(int64(d.Get("origin_https_port").(int))),
						OriginProtocolPolicy: aws.String(d.Get("origin_protocol_policy").(string)),
					},
				},
			},
		},
		ViewerCertificate: viewerCertificate,
		Logging: &cloudfront.LoggingConfig{
			Enabled:        aws.Bool(d.Get("logging_enabled").(bool)),
			IncludeCookies: aws.Bool(d.Get("logging_include_cookies").(bool)),
			Prefix:         aws.String(d.Get("logging_prefix").(string)),
			Bucket:         aws.String(d.Get("logging_bucket").(string)),
		},
		Restrictions: &cloudfront.Restrictions{
			GeoRestriction: &cloudfront.GeoRestriction{
				Quantity:        aws.Int64(int64(len(geoRestrictions))),
				RestrictionType: aws.String(d.Get("geo_restriction_type").(string)),
				Items:           geoRestrictions,
			},
		},
		DefaultCacheBehavior: &cloudfront.DefaultCacheBehavior{
			ForwardedValues: &cloudfront.ForwardedValues{
				Cookies:     cookies,
				QueryString: aws.Bool(d.Get("forward_query_string").(bool)),
				Headers: &cloudfront.Headers{
					Quantity: aws.Int64(int64(len(forwardedHeaders))),
					Items:    forwardedHeaders,
				},
			},
			TargetOriginId:       aws.String(originId),
			ViewerProtocolPolicy: aws.String(d.Get("viewer_protocol_policy").(string)),
			MinTTL:               aws.Int64(int64(d.Get("minimum_ttl").(int))),
			MaxTTL:               aws.Int64(int64(d.Get("maximum_ttl").(int))),
			DefaultTTL:           aws.Int64(int64(d.Get("default_ttl").(int))),
			TrustedSigners: &cloudfront.TrustedSigners{
				Enabled:  aws.Bool(false),
				Quantity: aws.Int64(0),
			},
			SmoothStreaming: aws.Bool(d.Get("smooth_streaming").(bool)),
			AllowedMethods: &cloudfront.AllowedMethods{
				Quantity: aws.Int64(int64(len(allowedMethods))),
				Items:    allowedMethods,
				CachedMethods: &cloudfront.CachedMethods{
					Quantity: aws.Int64(int64(len(cachedMethods))),
					Items:    cachedMethods,
				},
			},
		},
		CacheBehaviors: &cloudfront.CacheBehaviors{
			Quantity: aws.Int64(0),
		},
		CustomErrorResponses: &cloudfront.CustomErrorResponses{
			Quantity: aws.Int64(0),
		},
	}, nil
}

// resourceAwsCloudFrontWebDistributionWaitUntilDeployed blocks until the
// distribution is deployed. It currently takes exactly 15 minutes to deploy
// but that might change in the future.
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

func resourceAwsCloudFrontWebDistributionDistributionRetrieve(
	d *schema.ResourceData, meta interface{}) (*cloudfront.GetDistributionOutput, error) {
	cloudfrontconn := meta.(*AWSClient).cloudfrontconn

	req := &cloudfront.GetDistributionInput{
		Id: aws.String(d.Id()),
	}

	res, err := cloudfrontconn.GetDistribution(req)
	if err != nil {
		return nil, fmt.Errorf("Error retrieving CloudFront distribution: %s", err)
	}

	return res, nil
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

func resourceAwsCloudFrontWebDistributionHandleMethods(in interface{}) []*string {
	// Terraform schemas does not currently support arrays as default values
	if len(in.([]interface{})) == 0 {
		return []*string{
			aws.String("GET"),
			aws.String("HEAD"),
		}
	}

	return resourceAwsCloudFrontWebDistributionAwsStringLists(in)
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
			Quantity: aws.Int64(int64(len(whitelist))),
			Items:    whitelist,
		},
	}
}

func resourceAwsCloudFrontCopyItems(d []*string) []string {
	list := make([]string, 0, len(d))
	for _, item := range d {
		list = append(list, *item)
	}
	return list
}
