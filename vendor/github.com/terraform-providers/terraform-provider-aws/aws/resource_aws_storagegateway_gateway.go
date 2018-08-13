package aws

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/storagegateway"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsStorageGatewayGateway() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsStorageGatewayGatewayCreate,
		Read:   resourceAwsStorageGatewayGatewayRead,
		Update: resourceAwsStorageGatewayGatewayUpdate,
		Delete: resourceAwsStorageGatewayGatewayDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"activation_key": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"gateway_ip_address"},
			},
			"gateway_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"gateway_ip_address": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"activation_key"},
			},
			"gateway_name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
			},
			"gateway_timezone": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
			},
			"gateway_type": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "STORED",
				ValidateFunc: validation.StringInSlice([]string{
					"CACHED",
					"FILE_S3",
					"STORED",
					"VTL",
				}, false),
			},
			"medium_changer_type": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					"AWS-Gateway-VTL",
					"STK-L700",
				}, false),
			},
			"tape_drive_type": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					"IBM-ULT3580-TD5",
				}, false),
			},
		},
	}
}

func resourceAwsStorageGatewayGatewayCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).storagegatewayconn
	region := meta.(*AWSClient).region

	activationKey := d.Get("activation_key").(string)
	gatewayIpAddress := d.Get("gateway_ip_address").(string)

	// Perform one time fetch of activation key from gateway IP address
	if activationKey == "" {
		if gatewayIpAddress == "" {
			return fmt.Errorf("either activation_key or gateway_ip_address must be provided")
		}

		client := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
			Timeout: time.Second * 10,
		}

		requestURL := fmt.Sprintf("http://%s/?activationRegion=%s", gatewayIpAddress, region)
		log.Printf("[DEBUG] Creating HTTP request: %s", requestURL)
		request, err := http.NewRequest("GET", requestURL, nil)
		if err != nil {
			return fmt.Errorf("error creating HTTP request: %s", err)
		}

		err = resource.Retry(d.Timeout(schema.TimeoutCreate), func() *resource.RetryError {
			log.Printf("[DEBUG] Making HTTP request: %s", request.URL.String())
			response, err := client.Do(request)
			if err != nil {
				if err, ok := err.(net.Error); ok {
					errMessage := fmt.Errorf("error making HTTP request: %s", err)
					log.Printf("[DEBUG] retryable %s", errMessage)
					return resource.RetryableError(errMessage)
				}
				return resource.NonRetryableError(fmt.Errorf("error making HTTP request: %s", err))
			}

			log.Printf("[DEBUG] Received HTTP response: %#v", response)
			if response.StatusCode != 302 {
				return resource.NonRetryableError(fmt.Errorf("expected HTTP status code 302, received: %d", response.StatusCode))
			}

			redirectURL, err := response.Location()
			if err != nil {
				return resource.NonRetryableError(fmt.Errorf("error extracting HTTP Location header: %s", err))
			}

			activationKey = redirectURL.Query().Get("activationKey")

			return nil
		})
		if err != nil {
			return fmt.Errorf("error retrieving activation key from IP Address (%s): %s", gatewayIpAddress, err)
		}
		if activationKey == "" {
			return fmt.Errorf("empty activationKey received from IP Address: %s", gatewayIpAddress)
		}
	}

	input := &storagegateway.ActivateGatewayInput{
		ActivationKey:   aws.String(activationKey),
		GatewayRegion:   aws.String(region),
		GatewayName:     aws.String(d.Get("gateway_name").(string)),
		GatewayTimezone: aws.String(d.Get("gateway_timezone").(string)),
		GatewayType:     aws.String(d.Get("gateway_type").(string)),
	}

	if v, ok := d.GetOk("medium_changer_type"); ok {
		input.MediumChangerType = aws.String(v.(string))
	}

	if v, ok := d.GetOk("tape_drive_type"); ok {
		input.TapeDriveType = aws.String(v.(string))
	}

	log.Printf("[DEBUG] Activating Storage Gateway Gateway: %s", input)
	output, err := conn.ActivateGateway(input)
	if err != nil {
		return fmt.Errorf("error activating Storage Gateway Gateway: %s", err)
	}

	d.SetId(aws.StringValue(output.GatewayARN))

	// Gateway activations can take a few minutes
	err = resource.Retry(d.Timeout(schema.TimeoutCreate), func() *resource.RetryError {
		_, err := conn.DescribeGatewayInformation(&storagegateway.DescribeGatewayInformationInput{
			GatewayARN: aws.String(d.Id()),
		})
		if err != nil {
			if isAWSErr(err, storagegateway.ErrorCodeGatewayNotConnected, "") {
				return resource.RetryableError(err)
			}
			if isAWSErr(err, storagegateway.ErrCodeInvalidGatewayRequestException, "") {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("error waiting for Storage Gateway Gateway activation: %s", err)
	}

	return resourceAwsStorageGatewayGatewayRead(d, meta)
}

func resourceAwsStorageGatewayGatewayRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).storagegatewayconn

	input := &storagegateway.DescribeGatewayInformationInput{
		GatewayARN: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Reading Storage Gateway Gateway: %s", input)
	output, err := conn.DescribeGatewayInformation(input)
	if err != nil {
		if isAWSErrStorageGatewayGatewayNotFound(err) {
			log.Printf("[WARN] Storage Gateway Gateway %q not found - removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("error reading Storage Gateway Gateway: %s", err)
	}

	// The Storage Gateway API currently provides no way to read this value
	d.Set("activation_key", d.Get("activation_key").(string))
	d.Set("arn", output.GatewayARN)
	d.Set("gateway_id", output.GatewayId)
	// The Storage Gateway API currently provides no way to read this value
	d.Set("gateway_ip_address", d.Get("gateway_ip_address").(string))
	d.Set("gateway_name", output.GatewayName)
	d.Set("gateway_timezone", output.GatewayTimezone)
	d.Set("gateway_type", output.GatewayType)
	// The Storage Gateway API currently provides no way to read this value
	d.Set("medium_changer_type", d.Get("medium_changer_type").(string))
	// The Storage Gateway API currently provides no way to read this value
	d.Set("tape_drive_type", d.Get("tape_drive_type").(string))

	return nil
}

func resourceAwsStorageGatewayGatewayUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).storagegatewayconn

	input := &storagegateway.UpdateGatewayInformationInput{
		GatewayARN:      aws.String(d.Id()),
		GatewayName:     aws.String(d.Get("gateway_name").(string)),
		GatewayTimezone: aws.String(d.Get("gateway_timezone").(string)),
	}

	log.Printf("[DEBUG] Updating Storage Gateway Gateway: %s", input)
	_, err := conn.UpdateGatewayInformation(input)
	if err != nil {
		return fmt.Errorf("error updating Storage Gateway Gateway: %s", err)
	}

	return resourceAwsStorageGatewayGatewayRead(d, meta)
}

func resourceAwsStorageGatewayGatewayDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).storagegatewayconn

	input := &storagegateway.DeleteGatewayInput{
		GatewayARN: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Deleting Storage Gateway Gateway: %s", input)
	_, err := conn.DeleteGateway(input)
	if err != nil {
		if isAWSErrStorageGatewayGatewayNotFound(err) {
			return nil
		}
		return fmt.Errorf("error deleting Storage Gateway Gateway: %s", err)
	}

	return nil
}

// The API returns multiple responses for a missing gateway
func isAWSErrStorageGatewayGatewayNotFound(err error) bool {
	if isAWSErr(err, storagegateway.ErrCodeInvalidGatewayRequestException, "The specified gateway was not found.") {
		return true
	}
	if isAWSErr(err, storagegateway.ErrorCodeGatewayNotFound, "") {
		return true
	}
	return false
}
