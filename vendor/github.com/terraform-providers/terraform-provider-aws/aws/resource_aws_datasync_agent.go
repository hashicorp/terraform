package aws

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/datasync"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsDataSyncAgent() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDataSyncAgentCreate,
		Read:   resourceAwsDataSyncAgentRead,
		Update: resourceAwsDataSyncAgentUpdate,
		Delete: resourceAwsDataSyncAgentDelete,
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
				ConflictsWith: []string{"ip_address"},
			},
			"ip_address": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"activation_key"},
			},
			"name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"tags": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func resourceAwsDataSyncAgentCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).datasyncconn
	region := meta.(*AWSClient).region

	activationKey := d.Get("activation_key").(string)
	agentIpAddress := d.Get("ip_address").(string)

	// Perform one time fetch of activation key from gateway IP address
	if activationKey == "" {
		if agentIpAddress == "" {
			return fmt.Errorf("either activation_key or ip_address must be provided")
		}

		client := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
			Timeout: time.Second * 10,
		}

		requestURL := fmt.Sprintf("http://%s/?gatewayType=SYNC&activationRegion=%s", agentIpAddress, region)
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
			return fmt.Errorf("error retrieving activation key from IP Address (%s): %s", agentIpAddress, err)
		}
		if activationKey == "" {
			return fmt.Errorf("empty activationKey received from IP Address: %s", agentIpAddress)
		}
	}

	input := &datasync.CreateAgentInput{
		ActivationKey: aws.String(activationKey),
		Tags:          expandDataSyncTagListEntry(d.Get("tags").(map[string]interface{})),
	}

	if v, ok := d.GetOk("name"); ok {
		input.AgentName = aws.String(v.(string))
	}

	log.Printf("[DEBUG] Creating DataSync Agent: %s", input)
	output, err := conn.CreateAgent(input)
	if err != nil {
		return fmt.Errorf("error creating DataSync Agent: %s", err)
	}

	d.SetId(aws.StringValue(output.AgentArn))

	// Agent activations can take a few minutes
	err = resource.Retry(d.Timeout(schema.TimeoutCreate), func() *resource.RetryError {
		_, err := conn.DescribeAgent(&datasync.DescribeAgentInput{
			AgentArn: aws.String(d.Id()),
		})

		if isAWSErr(err, "InvalidRequestException", "not found") {
			return resource.RetryableError(err)
		}

		if err != nil {
			return resource.NonRetryableError(err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("error waiting for DataSync Agent (%s) creation: %s", d.Id(), err)
	}

	return resourceAwsDataSyncAgentRead(d, meta)
}

func resourceAwsDataSyncAgentRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).datasyncconn

	input := &datasync.DescribeAgentInput{
		AgentArn: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Reading DataSync Agent: %s", input)
	output, err := conn.DescribeAgent(input)

	if isAWSErr(err, "InvalidRequestException", "not found") {
		log.Printf("[WARN] DataSync Agent %q not found - removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("error reading DataSync Agent (%s): %s", d.Id(), err)
	}

	tagsInput := &datasync.ListTagsForResourceInput{
		ResourceArn: output.AgentArn,
	}

	log.Printf("[DEBUG] Reading DataSync Agent tags: %s", tagsInput)
	tagsOutput, err := conn.ListTagsForResource(tagsInput)

	if err != nil {
		return fmt.Errorf("error reading DataSync Agent (%s) tags: %s", d.Id(), err)
	}

	d.Set("arn", output.AgentArn)
	d.Set("name", output.Name)

	if err := d.Set("tags", flattenDataSyncTagListEntry(tagsOutput.Tags)); err != nil {
		return fmt.Errorf("error setting tags: %s", err)
	}

	return nil
}

func resourceAwsDataSyncAgentUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).datasyncconn

	if d.HasChange("name") {
		input := &datasync.UpdateAgentInput{
			AgentArn: aws.String(d.Id()),
			Name:     aws.String(d.Get("name").(string)),
		}

		log.Printf("[DEBUG] Updating DataSync Agent: %s", input)
		_, err := conn.UpdateAgent(input)
		if err != nil {
			return fmt.Errorf("error updating DataSync Agent (%s): %s", d.Id(), err)
		}
	}

	if d.HasChange("tags") {
		oldRaw, newRaw := d.GetChange("tags")
		createTags, removeTags := dataSyncTagsDiff(expandDataSyncTagListEntry(oldRaw.(map[string]interface{})), expandDataSyncTagListEntry(newRaw.(map[string]interface{})))

		if len(removeTags) > 0 {
			input := &datasync.UntagResourceInput{
				Keys:        dataSyncTagsKeys(removeTags),
				ResourceArn: aws.String(d.Id()),
			}

			log.Printf("[DEBUG] Untagging DataSync Agent: %s", input)
			if _, err := conn.UntagResource(input); err != nil {
				return fmt.Errorf("error untagging DataSync Agent (%s): %s", d.Id(), err)
			}
		}

		if len(createTags) > 0 {
			input := &datasync.TagResourceInput{
				ResourceArn: aws.String(d.Id()),
				Tags:        createTags,
			}

			log.Printf("[DEBUG] Tagging DataSync Agent: %s", input)
			if _, err := conn.TagResource(input); err != nil {
				return fmt.Errorf("error tagging DataSync Agent (%s): %s", d.Id(), err)
			}
		}
	}

	return resourceAwsDataSyncAgentRead(d, meta)
}

func resourceAwsDataSyncAgentDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).datasyncconn

	input := &datasync.DeleteAgentInput{
		AgentArn: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Deleting DataSync Agent: %s", input)
	_, err := conn.DeleteAgent(input)

	if isAWSErr(err, "InvalidRequestException", "not found") {
		return nil
	}

	if err != nil {
		return fmt.Errorf("error deleting DataSync Agent (%s): %s", d.Id(), err)
	}

	return nil
}
