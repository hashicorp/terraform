package aws

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsServiceDiscoveryPublicDnsNamespace() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsServiceDiscoveryPublicDnsNamespaceCreate,
		Read:   resourceAwsServiceDiscoveryPublicDnsNamespaceRead,
		Delete: resourceAwsServiceDiscoveryPublicDnsNamespaceDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"hosted_zone": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsServiceDiscoveryPublicDnsNamespaceCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sdconn

	name := d.Get("name").(string)
	// The CreatorRequestId has a limit of 64 bytes
	var requestId string
	if len(name) > (64 - resource.UniqueIDSuffixLength) {
		requestId = resource.PrefixedUniqueId(name[0:(64 - resource.UniqueIDSuffixLength - 1)])
	} else {
		requestId = resource.PrefixedUniqueId(name)
	}

	input := &servicediscovery.CreatePublicDnsNamespaceInput{
		Name:             aws.String(name),
		CreatorRequestId: aws.String(requestId),
	}

	if v, ok := d.GetOk("description"); ok {
		input.Description = aws.String(v.(string))
	}

	resp, err := conn.CreatePublicDnsNamespace(input)
	if err != nil {
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending: []string{servicediscovery.OperationStatusSubmitted, servicediscovery.OperationStatusPending},
		Target:  []string{servicediscovery.OperationStatusSuccess},
		Refresh: servicediscoveryOperationRefreshStatusFunc(conn, *resp.OperationId),
		Timeout: 5 * time.Minute,
	}

	opresp, err := stateConf.WaitForState()
	if err != nil {
		return err
	}

	d.SetId(*opresp.(*servicediscovery.GetOperationOutput).Operation.Targets["NAMESPACE"])
	return resourceAwsServiceDiscoveryPublicDnsNamespaceRead(d, meta)
}

func resourceAwsServiceDiscoveryPublicDnsNamespaceRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sdconn

	input := &servicediscovery.GetNamespaceInput{
		Id: aws.String(d.Id()),
	}

	resp, err := conn.GetNamespace(input)
	if err != nil {
		if isAWSErr(err, servicediscovery.ErrCodeNamespaceNotFound, "") {
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("name", resp.Namespace.Name)
	d.Set("description", resp.Namespace.Description)
	d.Set("arn", resp.Namespace.Arn)
	if resp.Namespace.Properties != nil {
		d.Set("hosted_zone", resp.Namespace.Properties.DnsProperties.HostedZoneId)
	}
	return nil
}

func resourceAwsServiceDiscoveryPublicDnsNamespaceDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sdconn

	input := &servicediscovery.DeleteNamespaceInput{
		Id: aws.String(d.Id()),
	}

	resp, err := conn.DeleteNamespace(input)
	if err != nil {
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending: []string{servicediscovery.OperationStatusSubmitted, servicediscovery.OperationStatusPending},
		Target:  []string{servicediscovery.OperationStatusSuccess},
		Refresh: servicediscoveryOperationRefreshStatusFunc(conn, *resp.OperationId),
		Timeout: 5 * time.Minute,
	}

	_, err = stateConf.WaitForState()
	return err
}

func servicediscoveryOperationRefreshStatusFunc(conn *servicediscovery.ServiceDiscovery, oid string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		input := &servicediscovery.GetOperationInput{
			OperationId: aws.String(oid),
		}
		resp, err := conn.GetOperation(input)

		if err != nil {
			return nil, servicediscovery.OperationStatusFail, err
		}

		// Error messages can also be contained in the response with FAIL status
		//   "ErrorCode":"CANNOT_CREATE_HOSTED_ZONE",
		//   "ErrorMessage":"The VPC that you chose, vpc-xxx in region xxx, is already associated with another private hosted zone that has an overlapping name space, xxx.. (Service: AmazonRoute53; Status Code: 400; Error Code: ConflictingDomainExists; Request ID: xxx)"
		//   "Status":"FAIL",

		if aws.StringValue(resp.Operation.Status) == servicediscovery.OperationStatusFail {
			return resp, servicediscovery.OperationStatusFail, fmt.Errorf("%s: %s", aws.StringValue(resp.Operation.ErrorCode), aws.StringValue(resp.Operation.ErrorMessage))
		}

		return resp, aws.StringValue(resp.Operation.Status), nil
	}
}
