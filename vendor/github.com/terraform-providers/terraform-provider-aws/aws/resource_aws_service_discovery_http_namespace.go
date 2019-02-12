package aws

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsServiceDiscoveryHttpNamespace() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsServiceDiscoveryHttpNamespaceCreate,
		Read:   resourceAwsServiceDiscoveryHttpNamespaceRead,
		Delete: resourceAwsServiceDiscoveryHttpNamespaceDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateServiceDiscoveryHttpNamespaceName,
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
		},
	}
}

func resourceAwsServiceDiscoveryHttpNamespaceCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sdconn

	name := d.Get("name").(string)

	input := &servicediscovery.CreateHttpNamespaceInput{
		Name:             aws.String(name),
		CreatorRequestId: aws.String(resource.UniqueId()),
	}

	if v, ok := d.GetOk("description"); ok {
		input.Description = aws.String(v.(string))
	}

	resp, err := conn.CreateHttpNamespace(input)
	if err != nil {
		return fmt.Errorf("error creating Service Discovery HTTP Namespace (%s): %s", name, err)
	}

	stateConf := &resource.StateChangeConf{
		Pending: []string{servicediscovery.OperationStatusSubmitted, servicediscovery.OperationStatusPending},
		Target:  []string{servicediscovery.OperationStatusSuccess},
		Refresh: servicediscoveryOperationRefreshStatusFunc(conn, *resp.OperationId),
		Timeout: 5 * time.Minute,
	}

	opresp, err := stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("error waiting for Service Discovery HTTP Namespace (%s) creation: %s", name, err)
	}

	d.SetId(*opresp.(*servicediscovery.GetOperationOutput).Operation.Targets["NAMESPACE"])
	return resourceAwsServiceDiscoveryHttpNamespaceRead(d, meta)
}

func resourceAwsServiceDiscoveryHttpNamespaceRead(d *schema.ResourceData, meta interface{}) error {
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
		return fmt.Errorf("error reading Service Discovery HTTP Namespace (%s): %s", d.Id(), err)
	}

	d.Set("name", resp.Namespace.Name)
	d.Set("description", resp.Namespace.Description)
	d.Set("arn", resp.Namespace.Arn)

	return nil
}

func resourceAwsServiceDiscoveryHttpNamespaceDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sdconn

	input := &servicediscovery.DeleteNamespaceInput{
		Id: aws.String(d.Id()),
	}

	resp, err := conn.DeleteNamespace(input)
	if err != nil {
		if isAWSErr(err, servicediscovery.ErrCodeNamespaceNotFound, "") {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("error deleting Service Discovery HTTP Namespace (%s): %s", d.Id(), err)
	}

	stateConf := &resource.StateChangeConf{
		Pending: []string{servicediscovery.OperationStatusSubmitted, servicediscovery.OperationStatusPending},
		Target:  []string{servicediscovery.OperationStatusSuccess},
		Refresh: servicediscoveryOperationRefreshStatusFunc(conn, *resp.OperationId),
		Timeout: 5 * time.Minute,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("error waiting for Service Discovery HTTP Namespace (%s) deletion: %s", d.Id(), err)
	}

	return nil
}
