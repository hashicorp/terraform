package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/awslabs/aws-sdk-go/aws"
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

			"origin_domain_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"origin_path": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
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

			"forward_query_string": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
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

			"minimum_ttl": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},

			"aliases": &schema.Schema{
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
				Set: func(v interface{}) int {
					return hashcode.String(v.(string))
				},
			},

			"geo_restriction_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "none",
			},

			"geo_restrictions": &schema.Schema{
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
				Set: func(v interface{}) int {
					return hashcode.String(v.(string))
				},
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

	res, serr := cloudfrontconn.CreateDistribution(&cloudfront.CreateDistributionInput{
		DistributionConfig: c,
	})

	aerr := aws.Error(serr)
	if aerr != nil {
		return fmt.Errorf("Error creating CloudFront distribution: %s", aerr)
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
		return err
	}

	c := v.Distribution.DistributionConfig
	d.Set("domain_name", v.Distribution.DomainName)
	d.Set("status", v.Distribution.Status)
	d.Set("enabled", c.Enabled)
	d.Set("comment", c.Comment)
	d.Set("price_class", c.PriceClass)
	d.Set("default_root_object", c.DefaultRootObject)
	d.Set("aliases", c.Aliases.Items)
	d.Set("viewer_protocol_policy", c.DefaultCacheBehavior.ViewerProtocolPolicy)
	d.Set("forward_cookie", c.DefaultCacheBehavior.ForwardedValues.Cookies)
	d.Set("forward_query_string", c.DefaultCacheBehavior.ForwardedValues.QueryString)
	d.Set("logging_enabled", c.Logging.Enabled)
	d.Set("logging_include_cookies", c.Logging.IncludeCookies)
	d.Set("logging_prefix", c.Logging.Prefix)
	d.Set("logging_bucket", c.Logging.Bucket)
	d.Set("minimum_ttl", c.DefaultCacheBehavior.MinTTL)
	d.Set("geo_restriction_type", c.Restrictions.GeoRestriction.RestrictionType)
	d.Set("geo_restrictions", c.Restrictions.GeoRestriction.Items)
	d.Set("zone_id", "Z2FDTNDATAQYW2")

	// CloudFront distributions supports multiple origins. However most of the above
	// configuration options also apply to a single origin which would result in
	// an overwhelming API
	o := c.Origins.Items[0]
	d.Set("origin_domain_name", o.DomainName)
	d.Set("origin_path", o.OriginPath)
	d.Set("http_port", o.CustomOriginConfig.HTTPPort)
	d.Set("https_port", o.CustomOriginConfig.HTTPSPort)
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
		ID:                 aws.String(string(d.Id())),
		IfMatch:            v.ETag,
	}

	_, serr := cloudfrontconn.UpdateDistribution(params)

	aerr := aws.Error(serr)
	if aerr != nil {
		return fmt.Errorf("Error updating CloudFront distribution: %s", aerr)
	}

	err = resourceAwsCloudFrontWebDistributionWaitUntilDeployed(d, meta)
	if err != nil {
		return err
	}

	return resourceAwsCloudFrontWebDistributionRead(d, meta)
}

