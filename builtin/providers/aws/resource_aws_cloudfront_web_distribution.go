package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	//"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudfront"
)

const resourceAwsCloudFrontS3OriginSuffix = ".s3.amazonaws.com"

func resourceAwsCloudFrontWebDistribution() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsCloudFrontWebDistributionCreate,
		Read:   resourceAwsCloudFrontWebDistributionRead,
		Update: resourceAwsCloudFrontWebDistributionUpdate,
		Delete: resourceAwsCloudFrontWebDistributionDelete,

		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"etag": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"domain_name": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"aliases": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"comment": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "Managed by Terraform",
			},

			"allowed_http_methods": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"cached_http_methods": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"default_ttl": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  86400,
			},

			"forward_cookies": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"forward_cookie_names": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"forward_header_names": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"forward_query_strings": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"max_ttl": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  31536000,
			},

			"min_ttl": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  0,
			},

			"enable_smooth_streaming": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"target_origin_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"viewer_protocol_policy": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "allow-all",
			},

			"default_root_object": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},

			"enabled": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},

			"price_class": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"custom_origin": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"domain_name": &schema.Schema{
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
							Default:  "match-viewer",
						},
					},
				},
			},

			"s3_origin": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"bucket_name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"origin_path": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"origin_access_identity": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func resourceAwsCloudFrontWebDistributionCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AWSClient).cloudfrontconn

	config, err := resourceAwsCloudFrontWebDistributionConfig(d)
	if err != nil {
		return err
	}

	req := &cloudfront.CreateDistributionInput{
		DistributionConfig: config,
	}

	res, err := client.CreateDistribution(req)
	if err != nil {
		return err
	}

	d.Set("etag", *res.ETag)
	d.Set("domain_name", *res.Distribution.DomainName)
	d.Set("id", *res.Distribution.Id)
	d.SetId(*res.Distribution.Id)

	resourceAwsCloudFrontWebDistributionWaitForDeploy(*res.Distribution.Id, true, client)

	return resourceAwsCloudFrontWebDistributionRead(d, meta)
}

func resourceAwsCloudFrontWebDistributionRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AWSClient).cloudfrontconn

	id := d.Id()

	req := &cloudfront.GetDistributionInput{
		Id: &id,
	}
	res, err := client.GetDistribution(req)
	if err != nil {
		return err
	}

	d.Set("etag", *res.ETag)

	distro := res.Distribution
	d.Set("domain_name", *distro.DomainName)
	d.Set("id", *distro.Id)
	d.SetId(*distro.Id)

	config := distro.DistributionConfig

	d.Set("aliases", resourceAwsCloudFrontWebDistributionUnpackStringList(config.Aliases.Items))
	d.Set("comment", *config.Comment)
	d.Set("default_root_object", *config.DefaultRootObject)
	d.Set("enabled", *config.Enabled)
	d.Set("price_class", *config.PriceClass)

	dcb := config.DefaultCacheBehavior
	// FIXME: This ones are causing a nil pointer dereference inside the
	// schema.Set code? Need to debug some more...
	//d.Set("allowed_http_methods", resourceAwsCloudFrontWebDistributionUnpackStringList(dcb.AllowedMethods.Items))
	//d.Set("cached_http_methods", resourceAwsCloudFrontWebDistributionUnpackStringList(dcb.AllowedMethods.CachedMethods.Items))
	d.Set("default_ttl", *dcb.DefaultTTL)

	cookiePref := dcb.ForwardedValues.Cookies
	d.Set("forward_cookies", *cookiePref.Forward != "none")
	if cookiePref.WhitelistedNames != nil {
		d.Set("forward_cookie_names", resourceAwsCloudFrontWebDistributionUnpackStringList(cookiePref.WhitelistedNames.Items))
	} else {
		d.Set("forward_cookie_names", []interface{}{})
	}

	if dcb.ForwardedValues.Headers != nil {
		d.Set("forward_header_names", resourceAwsCloudFrontWebDistributionUnpackStringList(dcb.ForwardedValues.Headers.Items))
	} else {
		d.Set("forward_header_names", []interface{}{})
	}

	d.Set("forward_query_strings", *dcb.ForwardedValues.QueryString)

	d.Set("max_ttl", *dcb.MaxTTL)
	d.Set("min_ttl", *dcb.MinTTL)
	d.Set("enable_smooth_streaming", *dcb.SmoothStreaming)
	d.Set("target_origin_id", *dcb.TargetOriginId)
	d.Set("viewer_protocol_policy", *dcb.ViewerProtocolPolicy)

	customOrigins := make([]interface{}, 0, len(config.Origins.Items))
	s3Origins := make([]interface{}, 0, len(config.Origins.Items))
	for _, origin := range config.Origins.Items {
		originM := map[string]interface{}{}
		originM["id"] = *origin.Id
		originM["origin_path"] = *origin.OriginPath
		if custom := origin.CustomOriginConfig; custom != nil {
			originM["domain_name"] = *origin.DomainName
			originM["http_port"] = int(*custom.HTTPPort)
			originM["https_port"] = int(*custom.HTTPSPort)
			originM["origin_protocol_policy"] = *custom.OriginProtocolPolicy
			customOrigins = append(customOrigins, originM)
		} else if s3 := origin.S3OriginConfig; s3 != nil {
			domainName := *origin.DomainName
			originM["bucket_name"] = domainName[:len(domainName)-len(*origin.DomainName)]
			originM["origin_access_identity"] = *s3.OriginAccessIdentity
			s3Origins = append(s3Origins, originM)
		} else {
			return fmt.Errorf("Unknown origin type for %s", *origin.Id)
		}
	}
	// FIXME: These crash with a nil pointer dereference. Why?
	//d.Set("custom_origin", customOrigins)
	//d.Set("s3_origin", s3Origins)

	return nil
}

