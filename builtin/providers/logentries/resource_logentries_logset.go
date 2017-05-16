package logentries

import (
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	logentries "github.com/logentries/le_goclient"
)

func resourceLogentriesLogSet() *schema.Resource {

	return &schema.Resource{
		Create: resourceLogentriesLogSetCreate,
		Read:   resourceLogentriesLogSetRead,
		Update: resourceLogentriesLogSetUpdate,
		Delete: resourceLogentriesLogSetDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"location": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "nonlocation",
			},
		},
	}
}

func resourceLogentriesLogSetCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*logentries.Client)
	res, err := client.LogSet.Create(logentries.LogSetCreateRequest{
		Name:     d.Get("name").(string),
		Location: d.Get("location").(string),
	})

	if err != nil {
		return err
	}

	d.SetId(res.Key)

	return resourceLogentriesLogSetRead(d, meta)
}

func resourceLogentriesLogSetRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*logentries.Client)
	res, err := client.LogSet.Read(logentries.LogSetReadRequest{
		Key: d.Id(),
	})
	if err != nil {
		if strings.Contains(err.Error(), "No such log set") {
			log.Printf("Logentries LogSet Not Found - Refreshing from State")
			d.SetId("")
			return nil
		}
		return err
	}

	if res == nil {
		d.SetId("")
		return nil
	}

	d.Set("location", res.Location)
	return nil
}

func resourceLogentriesLogSetUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*logentries.Client)
	_, err := client.LogSet.Update(logentries.LogSetUpdateRequest{
		Key:      d.Id(),
		Name:     d.Get("name").(string),
		Location: d.Get("location").(string),
	})
	if err != nil {
		return err
	}

	return resourceLogentriesLogRead(d, meta)
}

func resourceLogentriesLogSetDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*logentries.Client)
	err := client.LogSet.Delete(logentries.LogSetDeleteRequest{
		Key: d.Id(),
	})
	return err
}
