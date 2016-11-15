package ns1

import (
	"log"

	"github.com/hashicorp/terraform/helper/schema"

	ns1 "gopkg.in/ns1/ns1-go.v2/rest"
	"gopkg.in/ns1/ns1-go.v2/rest/model/dns"
)

func recordResource() *schema.Resource {
	return &schema.Resource{
		Create: resourceNS1RecordCreate,
		Read:   resourceNS1RecordRead,
		Update: resourceNS1RecordUpdate,
		Delete: resourceNS1RecordDelete,
		Exists: resourceNS1RecordExists,
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
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateRRType,
			},
			"ttl": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},
			"link": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"answer"},
			},
			"use_client_subnet": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true, // only used on POST
				Default:  true,
			},
			"answer": &schema.Schema{
				Type:          schema.TypeList,
				Optional:      true, // only optional if record contains 'link'
				ConflictsWith: []string{"link"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"rdata": &schema.Schema{
							Type:     schema.TypeList,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"region": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						// "meta": metadataSchema(),
					},
				},
			},
			"filter": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"disabled": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
						},
						"config": &schema.Schema{
							Type:     schema.TypeMap,
							Optional: true,
						},
					},
				},
			},
			// "region": &schema.Schema{
			// 	Type:     schema.TypeMap,
			// 	Optional: true,
			// },
			// "meta": metadataSchema(),
		},
	}
}

func resourceNS1RecordCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)

	r, err := buildNS1RecordStruct(d)
	if err != nil {
		return err
	}

	log.Printf("[INFO] Creating NS1 Record: %s \n", r)

	if _, err = client.Records.Create(r); err != nil {
		return err
	}

	return resourceNS1RecordRead(d, meta)
}

func resourceNS1RecordRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)

	log.Printf("[INFO] Reading NS1 Record: %s %s %s \n",
		d.Get("zone"), d.Get("domain"), d.Get("type"))

	r, _, err := client.Records.Get(
		d.Get("zone").(string),
		d.Get("domain").(string),
		d.Get("type").(string),
	)
	if err != nil {
		return err
	}

	d.SetId(r.ID)

	d.Set("zone", r.Zone)
	d.Set("domain", r.Domain)
	d.Set("type", r.Type)

	d.Set("ttl", r.TTL)
	d.Set("use_client_subnet", *r.UseClientSubnet)

	if r.Link != "" {
		d.Set("link", r.Link)
	}

	d.Set("answer", flattenNS1Answers(r.Answers))
	d.Set("filter", flattenNS1Filters(r.Filters))

	return nil
}

func resourceNS1RecordUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)

	r, err := buildNS1RecordStruct(d)
	if err != nil {
		return err
	}

	log.Printf("[INFO] Updating NS1 Record: %s \n", r)

	if _, err = client.Records.Update(r); err != nil {
		return err
	}

	return nil
}

func resourceNS1RecordDelete(d *schema.ResourceData, meta interface{}) (err error) {
	client := meta.(*ns1.Client)

	_, err = client.Records.Delete(
		d.Get("zone").(string),
		d.Get("domain").(string),
		d.Get("type").(string),
	)
	if err != nil {
		return err
	}

	d.SetId("")

	return nil
}

func resourceNS1RecordExists(d *schema.ResourceData, meta interface{}) (b bool, err error) {
	client := meta.(*ns1.Client)

	_, _, err = client.Records.Get(
		d.Get("zone").(string),
		d.Get("domain").(string),
		d.Get("type").(string),
	)
	if err == ns1.ErrRecordMissing {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

func buildNS1RecordStruct(d *schema.ResourceData) (*dns.Record, error) {
	var err error

	r := dns.NewRecord(
		d.Get("zone").(string),
		d.Get("domain").(string),
		d.Get("type").(string),
	)

	r.ID = d.Id()

	if v, ok := d.GetOk("ttl"); ok {
		r.TTL = v.(int)
	}

	if v, ok := d.GetOk("link"); ok {
		r.LinkTo(v.(string))
	}

	useSubnet := d.Get("use_client_subnet").(bool)
	r.UseClientSubnet = &useSubnet

	// r.Meta = expandNS1Metadata(d, "meta")
	// r.Regions = expandNS1Region(d)

	r.Answers, err = expandNS1Answers(d)
	if err != nil {
		return nil, err
	}

	r.Filters, err = expandNS1Filters(d)
	if err != nil {
		return nil, err
	}

	return r, nil

}
