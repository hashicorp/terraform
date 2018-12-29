package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsApiGatewayVpcLink() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsApiGatewayVpcLinkCreate,
		Read:   resourceAwsApiGatewayVpcLinkRead,
		Update: resourceAwsApiGatewayVpcLinkUpdate,
		Delete: resourceAwsApiGatewayVpcLinkDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"target_arns": {
				Type:     schema.TypeSet,
				MaxItems: 1,
				Required: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func resourceAwsApiGatewayVpcLinkCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	input := &apigateway.CreateVpcLinkInput{
		Name:       aws.String(d.Get("name").(string)),
		TargetArns: expandStringList(d.Get("target_arns").(*schema.Set).List()),
	}
	if v, ok := d.GetOk("description"); ok {
		input.Description = aws.String(v.(string))
	}

	resp, err := conn.CreateVpcLink(input)
	if err != nil {
		return err
	}

	d.SetId(*resp.Id)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{apigateway.VpcLinkStatusPending},
		Target:     []string{apigateway.VpcLinkStatusAvailable},
		Refresh:    apigatewayVpcLinkRefreshStatusFunc(conn, *resp.Id),
		Timeout:    8 * time.Minute,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		d.SetId("")
		return fmt.Errorf("Error waiting for APIGateway Vpc Link status to be \"%s\": %s", apigateway.VpcLinkStatusAvailable, err)
	}

	return nil
}

func resourceAwsApiGatewayVpcLinkRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	input := &apigateway.GetVpcLinkInput{
		VpcLinkId: aws.String(d.Id()),
	}

	resp, err := conn.GetVpcLink(input)
	if err != nil {
		if isAWSErr(err, apigateway.ErrCodeNotFoundException, "") {
			log.Printf("[WARN] VPC Link %s not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("name", resp.Name)
	d.Set("description", resp.Description)
	d.Set("target_arns", flattenStringList(resp.TargetArns))
	return nil
}

func resourceAwsApiGatewayVpcLinkUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	operations := make([]*apigateway.PatchOperation, 0)

	if d.HasChange("name") {
		operations = append(operations, &apigateway.PatchOperation{
			Op:    aws.String("replace"),
			Path:  aws.String("/name"),
			Value: aws.String(d.Get("name").(string)),
		})
	}

	if d.HasChange("description") {
		operations = append(operations, &apigateway.PatchOperation{
			Op:    aws.String("replace"),
			Path:  aws.String("/description"),
			Value: aws.String(d.Get("description").(string)),
		})
	}

	input := &apigateway.UpdateVpcLinkInput{
		VpcLinkId:       aws.String(d.Id()),
		PatchOperations: operations,
	}

	_, err := conn.UpdateVpcLink(input)
	if err != nil {
		if isAWSErr(err, apigateway.ErrCodeNotFoundException, "") {
			log.Printf("[WARN] VPC Link %s not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{apigateway.VpcLinkStatusPending},
		Target:     []string{apigateway.VpcLinkStatusAvailable},
		Refresh:    apigatewayVpcLinkRefreshStatusFunc(conn, d.Id()),
		Timeout:    8 * time.Minute,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for APIGateway Vpc Link status to be \"%s\": %s", apigateway.VpcLinkStatusAvailable, err)
	}

	return nil
}

func resourceAwsApiGatewayVpcLinkDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	input := &apigateway.DeleteVpcLinkInput{
		VpcLinkId: aws.String(d.Id()),
	}

	_, err := conn.DeleteVpcLink(input)
	if err != nil {
		if isAWSErr(err, apigateway.ErrCodeNotFoundException, "") {
			return nil
		}
		return err
	}

	stateConf := resource.StateChangeConf{
		Pending: []string{apigateway.VpcLinkStatusPending,
			apigateway.VpcLinkStatusAvailable,
			apigateway.VpcLinkStatusDeleting},
		Target:     []string{""},
		Timeout:    5 * time.Minute,
		MinTimeout: 1 * time.Second,
		Refresh: func() (interface{}, string, error) {
			resp, err := conn.GetVpcLink(&apigateway.GetVpcLinkInput{
				VpcLinkId: aws.String(d.Id()),
			})
			if err != nil {
				if isAWSErr(err, apigateway.ErrCodeNotFoundException, "") {
					return 1, "", nil
				}
				return nil, "failed", err
			}
			return resp, *resp.Status, nil
		},
	}

	if _, err := stateConf.WaitForState(); err != nil {
		return err
	}

	return nil
}

func apigatewayVpcLinkRefreshStatusFunc(conn *apigateway.APIGateway, vl string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		input := &apigateway.GetVpcLinkInput{
			VpcLinkId: aws.String(vl),
		}
		resp, err := conn.GetVpcLink(input)
		if err != nil {
			return nil, "failed", err
		}
		return resp, *resp.Status, nil
	}
}
