package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/directconnect"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsDxConnection() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDxConnectionCreate,
		Read:   resourceAwsDxConnectionRead,
		Update: resourceAwsDxConnectionUpdate,
		Delete: resourceAwsDxConnectionDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"bandwidth": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateDxConnectionBandWidth,
			},
			"location": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"tags": tagsSchema(),
		},
	}
}

func resourceAwsDxConnectionCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dxconn

	req := &directconnect.CreateConnectionInput{
		Bandwidth:      aws.String(d.Get("bandwidth").(string)),
		ConnectionName: aws.String(d.Get("name").(string)),
		Location:       aws.String(d.Get("location").(string)),
	}

	log.Printf("[DEBUG] Creating Direct Connect connection: %#v", req)
	resp, err := conn.CreateConnection(req)
	if err != nil {
		return err
	}

	d.SetId(aws.StringValue(resp.ConnectionId))
	return resourceAwsDxConnectionUpdate(d, meta)
}

func resourceAwsDxConnectionRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dxconn

	resp, err := conn.DescribeConnections(&directconnect.DescribeConnectionsInput{
		ConnectionId: aws.String(d.Id()),
	})
	if err != nil {
		if isNoSuchDxConnectionErr(err) {
			log.Printf("[WARN] Direct Connect connection (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	if len(resp.Connections) < 1 {
		log.Printf("[WARN] Direct Connect connection (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}
	if len(resp.Connections) != 1 {
		return fmt.Errorf("[ERROR] Number of Direct Connect connections (%s) isn't one, got %d", d.Id(), len(resp.Connections))
	}
	connection := resp.Connections[0]
	if d.Id() != aws.StringValue(connection.ConnectionId) {
		return fmt.Errorf("[ERROR] Direct Connect connection (%s) not found", d.Id())
	}
	if aws.StringValue(connection.ConnectionState) == directconnect.ConnectionStateDeleted {
		log.Printf("[WARN] Direct Connect connection (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	arn := arn.ARN{
		Partition: meta.(*AWSClient).partition,
		Region:    meta.(*AWSClient).region,
		Service:   "directconnect",
		AccountID: meta.(*AWSClient).accountid,
		Resource:  fmt.Sprintf("dxcon/%s", d.Id()),
	}.String()
	d.Set("arn", arn)
	d.Set("name", connection.ConnectionName)
	d.Set("bandwidth", connection.Bandwidth)
	d.Set("location", connection.Location)

	if err := getTagsDX(conn, d, arn); err != nil {
		return err
	}

	return nil
}

func resourceAwsDxConnectionUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dxconn

	arn := arn.ARN{
		Partition: meta.(*AWSClient).partition,
		Region:    meta.(*AWSClient).region,
		Service:   "directconnect",
		AccountID: meta.(*AWSClient).accountid,
		Resource:  fmt.Sprintf("dxcon/%s", d.Id()),
	}.String()
	if err := setTagsDX(conn, d, arn); err != nil {
		return err
	}

	return resourceAwsDxConnectionRead(d, meta)
}

func resourceAwsDxConnectionDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dxconn

	log.Printf("[DEBUG] Deleting Direct Connect connection: %s", d.Id())
	_, err := conn.DeleteConnection(&directconnect.DeleteConnectionInput{
		ConnectionId: aws.String(d.Id()),
	})
	if err != nil {
		if isNoSuchDxConnectionErr(err) {
			return nil
		}
		return err
	}

	deleteStateConf := &resource.StateChangeConf{
		Pending:    []string{directconnect.ConnectionStatePending, directconnect.ConnectionStateOrdering, directconnect.ConnectionStateAvailable, directconnect.ConnectionStateRequested, directconnect.ConnectionStateDeleting},
		Target:     []string{directconnect.ConnectionStateDeleted},
		Refresh:    dxConnectionRefreshStateFunc(conn, d.Id()),
		Timeout:    10 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}
	_, err = deleteStateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for Direct Connect connection (%s) to be deleted: %s", d.Id(), err)
	}

	return nil
}

func dxConnectionRefreshStateFunc(conn *directconnect.DirectConnect, connId string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		input := &directconnect.DescribeConnectionsInput{
			ConnectionId: aws.String(connId),
		}
		resp, err := conn.DescribeConnections(input)
		if err != nil {
			return nil, "failed", err
		}
		if len(resp.Connections) < 1 {
			return resp, directconnect.ConnectionStateDeleted, nil
		}
		return resp, *resp.Connections[0].ConnectionState, nil
	}
}

func isNoSuchDxConnectionErr(err error) bool {
	return isAWSErr(err, "DirectConnectClientException", "Could not find Connection with ID")
}
