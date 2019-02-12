package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/pinpoint"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsPinpointApp() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsPinpointAppCreate,
		Read:   resourceAwsPinpointAppRead,
		Update: resourceAwsPinpointAppUpdate,
		Delete: resourceAwsPinpointAppDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name_prefix"},
			},
			"name_prefix": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"application_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			//"cloudwatch_metrics_enabled": {
			//	Type:     schema.TypeBool,
			//	Optional: true,
			//	Default:  false,
			//},
			"campaign_hook": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					if old == "1" && new == "0" {
						return true
					}
					return false
				},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"lambda_function_name": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"mode": {
							Type:     schema.TypeString,
							Optional: true,
							ValidateFunc: validation.StringInSlice([]string{
								pinpoint.ModeDelivery,
								pinpoint.ModeFilter,
							}, false),
						},
						"web_url": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"limits": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					if old == "1" && new == "0" {
						return true
					}
					return false
				},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"daily": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"maximum_duration": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"messages_per_second": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"total": {
							Type:     schema.TypeInt,
							Optional: true,
						},
					},
				},
			},
			"quiet_time": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					if old == "1" && new == "0" {
						return true
					}
					return false
				},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"end": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"start": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
		},
	}
}

func resourceAwsPinpointAppCreate(d *schema.ResourceData, meta interface{}) error {
	pinpointconn := meta.(*AWSClient).pinpointconn

	var name string
	if v, ok := d.GetOk("name"); ok {
		name = v.(string)
	} else if v, ok := d.GetOk("name_prefix"); ok {
		name = resource.PrefixedUniqueId(v.(string))
	} else {
		name = resource.UniqueId()
	}

	log.Printf("[DEBUG] Pinpoint create app: %s", name)

	req := &pinpoint.CreateAppInput{
		CreateApplicationRequest: &pinpoint.CreateApplicationRequest{
			Name: aws.String(name),
		},
	}

	output, err := pinpointconn.CreateApp(req)
	if err != nil {
		return fmt.Errorf("error creating Pinpoint app: %s", err)
	}

	d.SetId(*output.ApplicationResponse.Id)

	return resourceAwsPinpointAppUpdate(d, meta)
}

func resourceAwsPinpointAppUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).pinpointconn

	appSettings := &pinpoint.WriteApplicationSettingsRequest{}

	//if d.HasChange("cloudwatch_metrics_enabled") {
	//	appSettings.CloudWatchMetricsEnabled = aws.Bool(d.Get("cloudwatch_metrics_enabled").(bool));
	//}

	if d.HasChange("campaign_hook") {
		appSettings.CampaignHook = expandPinpointCampaignHook(d.Get("campaign_hook").([]interface{}))
	}

	if d.HasChange("limits") {
		appSettings.Limits = expandPinpointCampaignLimits(d.Get("limits").([]interface{}))
	}

	if d.HasChange("quiet_time") {
		appSettings.QuietTime = expandPinpointQuietTime(d.Get("quiet_time").([]interface{}))
	}

	req := pinpoint.UpdateApplicationSettingsInput{
		ApplicationId:                   aws.String(d.Id()),
		WriteApplicationSettingsRequest: appSettings,
	}

	_, err := conn.UpdateApplicationSettings(&req)
	if err != nil {
		return err
	}

	return resourceAwsPinpointAppRead(d, meta)
}

func resourceAwsPinpointAppRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).pinpointconn

	log.Printf("[INFO] Reading Pinpoint App Attributes for %s", d.Id())

	app, err := conn.GetApp(&pinpoint.GetAppInput{
		ApplicationId: aws.String(d.Id()),
	})
	if err != nil {
		if isAWSErr(err, pinpoint.ErrCodeNotFoundException, "") {
			log.Printf("[WARN] Pinpoint App (%s) not found, error code (404)", d.Id())
			d.SetId("")
			return nil
		}

		return err
	}

	settings, err := conn.GetApplicationSettings(&pinpoint.GetApplicationSettingsInput{
		ApplicationId: aws.String(d.Id()),
	})
	if err != nil {
		if isAWSErr(err, pinpoint.ErrCodeNotFoundException, "") {
			log.Printf("[WARN] Pinpoint App (%s) not found, error code (404)", d.Id())
			d.SetId("")
			return nil
		}

		return err
	}

	d.Set("name", app.ApplicationResponse.Name)
	d.Set("application_id", app.ApplicationResponse.Id)

	if err := d.Set("campaign_hook", flattenPinpointCampaignHook(settings.ApplicationSettingsResource.CampaignHook)); err != nil {
		return fmt.Errorf("error setting campaign_hook: %s", err)
	}
	if err := d.Set("limits", flattenPinpointCampaignLimits(settings.ApplicationSettingsResource.Limits)); err != nil {
		return fmt.Errorf("error setting limits: %s", err)
	}
	if err := d.Set("quiet_time", flattenPinpointQuietTime(settings.ApplicationSettingsResource.QuietTime)); err != nil {
		return fmt.Errorf("error setting quiet_time: %s", err)
	}

	return nil
}

func resourceAwsPinpointAppDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).pinpointconn

	log.Printf("[DEBUG] Pinpoint Delete App: %s", d.Id())
	_, err := conn.DeleteApp(&pinpoint.DeleteAppInput{
		ApplicationId: aws.String(d.Id()),
	})

	if isAWSErr(err, pinpoint.ErrCodeNotFoundException, "") {
		return nil
	}

	if err != nil {
		return err
	}
	return nil
}

func expandPinpointCampaignHook(configs []interface{}) *pinpoint.CampaignHook {
	if len(configs) == 0 {
		return nil
	}

	m := configs[0].(map[string]interface{})

	ch := &pinpoint.CampaignHook{}

	if v, ok := m["lambda_function_name"]; ok {
		ch.LambdaFunctionName = aws.String(v.(string))
	}

	if v, ok := m["mode"]; ok {
		ch.Mode = aws.String(v.(string))
	}

	if v, ok := m["web_url"]; ok {
		ch.WebUrl = aws.String(v.(string))
	}

	return ch
}

func flattenPinpointCampaignHook(ch *pinpoint.CampaignHook) []interface{} {
	l := make([]interface{}, 0)

	m := map[string]interface{}{}

	m["lambda_function_name"] = aws.StringValue(ch.LambdaFunctionName)
	m["mode"] = aws.StringValue(ch.Mode)
	m["web_url"] = aws.StringValue(ch.WebUrl)

	l = append(l, m)

	return l
}

func expandPinpointCampaignLimits(configs []interface{}) *pinpoint.CampaignLimits {
	if len(configs) == 0 {
		return nil
	}

	m := configs[0].(map[string]interface{})

	cl := pinpoint.CampaignLimits{}

	if v, ok := m["daily"]; ok {
		cl.Daily = aws.Int64(int64(v.(int)))
	}

	if v, ok := m["maximum_duration"]; ok {
		cl.MaximumDuration = aws.Int64(int64(v.(int)))
	}

	if v, ok := m["messages_per_second"]; ok {
		cl.MessagesPerSecond = aws.Int64(int64(v.(int)))
	}

	if v, ok := m["total"]; ok {
		cl.Total = aws.Int64(int64(v.(int)))
	}

	return &cl
}

func flattenPinpointCampaignLimits(cl *pinpoint.CampaignLimits) []interface{} {
	l := make([]interface{}, 0)

	m := map[string]interface{}{}

	m["daily"] = aws.Int64Value(cl.Daily)
	m["maximum_duration"] = aws.Int64Value(cl.MaximumDuration)
	m["messages_per_second"] = aws.Int64Value(cl.MessagesPerSecond)
	m["total"] = aws.Int64Value(cl.Total)

	l = append(l, m)

	return l
}

func expandPinpointQuietTime(configs []interface{}) *pinpoint.QuietTime {
	if len(configs) == 0 {
		return nil
	}

	m := configs[0].(map[string]interface{})

	qt := pinpoint.QuietTime{}

	if v, ok := m["end"]; ok {
		qt.End = aws.String(v.(string))
	}

	if v, ok := m["start"]; ok {
		qt.Start = aws.String(v.(string))
	}

	return &qt
}

func flattenPinpointQuietTime(qt *pinpoint.QuietTime) []interface{} {
	l := make([]interface{}, 0)

	m := map[string]interface{}{}

	m["end"] = aws.StringValue(qt.End)
	m["start"] = aws.StringValue(qt.Start)

	l = append(l, m)

	return l
}