func resourceAwsCloudFrontWebDistributionUpdate(d *schema.ResourceData, meta interface{}) error {
	//client := meta.(*AWSClient).cloudfrontconn
	return nil
}

func resourceAwsCloudFrontWebDistributionDelete(d *schema.ResourceData, meta interface{}) error {
	//client := meta.(*AWSClient).cloudfrontconn
	return nil
}

func resourceAwsCloudFrontWebDistributionConfig(d *schema.ResourceData) (*cloudfront.DistributionConfig, error) {

	awsStringList := func(in []interface{}) []*string {
		out := make([]*string, len(in))
		for i, valI := range in {
			out[i] = aws.String(valI.(string))
		}
		return out
	}

	aliases := d.Get("aliases").(*schema.Set).List()
	allowedHTTPMethods := d.Get("allowed_http_methods").(*schema.Set).List()
	cachedHTTPMethods := d.Get("cached_http_methods").(*schema.Set).List()
	forwardCookieNames := d.Get("forward_cookie_names").(*schema.Set).List()
	forwardHeaderNames := d.Get("forward_header_names").(*schema.Set).List()

	cookiesEnabled := d.Get("forward_cookies").(bool)

	var cookiesMode string
	if cookiesEnabled {
		if len(forwardCookieNames) > 0 {
			cookiesMode = "whitelist"
		} else {
			cookiesMode = "all"
		}
	} else {
		if len(forwardCookieNames) > 0 {
			return nil, fmt.Errorf("can't use forward_cookie_names when forward_cookies is not set")
		} else {
			cookiesMode = "none"
		}
	}

	config := &cloudfront.DistributionConfig{
		CallerReference: aws.String(time.Now().Format(time.RFC3339Nano)),
		Aliases: &cloudfront.Aliases{
			Items:    awsStringList(aliases),
			Quantity: aws.Int64(int64(len(aliases))),
		},
		Comment: aws.String(d.Get("comment").(string)),
		DefaultCacheBehavior: &cloudfront.DefaultCacheBehavior{
			AllowedMethods: &cloudfront.AllowedMethods{
				CachedMethods: &cloudfront.CachedMethods{
					Items:    awsStringList(cachedHTTPMethods),
					Quantity: aws.Int64(int64(len(cachedHTTPMethods))),
				},
				Items:    awsStringList(allowedHTTPMethods),
				Quantity: aws.Int64(int64(len(allowedHTTPMethods))),
			},
			DefaultTTL: aws.Int64(int64(d.Get("default_ttl").(int))),
			ForwardedValues: &cloudfront.ForwardedValues{
				Cookies: &cloudfront.CookiePreference{
					Forward: aws.String(cookiesMode),
					WhitelistedNames: &cloudfront.CookieNames{
						Items:    awsStringList(forwardCookieNames),
						Quantity: aws.Int64(int64(len(forwardCookieNames))),
					},
				},
				Headers: &cloudfront.Headers{
					Items:    awsStringList(forwardHeaderNames),
					Quantity: aws.Int64(int64(len(forwardHeaderNames))),
				},
				QueryString: aws.Bool(d.Get("forward_query_strings").(bool)),
			},
			MaxTTL:          aws.Int64(int64(d.Get("max_ttl").(int))),
			MinTTL:          aws.Int64(int64(d.Get("min_ttl").(int))),
			SmoothStreaming: aws.Bool(d.Get("enable_smooth_streaming").(bool)),
			TargetOriginId:  aws.String(d.Get("target_origin_id").(string)),
			TrustedSigners: &cloudfront.TrustedSigners{
				Enabled:  aws.Bool(false),
				Items:    []*string{},
				Quantity: aws.Int64(0),
			},
			ViewerProtocolPolicy: aws.String(d.Get("viewer_protocol_policy").(string)),
		},
		DefaultRootObject: aws.String(d.Get("default_root_object").(string)),
		Enabled:           aws.Bool(d.Get("enabled").(bool)),
		PriceClass:        aws.String(d.Get("price_class").(string)),
	}

	s3Origins := d.Get("s3_origin").(*schema.Set).List()
	customOrigins := d.Get("custom_origin").(*schema.Set).List()
	origins := make([]*cloudfront.Origin, 0, len(s3Origins)+len(customOrigins))

	if cap(origins) == 0 {
		return nil, fmt.Errorf("web distribution must have at least one origin")
	}

	for _, originI := range s3Origins {
		originM := originI.(map[string]interface{})
		origin := &cloudfront.Origin{
			DomainName: aws.String(originM["bucket_name"].(string) + resourceAwsCloudFrontS3OriginSuffix),
			Id:         aws.String(originM["id"].(string)),
			OriginPath: aws.String(originM["origin_path"].(string)),
			S3OriginConfig: &cloudfront.S3OriginConfig{
				OriginAccessIdentity: aws.String(originM["origin_access_identity"].(string)),
			},
		}
		origins = append(origins, origin)
	}

	for _, originI := range customOrigins {
		originM := originI.(map[string]interface{})
		origin := &cloudfront.Origin{
			DomainName: aws.String(originM["domain_name"].(string)),
			Id:         aws.String(originM["id"].(string)),
			OriginPath: aws.String(originM["origin_path"].(string)),
			CustomOriginConfig: &cloudfront.CustomOriginConfig{
				HTTPPort:             aws.Int64(int64(originM["http_port"].(int))),
				HTTPSPort:            aws.Int64(int64(originM["https_port"].(int))),
				OriginProtocolPolicy: aws.String(originM["origin_protocol_policy"].(string)),
			},
		}
		origins = append(origins, origin)
	}

	config.Origins = &cloudfront.Origins{
		Items:    origins,
		Quantity: aws.Int64(int64(len(origins))),
	}

	return config, nil
}

func resourceAwsCloudFrontWebDistributionUnpackStringList(in []*string) []interface{} {
	ret := make([]interface{}, len(in))
	for i, v := range in {
		ret[i] = *v
	}
	return ret
}

func resourceAwsCloudFrontWebDistributionWaitForDeploy(id string, enabled bool, client *cloudfront.CloudFront) error {
	log.Printf("Waiting for CloudFront Web Distribution %s to be deployed", id)

	req := &cloudfront.GetDistributionInput{
		Id: &id,
	}

	for {
		res, err := client.GetDistribution(req)
		if err != nil {
			return err
		}

		status := *res.Distribution.Status

		if status == "Deployed" {
			// Fail if the enabled status is not what we expect. This
			// suggests that someone messed with the object via some
			// other client while the deploy was in progress, so the
			// object is likely not in the state we need it to be in
			// to continue.
			if *res.Distribution.DistributionConfig.Enabled != enabled {
				return fmt.Errorf(
					"enabled state is %v but I was expecting %v",
					*res.Distribution.DistributionConfig.Enabled,
					enabled,
				)
			}
			break
		}

		time.Sleep(4 * time.Second)
	}

	return nil
}
