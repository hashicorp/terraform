package alicloud

import (
	"fmt"
	"github.com/denverdino/aliyungo/ecs"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAliyunSnatEntry() *schema.Resource {
	return &schema.Resource{
		Create: resourceAliyunSnatEntryCreate,
		Read:   resourceAliyunSnatEntryRead,
		Update: resourceAliyunSnatEntryUpdate,
		Delete: resourceAliyunSnatEntryDelete,

		Schema: map[string]*schema.Schema{
			"snat_table_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"source_vswitch_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"snat_ip": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceAliyunSnatEntryCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AliyunClient).vpcconn

	args := &ecs.CreateSnatEntryArgs{
		RegionId:        getRegion(d, meta),
		SnatTableId:     d.Get("snat_table_id").(string),
		SourceVSwitchId: d.Get("source_vswitch_id").(string),
		SnatIp:          d.Get("snat_ip").(string),
	}

	resp, err := conn.CreateSnatEntry(args)
	if err != nil {
		return fmt.Errorf("CreateSnatEntry got error: %#v", err)
	}

	d.SetId(resp.SnatEntryId)
	d.Set("snat_table_id", d.Get("snat_table_id").(string))

	return resourceAliyunSnatEntryRead(d, meta)
}

func resourceAliyunSnatEntryRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AliyunClient)

	snatEntry, err := client.DescribeSnatEntry(d.Get("snat_table_id").(string), d.Id())

	if err != nil {
		if notFoundError(err) {
			return nil
		}
		return err
	}

	d.Set("snat_table_id", snatEntry.SnatTableId)
	d.Set("source_vswitch_id", snatEntry.SourceVSwitchId)
	d.Set("snat_ip", snatEntry.SnatIp)

	return nil
}

func resourceAliyunSnatEntryUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AliyunClient)
	conn := client.vpcconn

	snatEntry, err := client.DescribeSnatEntry(d.Get("snat_table_id").(string), d.Id())
	if err != nil {
		return err
	}

	d.Partial(true)
	attributeUpdate := false
	args := &ecs.ModifySnatEntryArgs{
		RegionId:    getRegion(d, meta),
		SnatTableId: snatEntry.SnatTableId,
		SnatEntryId: snatEntry.SnatEntryId,
	}

	if d.HasChange("snat_ip") {
		d.SetPartial("snat_ip")
		var snat_ip string
		if v, ok := d.GetOk("snat_ip"); ok {
			snat_ip = v.(string)
		} else {
			return fmt.Errorf("cann't change snap_ip to empty string")
		}
		args.SnatIp = snat_ip

		attributeUpdate = true
	}

	if attributeUpdate {
		if err := conn.ModifySnatEntry(args); err != nil {
			return err
		}
	}

	d.Partial(false)

	return resourceAliyunSnatEntryRead(d, meta)
}

func resourceAliyunSnatEntryDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AliyunClient)
	conn := client.vpcconn

	snatEntryId := d.Id()
	snatTableId := d.Get("snat_table_id").(string)

	args := &ecs.DeleteSnatEntryArgs{
		RegionId:    getRegion(d, meta),
		SnatTableId: snatTableId,
		SnatEntryId: snatEntryId,
	}

	if err := conn.DeleteSnatEntry(args); err != nil {
		return err
	}

	return nil
}
