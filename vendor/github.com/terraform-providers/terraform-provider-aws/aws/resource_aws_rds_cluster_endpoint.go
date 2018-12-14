package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

const (
	AWSRDSClusterEndpointRetryDelay      = 5 * time.Second
	AWSRDSClusterEndpointRetryMinTimeout = 3 * time.Second
)

func resourceAwsRDSClusterEndpoint() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsRDSClusterEndpointCreate,
		Read:   resourceAwsRDSClusterEndpointRead,
		Update: resourceAwsRDSClusterEndpointUpdate,
		Delete: resourceAwsRDSClusterEndpointDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"cluster_endpoint_identifier": {
				Type:         schema.TypeString,
				ForceNew:     true,
				Required:     true,
				ValidateFunc: validateRdsIdentifier,
			},
			"cluster_identifier": {
				Type:         schema.TypeString,
				ForceNew:     true,
				Required:     true,
				ValidateFunc: validateRdsIdentifier,
			},
			"custom_endpoint_type": {
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: validation.StringInSlice([]string{
					"READER",
					"ANY",
				}, false),
			},
			"excluded_members": {
				Type:          schema.TypeSet,
				Optional:      true,
				ConflictsWith: []string{"static_members"},
				Elem:          &schema.Schema{Type: schema.TypeString},
				Set:           schema.HashString,
			},
			"static_members": {
				Type:          schema.TypeSet,
				Optional:      true,
				ConflictsWith: []string{"excluded_members"},
				Elem:          &schema.Schema{Type: schema.TypeString},
				Set:           schema.HashString,
			},
			"endpoint": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsRDSClusterEndpointCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn

	clusterId := d.Get("cluster_identifier").(string)
	endpointId := d.Get("cluster_endpoint_identifier").(string)
	endpointType := d.Get("custom_endpoint_type").(string)

	createClusterEndpointInput := &rds.CreateDBClusterEndpointInput{
		DBClusterIdentifier:         aws.String(clusterId),
		DBClusterEndpointIdentifier: aws.String(endpointId),
		EndpointType:                aws.String(endpointType),
	}

	if v := d.Get("static_members"); v != nil {
		createClusterEndpointInput.StaticMembers = expandStringSet(v.(*schema.Set))
	}
	if v := d.Get("excluded_members"); v != nil {
		createClusterEndpointInput.ExcludedMembers = expandStringSet(v.(*schema.Set))
	}

	_, err := conn.CreateDBClusterEndpoint(createClusterEndpointInput)
	if err != nil {
		return fmt.Errorf("Error creating RDS Cluster Endpoint: %s", err)
	}

	d.SetId(endpointId)

	err = resourceAwsRDSClusterEndpointWaitForAvailable(d.Timeout(schema.TimeoutDelete), d.Id(), conn)
	if err != nil {
		return err
	}

	return resourceAwsRDSClusterEndpointRead(d, meta)
}

func resourceAwsRDSClusterEndpointRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn

	input := &rds.DescribeDBClusterEndpointsInput{
		DBClusterEndpointIdentifier: aws.String(d.Id()),
	}
	log.Printf("[DEBUG] Describing RDS Cluster: %s", input)
	resp, err := conn.DescribeDBClusterEndpoints(input)

	if err != nil {
		return fmt.Errorf("error describing RDS Cluster Endpoints (%s): %s", d.Id(), err)
	}

	if resp == nil {
		return fmt.Errorf("Error retrieving RDS Cluster Endpoints: empty response for: %s", input)
	}

	var clusterEp *rds.DBClusterEndpoint
	for _, e := range resp.DBClusterEndpoints {
		if aws.StringValue(e.DBClusterEndpointIdentifier) == d.Id() {
			clusterEp = e
			break
		}
	}

	if clusterEp == nil {
		log.Printf("[WARN] RDS Cluster Endpoint (%s) not found", d.Id())
		d.SetId("")
		return nil
	}

	d.Set("cluster_endpoint_identifier", clusterEp.DBClusterEndpointIdentifier)
	d.Set("cluster_identifier", clusterEp.DBClusterIdentifier)
	d.Set("arn", clusterEp.DBClusterEndpointArn)
	d.Set("endpoint", clusterEp.Endpoint)
	d.Set("custom_endpoint_type", clusterEp.CustomEndpointType)

	if err := d.Set("excluded_members", flattenStringList(clusterEp.ExcludedMembers)); err != nil {
		return fmt.Errorf("error setting excluded_members: %s", err)
	}

	if err := d.Set("static_members", flattenStringList(clusterEp.StaticMembers)); err != nil {
		return fmt.Errorf("error setting static_members: %s", err)
	}

	return nil
}

func resourceAwsRDSClusterEndpointUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn
	input := &rds.ModifyDBClusterEndpointInput{
		DBClusterEndpointIdentifier: aws.String(d.Id()),
	}

	if v, ok := d.GetOk("custom_endpoint_type"); ok {
		input.EndpointType = aws.String(v.(string))
	}

	if attr := d.Get("excluded_members").(*schema.Set); attr.Len() > 0 {
		input.ExcludedMembers = expandStringList(attr.List())
	} else {
		input.ExcludedMembers = make([]*string, 0)
	}

	if attr := d.Get("static_members").(*schema.Set); attr.Len() > 0 {
		input.StaticMembers = expandStringList(attr.List())
	} else {
		input.StaticMembers = make([]*string, 0)
	}

	_, err := conn.ModifyDBClusterEndpoint(input)
	if err != nil {
		return fmt.Errorf("Error modifying RDS Cluster Endpoint: %s", err)
	}

	return resourceAwsRDSClusterEndpointRead(d, meta)
}

func resourceAwsRDSClusterEndpointDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn
	input := &rds.DeleteDBClusterEndpointInput{
		DBClusterEndpointIdentifier: aws.String(d.Id()),
	}
	_, err := conn.DeleteDBClusterEndpoint(input)
	if err != nil {
		return fmt.Errorf("Error deleting RDS Cluster Endpoint: %s", err)
	}

	if err := resourceAwsRDSClusterEndpointWaitForDestroy(d.Timeout(schema.TimeoutDelete), d.Id(), conn); err != nil {
		return err
	}

	return nil
}

func resourceAwsRDSClusterEndpointWaitForDestroy(timeout time.Duration, id string, conn *rds.RDS) error {
	log.Printf("Waiting for RDS Cluster Endpoint %s to be deleted...", id)
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"available", "deleting"},
		Target:     []string{"destroyed"},
		Refresh:    DBClusterEndpointStateRefreshFunc(conn, id),
		Timeout:    timeout,
		Delay:      AWSRDSClusterEndpointRetryDelay,
		MinTimeout: AWSRDSClusterEndpointRetryMinTimeout,
	}
	_, err := stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for RDS Cluster Endpoint (%s) to be deleted: %v", id, err)
	}
	return nil
}

func resourceAwsRDSClusterEndpointWaitForAvailable(timeout time.Duration, id string, conn *rds.RDS) error {
	log.Printf("Waiting for RDS Cluster Endpoint %s to become available...", id)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"creating"},
		Target:     []string{"available"},
		Refresh:    DBClusterEndpointStateRefreshFunc(conn, id),
		Timeout:    timeout,
		Delay:      AWSRDSClusterEndpointRetryDelay,
		MinTimeout: AWSRDSClusterEndpointRetryMinTimeout,
	}

	_, err := stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for RDS Cluster Endpoint (%s) to be ready: %v", id, err)
	}
	return nil
}

func DBClusterEndpointStateRefreshFunc(conn *rds.RDS, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		emptyResp := &rds.DescribeDBClusterEndpointsOutput{}

		resp, err := conn.DescribeDBClusterEndpoints(
			&rds.DescribeDBClusterEndpointsInput{
				DBClusterEndpointIdentifier: aws.String(id),
			})
		if err != nil {
			if isAWSErr(err, rds.ErrCodeDBClusterNotFoundFault, "") {
				return emptyResp, "destroyed", nil
			} else if resp != nil && len(resp.DBClusterEndpoints) == 0 {
				return emptyResp, "destroyed", nil
			} else {
				return emptyResp, "", fmt.Errorf("Error on refresh: %+v", err)
			}
		}

		if resp == nil || resp.DBClusterEndpoints == nil || len(resp.DBClusterEndpoints) == 0 {
			return emptyResp, "destroyed", nil
		}

		return resp.DBClusterEndpoints[0], *resp.DBClusterEndpoints[0].Status, nil
	}
}
