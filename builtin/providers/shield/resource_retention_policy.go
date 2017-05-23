package shield

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
)

type Retention struct {
	Name    string `json:"name,omitempty"`
	Summary string `json:"summary,omitempty"`
	Expires int    `json:"expires,omitempty"`
	Uuid    string `json:"uuid,omitempty"`
}

func resourceRetention() *schema.Resource {
	return &schema.Resource{
		Create: resourceRetentionCreate,
		Read:   resourceRetentionRead,
		Update: resourceRetentionUpdate,
		Delete: resourceRetentionDelete,
		Exists: resourceRetentionExists,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"summary": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"expires": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},

			"uuid": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func createRetention(d *schema.ResourceData) *Retention {

	return &Retention{
		Name:    d.Get("name").(string),
		Summary: d.Get("summary").(string),
		Expires: d.Get("expires").(int),
	}

}

func resourceRetentionCreate(d *schema.ResourceData, m interface{}) error {
	client := m.(*ShieldClient)
	retention := createRetention(d)

	var jsonbuffer []byte

	jsonpayload := bytes.NewBuffer(jsonbuffer)
	enc := json.NewEncoder(jsonpayload)
	enc.Encode(retention)

	retention_req, err := client.Post(fmt.Sprintf("v1/retention"), jsonpayload)

	decoder := json.NewDecoder(retention_req.Body)
	err = decoder.Decode(&retention)
	if err != nil {
		return err
	}

	d.SetId(retention.Uuid)
	d.Set("uuid", retention.Uuid)

	return resourceRetentionRead(d, m)
}

func resourceRetentionRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*ShieldClient)
	retention_req, err := client.Get(fmt.Sprintf("v1/retention"))
	if err != nil {
		return err
	}

	decoder := json.NewDecoder(retention_req.Body)
	decoder.Token()

	for decoder.More() {
		var retention Retention

		err := decoder.Decode(&retention)
		if err != nil {
			return err
		}
		if retention.Uuid == d.Get("uuid") {
			d.Set("uuid", retention.Uuid)
			d.Set("name", retention.Name)
			d.Set("summary", retention.Summary)
			d.Set("expires", retention.Expires)
			break
		}
	}

	return nil
}

func resourceRetentionUpdate(d *schema.ResourceData, m interface{}) error {
	client := m.(*ShieldClient)
	retention := createRetention(d)

	var jsonbuffer []byte

	jsonpayload := bytes.NewBuffer(jsonbuffer)
	enc := json.NewEncoder(jsonpayload)
	enc.Encode(retention)

	retention_req, err := client.Put(fmt.Sprintf("v1/retention/%s",
		d.Get("uuid").(string),
	), jsonpayload)

	if err != nil {
		return err
	}

	decoder := json.NewDecoder(retention_req.Body)
	err = decoder.Decode(&retention)
	if err != nil {
		return err
	}

	return resourceRetentionRead(d, m)
}

func resourceRetentionExists(d *schema.ResourceData, m interface{}) (bool, error) {
	client := m.(*ShieldClient)
	if _, okay := d.GetOk("uuid"); okay {
		retention_req, err := client.Get(fmt.Sprintf("v1/retention/%s",
			d.Get("uuid").(string),
		))

		if err != nil {
			panic(err)
		}

		if retention_req.StatusCode != 200 {
			d.SetId("")
			return false, nil
		}

		return true, nil
	} else {
		return false, nil
	}

}

func resourceRetentionDelete(d *schema.ResourceData, m interface{}) error {
	client := m.(*ShieldClient)
	_, err := client.Delete(fmt.Sprintf("v1/retention/%s",
		d.Get("uuid").(string),
	))

	if err != nil {
		return err
	}
	return nil
}
