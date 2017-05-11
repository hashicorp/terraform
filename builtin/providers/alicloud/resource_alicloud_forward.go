package alicloud

import (
	"fmt"
	"github.com/denverdino/aliyungo/ecs"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAliyunForwardEntry() *schema.Resource {
	return &schema.Resource{
		Create: resourceAliyunForwardEntryCreate,
		Read:   resourceAliyunForwardEntryRead,
		Update: resourceAliyunForwardEntryUpdate,
		Delete: resourceAliyunForwardEntryDelete,

		Schema: map[string]*schema.Schema{
			"forward_table_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"external_ip": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"external_port": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateForwardPort,
			},
			"ip_protocol": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateAllowedStringValue([]string{"tcp", "udp", "any"}),
			},
			"internal_ip": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"internal_port": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateForwardPort,
			},
		},
	}
}

func resourceAliyunForwardEntryCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AliyunClient).vpcconn

	args := &ecs.CreateForwardEntryArgs{
		RegionId:       getRegion(d, meta),
		ForwardTableId: d.Get("forward_table_id").(string),
		ExternalIp:     d.Get("external_ip").(string),
		ExternalPort:   d.Get("external_port").(string),
		IpProtocol:     d.Get("ip_protocol").(string),
		InternalIp:     d.Get("internal_ip").(string),
		InternalPort:   d.Get("internal_port").(string),
	}

	resp, err := conn.CreateForwardEntry(args)
	if err != nil {
		return fmt.Errorf("CreateForwardEntry got error: %#v", err)
	}

	d.SetId(resp.ForwardEntryId)
	d.Set("forward_table_id", d.Get("forward_table_id").(string))

	return resourceAliyunForwardEntryRead(d, meta)
}

func resourceAliyunForwardEntryRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AliyunClient)

	forwardEntry, err := client.DescribeForwardEntry(d.Get("forward_table_id").(string), d.Id())

	if err != nil {
		if notFoundError(err) {
			return nil
		}
		return err
	}

	d.Set("forward_table_id", forwardEntry.ForwardTableId)
	d.Set("external_ip", forwardEntry.ExternalIp)
	d.Set("external_port", forwardEntry.ExternalPort)
	d.Set("ip_protocol", forwardEntry.IpProtocol)
	d.Set("internal_ip", forwardEntry.InternalIp)
	d.Set("internal_port", forwardEntry.InternalPort)

	return nil
}

func resourceAliyunForwardEntryUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AliyunClient)
	conn := client.vpcconn

	forwardEntry, err := client.DescribeForwardEntry(d.Get("forward_table_id").(string), d.Id())
	if err != nil {
		return err
	}

	d.Partial(true)
	attributeUpdate := false
	args := &ecs.ModifyForwardEntryArgs{
		RegionId:       getRegion(d, meta),
		ForwardTableId: forwardEntry.ForwardTableId,
		ForwardEntryId: forwardEntry.ForwardEntryId,
		ExternalIp:     forwardEntry.ExternalIp,
		IpProtocol:     forwardEntry.IpProtocol,
		ExternalPort:   forwardEntry.ExternalPort,
		InternalIp:     forwardEntry.InternalIp,
		InternalPort:   forwardEntry.InternalPort,
	}

	if d.HasChange("external_port") {
		d.SetPartial("external_port")
		args.ExternalPort = d.Get("external_port").(string)
		attributeUpdate = true
	}

	if d.HasChange("ip_protocol") {
		d.SetPartial("ip_protocol")
		args.IpProtocol = d.Get("ip_protocol").(string)
		attributeUpdate = true
	}

	if d.HasChange("internal_port") {
		d.SetPartial("internal_port")
		args.InternalPort = d.Get("internal_port").(string)
		attributeUpdate = true
	}

	if attributeUpdate {
		if err := conn.ModifyForwardEntry(args); err != nil {
			return err
		}
	}

	d.Partial(false)

	return resourceAliyunForwardEntryRead(d, meta)
}

func resourceAliyunForwardEntryDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AliyunClient)
	conn := client.vpcconn

	forwardEntryId := d.Id()
	forwardTableId := d.Get("forward_table_id").(string)

	args := &ecs.DeleteForwardEntryArgs{
		RegionId:       getRegion(d, meta),
		ForwardTableId: forwardTableId,
		ForwardEntryId: forwardEntryId,
	}

	if err := conn.DeleteForwardEntry(args); err != nil {
		return err
	}

	return nil
}
