package aws

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/directconnect"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsDxConnection() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDxConnectionCreate,
		Read:   resourceAwsDxConnectionRead,
		Delete: resourceAwsDxConnectionDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"bandwidth": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateDxConnectionBandWidth,
			},
			"location": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsDxConnectionCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dxconn

	input := &directconnect.CreateConnectionInput{
		Bandwidth:      aws.String(d.Get("bandwidth").(string)),
		ConnectionName: aws.String(d.Get("name").(string)),
		Location:       aws.String(d.Get("location").(string)),
	}
	resp, err := conn.CreateConnection(input)
	if err != nil {
		return err
	}
	d.SetId(*resp.ConnectionId)
	return resourceAwsDxConnectionRead(d, meta)
}

func resourceAwsDxConnectionRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dxconn

	connectionId := d.Id()
	input := &directconnect.DescribeConnectionsInput{
		ConnectionId: aws.String(connectionId),
	}
	resp, err := conn.DescribeConnections(input)
	if err != nil {
		return err
	}
	if len(resp.Connections) < 1 {
		d.SetId("")
		return nil
	}
	if len(resp.Connections) != 1 {
		return fmt.Errorf("[ERROR] Number of DX Connection (%s) isn't one, got %d", connectionId, len(resp.Connections))
	}
	if d.Id() != *resp.Connections[0].ConnectionId {
		return fmt.Errorf("[ERROR] DX Connection (%s) not found", connectionId)
	}
	return nil
}

func resourceAwsDxConnectionDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dxconn

	input := &directconnect.DeleteConnectionInput{
		ConnectionId: aws.String(d.Id()),
	}
	_, err := conn.DeleteConnection(input)
	if err != nil {
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
		return fmt.Errorf("Error waiting for Dx Connection (%s) to be deleted: %s", d.Id(), err)
	}
	d.SetId("")
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
		return resp, *resp.Connections[0].ConnectionState, nil
	}
}
