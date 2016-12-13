package akamai

import (
	"log"
	"strconv"

	"github.com/Comcast/go-edgegrid/edgegrid"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAkamaiGTMDataCenter() *schema.Resource {
	return &schema.Resource{
		Create: resourceGTMDataCenterCreate,
		Read:   resourceGTMDataCenterRead,
		Update: resourceGTMDataCenterUpdate,
		Delete: resourceGTMDataCenterDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"id": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},
			"domain": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"city": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"state_or_province": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"country": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"continent": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"latitude": &schema.Schema{
				Type:     schema.TypeFloat,
				Optional: true,
			},
			"longitude": &schema.Schema{
				Type:     schema.TypeFloat,
				Optional: true,
			},
			"virtual": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"cloud_server_targeting": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
		},
	}
}

func resourceGTMDataCenterCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Creating GTM Data Center: %+v", d)

	created, err := meta.(*Clients).GTM.DataCenterCreate(d.Get("domain").(string), dc(d))
	if err != nil {
		log.Printf("resourceDataCenterCreate: %v", err)
		return err
	}

	log.Printf("[INFO] Created GTM Data Center named: %s, with ID of: %d", created.DataCenter.Nickname, created.DataCenter.DataCenterID)

	d.SetId(strconv.Itoa(created.DataCenter.DataCenterID))

	return resourceGTMDataCenterRead(d, meta)
}

func resourceGTMDataCenterRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Reading GTM Data Center: %s", d.Id())
	dcID, err := strconv.Atoi(d.Id())
	if err != nil {
		return err
	}
	read, err := meta.(*Clients).GTM.DataCenter(d.Get("domain").(string), dcID)
	if err != nil {
		return err
	}

	d.Set("name", read.Nickname)

	return nil
}

func resourceGTMDataCenterUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Updating GTM Data Center: %s", d.Id())

	updateBody := dc(d)
	dcID, err := strconv.Atoi(d.Id())
	if err != nil {
		return err
	}
	updateBody.DataCenterID = dcID

	_, err = meta.(*Clients).GTM.DataCenterUpdate(d.Get("domain").(string), updateBody)
	if err != nil {
		return err
	}

	return resourceGTMDataCenterRead(d, meta)
}

func resourceGTMDataCenterDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Deleting Data Center: %s", d.Id())
	dcID, err := strconv.Atoi(d.Id())
	if err != nil {
		return err
	}
	err = meta.(*Clients).GTM.DataCenterDelete(d.Get("domain").(string), dcID)
	if err != nil {
		return err
	}

	d.SetId("")
	return err
}

func dc(d *schema.ResourceData) *edgegrid.DataCenter {
	return &edgegrid.DataCenter{
		Nickname:             d.Get("name").(string),
		City:                 d.Get("city").(string),
		Country:              d.Get("country").(string),
		StateOrProvince:      d.Get("state_or_province").(string),
		Continent:            d.Get("continent").(string),
		Latitude:             d.Get("latitude").(float64),
		Longitude:            d.Get("longitude").(float64),
		Virtual:              d.Get("virtual").(bool),
		CloudServerTargeting: d.Get("cloud_server_targeting").(bool),
	}
}
