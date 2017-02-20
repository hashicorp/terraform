package shield

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
)

type Schedule struct {
	Name    string `json:"name,omitempty"`
	Summary string `json:"summary,omitempty"`
	When    string `json:"when,omitempty"`
	Uuid    string `json:"uuid,omitempty"`
}

func resourceSchedule() *schema.Resource {
	return &schema.Resource{
		Create: resourceScheduleCreate,
		Read:   resourceScheduleRead,
		Update: resourceScheduleUpdate,
		Delete: resourceScheduleDelete,
		Exists: resourceScheduleExists,

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

			"when": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"uuid": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func createSchedule(d *schema.ResourceData) *Schedule {

	return &Schedule{
		Name:    d.Get("name").(string),
		Summary: d.Get("summary").(string),
		When:    d.Get("when").(string),
	}

}

func resourceScheduleCreate(d *schema.ResourceData, m interface{}) error {
	client := m.(*ShieldClient)
	schedule := createSchedule(d)

	var jsonbuffer []byte

	jsonpayload := bytes.NewBuffer(jsonbuffer)
	enc := json.NewEncoder(jsonpayload)
	enc.Encode(schedule)

	schedule_req, err := client.Post(fmt.Sprintf("v1/schedules"), jsonpayload)

	decoder := json.NewDecoder(schedule_req.Body)
	err = decoder.Decode(&schedule)
	if err != nil {
		return err
	}

	d.SetId(schedule.Uuid)
	d.Set("uuid", schedule.Uuid)

	return resourceScheduleRead(d, m)
}

func resourceScheduleRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*ShieldClient)
	schedule_req, err := client.Get(fmt.Sprintf("v1/schedules"))
	if err != nil {
		return err
	}

	decoder := json.NewDecoder(schedule_req.Body)
	decoder.Token()

	for decoder.More() {
		var schedule Schedule

		err := decoder.Decode(&schedule)
		if err != nil {
			return err
		}
		if schedule.Uuid == d.Get("uuid") {
			d.Set("uuid", schedule.Uuid)
			d.Set("name", schedule.Name)
			d.Set("summary", schedule.Summary)
			d.Set("when", schedule.When)
			break
		}
	}

	return nil
}

func resourceScheduleUpdate(d *schema.ResourceData, m interface{}) error {
	client := m.(*ShieldClient)
	schedule := createSchedule(d)

	var jsonbuffer []byte

	jsonpayload := bytes.NewBuffer(jsonbuffer)
	enc := json.NewEncoder(jsonpayload)
	enc.Encode(schedule)

	schedule_req, err := client.Put(fmt.Sprintf("v1/schedule/%s",
		d.Get("uuid").(string),
	), jsonpayload)

	if err != nil {
		return err
	}

	decoder := json.NewDecoder(schedule_req.Body)
	err = decoder.Decode(&schedule)
	if err != nil {
		return err
	}

	return resourceScheduleRead(d, m)
}

func resourceScheduleExists(d *schema.ResourceData, m interface{}) (bool, error) {
	client := m.(*ShieldClient)
	if _, okay := d.GetOk("uuid"); okay {
		schedule_req, err := client.Get(fmt.Sprintf("v1/schedule/%s",
			d.Get("uuid").(string),
		))

		if err != nil {
			panic(err)
		}

		if schedule_req.StatusCode != 200 {
			d.SetId("")
			return false, nil
		}

		return true, nil
	} else {
		return false, nil
	}

}

func resourceScheduleDelete(d *schema.ResourceData, m interface{}) error {
	client := m.(*ShieldClient)
	_, err := client.Delete(fmt.Sprintf("v1/schedule/%s",
		d.Get("uuid").(string),
	))

	if err != nil {
		return err
	}
	return nil
}
