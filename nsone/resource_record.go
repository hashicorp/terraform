package nsone

import (
	"github.com/bobtfish/go-nsone-api"
	"github.com/hashicorp/terraform/helper/schema"
)

func recordResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"zone": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"domain": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"meta": metaSchema(),
			/*			"answers": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},
						"expiry": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},
						"hostmaster": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
			*/
		},
		Create: RecordCreate,
		Read:   RecordRead,
		Update: RecordUpdate,
		Delete: RecordDelete,
	}
}

func recordToResourceData(d *schema.ResourceData, z *nsone.Record) {
	d.SetId(r.Id)
}

func RecordCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	r := nsone.NewRecord(d.Get("zone").(string), d.Get("domain").(string), d.Get("type").(string))
	err := client.CreateRecord(r)
	//    zone := d.Get("zone").(string)
	//    hostmaster := d.Get("hostmaster").(string)
	if err != nil {
		return err
	}
	recordToResourceData(d, r)
	return nil
}

func RecordRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	r, err := client.GetRecord(d.Get("zone").(string), d.Get("domain").(string), d.Get("type").(string))
	//    zone := d.Get("zone").(string)
	//    hostmaster := d.Get("hostmaster").(string)
	if err != nil {
		return err
	}
	recordToResourceData(d, r)
	return nil
}

func RecordDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	err := client.DeleteRecord(d.Get("zone").(string), d.Get("domain").(string), d.Get("type").(string))
	d.SetId("")
	return err
}

func RecordUpdate(d *schema.ResourceData, meta interface{}) error {
	panic("Update not implemented")
	return nil
}