func resourceAwsCloudFrontWebDistributionDelete(d *schema.ResourceData, meta interface{}) error {
	cloudfrontconn := meta.(*AWSClient).cloudfrontconn

	// TODO: Fail quietly if resource no longer exists?
	v, err := resourceAwsCloudFrontWebDistributionDistributionRetrieve(d, meta)
	if err != nil {
		return err
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

	_, serr := cloudfrontconn.DeleteDistribution(params)

	aerr := aws.Error(serr)
	if aerr != nil {
		return fmt.Errorf("Error deleting CloudFront distribution: %s", aerr)
	}

	return nil
}

func resourceAwsCloudFrontWebDistributionDistributionConfig(
	d *schema.ResourceData, meta interface{},
	callerReference *string) (*cloudfront.DistributionConfig, error) {

	enabled := d.Get("enabled").(bool)
	comment := d.Get("comment").(string)
	priceClass := d.Get("price_class").(string)
	defaultRootObject := d.Get("default_root_object").(string)
	originDomainName := d.Get("origin_domain_name").(string)
	originID := fmt.Sprintf("%s-origin", originDomainName)
	originPath := d.Get("origin_path").(string)
	originProtocolPolicy := d.Get("origin_protocol_policy").(string)
	originHTTPPort := d.Get("http_port").(int)
	originHTTPSPort := d.Get("https_port").(int)
	viewerProtocolPolicy := d.Get("viewer_protocol_policy").(string)
	forwardCookie := d.Get("forward_cookie").(string)
	forwardQueryString := d.Get("forward_query_string").(bool)
	loggingEnabled := d.Get("logging_enabled").(bool)
	loggingIncludeCookies := d.Get("logging_include_cookies").(bool)
	loggingPrefix := d.Get("logging_prefix").(string)
	loggingBucket := d.Get("logging_bucket").(string)
	minimumTTL := d.Get("minimum_ttl").(int)
	geoRestrictionType := d.Get("geo_restriction_type").(string)

	aliasesList := d.Get("aliases").(*schema.Set).List()
	aliases := make([]*string, 0, len(aliasesList))
	for _, r := range aliasesList {
		s := r.(string)
		aliases = append(aliases, aws.String(s))
	}

	geoRestrictionsList := d.Get("geo_restrictions").(*schema.Set).List()
	geoRestrictions := make([]*string, 0, len(geoRestrictionsList))
	for _, r := range geoRestrictionsList {
		s := r.(string)
		geoRestrictions = append(geoRestrictions, aws.String(s))
	}

	// PUT DistributionConfig requires, unlike POST, EVERY possible option to be set.
	// Except for the configurable options, these are the defaults options.
	return &cloudfront.DistributionConfig{
		CallerReference:   callerReference,
		Enabled:           aws.Boolean(enabled),
		Comment:           aws.String(comment),
		PriceClass:        aws.String(priceClass),
		DefaultRootObject: aws.String(defaultRootObject),
		Aliases: &cloudfront.Aliases{
			Quantity: aws.Long(int64(len(aliases))),
			Items:    aliases,
		},
		Origins: &cloudfront.Origins{
			Quantity: aws.Long(1),
			Items: []*cloudfront.Origin{
				&cloudfront.Origin{
					DomainName: aws.String(originDomainName),
					ID:         aws.String(originID),
					OriginPath: aws.String(originPath),
					CustomOriginConfig: &cloudfront.CustomOriginConfig{
						HTTPPort:             aws.Long(int64(originHTTPPort)),
						HTTPSPort:            aws.Long(int64(originHTTPSPort)),
						OriginProtocolPolicy: aws.String(originProtocolPolicy),
					},
				},
			},
		},
		ViewerCertificate: &cloudfront.ViewerCertificate{
			CloudFrontDefaultCertificate: aws.Boolean(true),
			MinimumProtocolVersion:       aws.String("SSLv3"),
		},
		Logging: &cloudfront.LoggingConfig{
			Enabled:        aws.Boolean(loggingEnabled),
			IncludeCookies: aws.Boolean(loggingIncludeCookies),
			Prefix:         aws.String(loggingPrefix),
			Bucket:         aws.String(loggingBucket),
		},
		Restrictions: &cloudfront.Restrictions{
			GeoRestriction: &cloudfront.GeoRestriction{
				Quantity:        aws.Long(int64(len(geoRestrictions))),
				RestrictionType: aws.String(geoRestrictionType),
				Items:           geoRestrictions,
			},
		},
		CacheBehaviors: &cloudfront.CacheBehaviors{
			Quantity: aws.Long(0),
		},
		CustomErrorResponses: &cloudfront.CustomErrorResponses{
			Quantity: aws.Long(0),
		},
		DefaultCacheBehavior: &cloudfront.DefaultCacheBehavior{
			ForwardedValues: &cloudfront.ForwardedValues{
				Cookies:     &cloudfront.CookiePreference{Forward: aws.String(forwardCookie)},
				QueryString: aws.Boolean(forwardQueryString),
				Headers: &cloudfront.Headers{
					Quantity: aws.Long(0),
				},
			},
			TargetOriginID:       aws.String(originID),
			ViewerProtocolPolicy: aws.String(viewerProtocolPolicy),
			MinTTL:               aws.Long(int64(minimumTTL)),
			TrustedSigners: &cloudfront.TrustedSigners{
				Enabled:  aws.Boolean(false),
				Quantity: aws.Long(0),
			},
			SmoothStreaming: aws.Boolean(false),
			AllowedMethods: &cloudfront.AllowedMethods{
				Quantity: aws.Long(2),
				Items: []*string{
					aws.String("GET"),
					aws.String("HEAD"),
				},
				CachedMethods: &cloudfront.CachedMethods{
					Quantity: aws.Long(2),
					Items: []*string{
						aws.String("GET"),
						aws.String("HEAD"),
					},
				},
			},
		},
	}, nil
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
