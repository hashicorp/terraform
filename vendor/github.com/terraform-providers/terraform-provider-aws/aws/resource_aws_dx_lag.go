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

func resourceAwsDxLag() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDxLagCreate,
		Read:   resourceAwsDxLagRead,
		Update: resourceAwsDxLagUpdate,
		Delete: resourceAwsDxLagDelete,
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
			},
			"connections_bandwidth": {
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
			"number_of_connections": {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
				ForceNew: true,
				Deprecated: "Use aws_dx_connection and aws_dx_connection_association resources instead. " +
					"Default connections will be removed as part of LAG creation automatically in future versions.",
			},
			"force_destroy": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"tags": tagsSchema(),
		},
	}
}

func resourceAwsDxLagCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dxconn

	var noOfConnections int
	if v, ok := d.GetOk("number_of_connections"); ok {
		noOfConnections = v.(int)
	} else {
		noOfConnections = 1
	}

	req := &directconnect.CreateLagInput{
		ConnectionsBandwidth: aws.String(d.Get("connections_bandwidth").(string)),
		LagName:              aws.String(d.Get("name").(string)),
		Location:             aws.String(d.Get("location").(string)),
		NumberOfConnections:  aws.Int64(int64(noOfConnections)),
	}

	log.Printf("[DEBUG] Creating Direct Connect LAG: %#v", req)
	resp, err := conn.CreateLag(req)
	if err != nil {
		return err
	}

	// TODO: Remove default connection(s) automatically provisioned by AWS
	// per NumberOfConnections

	d.SetId(aws.StringValue(resp.LagId))
	return resourceAwsDxLagUpdate(d, meta)
}

func resourceAwsDxLagRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dxconn

	resp, err := conn.DescribeLags(&directconnect.DescribeLagsInput{
		LagId: aws.String(d.Id()),
	})
	if err != nil {
		if isNoSuchDxLagErr(err) {
			log.Printf("[WARN] Direct Connect LAG (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	if len(resp.Lags) < 1 {
		log.Printf("[WARN] Direct Connect LAG (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}
	if len(resp.Lags) != 1 {
		return fmt.Errorf("[ERROR] Number of Direct Connect LAGs (%s) isn't one, got %d", d.Id(), len(resp.Lags))
	}
	lag := resp.Lags[0]
	if d.Id() != aws.StringValue(lag.LagId) {
		return fmt.Errorf("[ERROR] Direct Connect LAG (%s) not found", d.Id())
	}

	if aws.StringValue(lag.LagState) == directconnect.LagStateDeleted {
		log.Printf("[WARN] Direct Connect LAG (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	arn := arn.ARN{
		Partition: meta.(*AWSClient).partition,
		Region:    meta.(*AWSClient).region,
		Service:   "directconnect",
		AccountID: meta.(*AWSClient).accountid,
		Resource:  fmt.Sprintf("dxlag/%s", d.Id()),
	}.String()
	d.Set("arn", arn)
	d.Set("name", lag.LagName)
	d.Set("connections_bandwidth", lag.ConnectionsBandwidth)
	d.Set("location", lag.Location)

	if err := getTagsDX(conn, d, arn); err != nil {
		return err
	}

	return nil
}

func resourceAwsDxLagUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dxconn

	d.Partial(true)

	if d.HasChange("name") {
		req := &directconnect.UpdateLagInput{
			LagId:   aws.String(d.Id()),
			LagName: aws.String(d.Get("name").(string)),
		}

		log.Printf("[DEBUG] Updating Direct Connect LAG: %#v", req)
		_, err := conn.UpdateLag(req)
		if err != nil {
			return err
		} else {
			d.SetPartial("name")
		}
	}

	arn := arn.ARN{
		Partition: meta.(*AWSClient).partition,
		Region:    meta.(*AWSClient).region,
		Service:   "directconnect",
		AccountID: meta.(*AWSClient).accountid,
		Resource:  fmt.Sprintf("dxlag/%s", d.Id()),
	}.String()
	if err := setTagsDX(conn, d, arn); err != nil {
		return err
	} else {
		d.SetPartial("tags")
	}

	d.Partial(false)

	return resourceAwsDxLagRead(d, meta)
}

func resourceAwsDxLagDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dxconn

	if d.Get("force_destroy").(bool) {
		resp, err := conn.DescribeLags(&directconnect.DescribeLagsInput{
			LagId: aws.String(d.Id()),
		})
		if err != nil {
			if isNoSuchDxLagErr(err) {
				return nil
			}
			return err
		}

		if len(resp.Lags) < 1 {
			return nil
		}
		lag := resp.Lags[0]
		for _, v := range lag.Connections {
			log.Printf("[DEBUG] Deleting Direct Connect connection: %s", aws.StringValue(v.ConnectionId))
			_, err := conn.DeleteConnection(&directconnect.DeleteConnectionInput{
				ConnectionId: v.ConnectionId,
			})
			if err != nil && !isNoSuchDxConnectionErr(err) {
				return err
			}
		}
	}

	log.Printf("[DEBUG] Deleting Direct Connect LAG: %s", d.Id())
	_, err := conn.DeleteLag(&directconnect.DeleteLagInput{
		LagId: aws.String(d.Id()),
	})
	if err != nil {
		if isNoSuchDxLagErr(err) {
			return nil
		}
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
		return fmt.Errorf("Error waiting for Direct Connect LAG (%s) to be deleted: %s", d.Id(), err)
	}

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
		if len(resp.Lags) < 1 {
			return resp, directconnect.LagStateDeleted, nil
		}
		return resp, *resp.Lags[0].LagState, nil
	}
}

func isNoSuchDxLagErr(err error) bool {
	return isAWSErr(err, "DirectConnectClientException", "Could not find Lag with ID")
}
