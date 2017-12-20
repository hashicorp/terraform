package aws

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/directconnect"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsDxLag() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDxLagCreate,
		Read:   resourceAwsDxLagRead,
		Update: resourceAwsDxLagUpdate,
		Delete: resourceAwsDxLagDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"connections_bandwidth": &schema.Schema{
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
			"number_of_connections": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
			"force_destroy": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
		},
	}
}

func resourceAwsDxLagCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dxconn

	input := &directconnect.CreateLagInput{
		ConnectionsBandwidth: aws.String(d.Get("connections_bandwidth").(string)),
		LagName:              aws.String(d.Get("name").(string)),
		Location:             aws.String(d.Get("location").(string)),
		NumberOfConnections:  aws.Int64(int64(d.Get("number_of_connections").(int))),
	}
	resp, err := conn.CreateLag(input)
	if err != nil {
		return err
	}
	d.SetId(*resp.LagId)
	return resourceAwsDxLagRead(d, meta)
}

func resourceAwsDxLagRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dxconn

	lagId := d.Id()
	input := &directconnect.DescribeLagsInput{
		LagId: aws.String(lagId),
	}
	resp, err := conn.DescribeLags(input)
	if err != nil {
		return err
	}
	if len(resp.Lags) < 1 {
		d.SetId("")
		return nil
	}
	if len(resp.Lags) != 1 {
		return fmt.Errorf("[ERROR] Number of DX Lag (%s) isn't one, got %d", lagId, len(resp.Lags))
	}
	if d.Id() != *resp.Lags[0].LagId {
		return fmt.Errorf("[ERROR] DX Lag (%s) not found", lagId)
	}
	return nil
}

func resourceAwsDxLagUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dxconn

	input := &directconnect.UpdateLagInput{
		LagId: aws.String(d.Id()),
	}
	if d.HasChange("name") {
		input.LagName = aws.String(d.Get("name").(string))
	}
	_, err := conn.UpdateLag(input)
	if err != nil {
		return err
	}
	return nil
}

func resourceAwsDxLagDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dxconn

	if d.Get("force_destroy").(bool) {
		input := &directconnect.DescribeLagsInput{
			LagId: aws.String(d.Id()),
		}
		resp, err := conn.DescribeLags(input)
		if err != nil {
			return err
		}
		lag := resp.Lags[0]
		for _, v := range lag.Connections {
			dcinput := &directconnect.DeleteConnectionInput{
				ConnectionId: v.ConnectionId,
			}
			if _, err := conn.DeleteConnection(dcinput); err != nil {
				return err
			}
		}
	}

	input := &directconnect.DeleteLagInput{
		LagId: aws.String(d.Id()),
	}
	_, err := conn.DeleteLag(input)
	if err != nil {
		return err
	}
	deleteStateConf := &resource.StateChangeConf{
		Pending:    []string{directconnect.LagStateAvailable, directconnect.LagStateRequested, directconnect.LagStatePending, directconnect.LagStateDeleting},
		Target:     []string{directconnect.LagStateDeleted},
		Refresh:    dxLagRefreshStateFunc(conn, d.Id()),
		Timeout:    10 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}
	_, err = deleteStateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for Dx Lag (%s) to be deleted: %s", d.Id(), err)
	}
	d.SetId("")
	return nil
}

func dxLagRefreshStateFunc(conn *directconnect.DirectConnect, lagId string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		input := &directconnect.DescribeLagsInput{
			LagId: aws.String(lagId),
		}
		resp, err := conn.DescribeLags(input)
		if err != nil {
			return nil, "failed", err
		}
		return resp, *resp.Lags[0].LagState, nil
	}
}
