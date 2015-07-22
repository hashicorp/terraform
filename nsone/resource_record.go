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

func RecordCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	r := nsone.NewRecord(d.Get("zone").(string), d.Get("domain").(string), d.Get("type").(string))
	err := client.CreateRecord(r)
	//    zone := d.Get("zone").(string)
	//    hostmaster := d.Get("hostmaster").(string)
	if err != nil {
		return err
	}
	d.SetId(r.Id)
	return nil
}

func RecordRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	r := nsone.NewRecord(d.Get("zone").(string), d.Get("domain").(string), d.Get("type").(string))
	err := client.GetRecord(r)
	//    zone := d.Get("zone").(string)
	//    hostmaster := d.Get("hostmaster").(string)
	if err != nil {
		return err
	}
	d.SetId(r.Id)
	return nil
}

func RecordDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	r := nsone.NewRecord(d.Get("zone").(string), d.Get("domain").(string), d.Get("type").(string))
	err := client.DeleteRecord(r)
	d.SetId("")
	return err
}

func RecordUpdate(d *schema.ResourceData, meta interface{}) error {
	panic("Update not implemented")
	return nil
}
