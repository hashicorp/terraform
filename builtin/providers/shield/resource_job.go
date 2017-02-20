package shield

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
)

type Job struct {
	Name      string `json:"name,omitempty"`
	Summary   string `json:"summary,omitempty"`
	Store     string `json:"store,omitempty"`
	Target    string `json:"target,omitempty"`
	Retention string `json:"retention,omitempty"`
	Schedule  string `json:"schedule,omitempty"`
	Paused    bool   `json:"paused,omitempty"`
	Uuid      string `json:"uuid,omitempty"`
}

func resourceJob() *schema.Resource {
	return &schema.Resource{
		Create: resourceJobCreate,
		Read:   resourceJobRead,
		Update: resourceJobUpdate,
		Delete: resourceJobDelete,
		Exists: resourceJobExists,

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

			"store": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"target": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"retention": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"schedule": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"paused": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},

			"uuid": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func createJob(d *schema.ResourceData) *Job {

	return &Job{
		Name:      d.Get("name").(string),
		Summary:   d.Get("summary").(string),
		Store:     d.Get("store").(string),
		Target:    d.Get("target").(string),
		Retention: d.Get("retention").(string),
		Schedule:  d.Get("schedule").(string),
		Paused:    d.Get("paused").(bool),
		Uuid:      d.Get("uuid").(string),
	}

}

func resourceJobCreate(d *schema.ResourceData, m interface{}) error {
	client := m.(*ShieldClient)
	job := createJob(d)

	var jsonbuffer []byte

	jsonpayload := bytes.NewBuffer(jsonbuffer)
	enc := json.NewEncoder(jsonpayload)
	enc.Encode(job)

	job_req, err := client.Post(fmt.Sprintf("v1/jobs"), jsonpayload)

	decoder := json.NewDecoder(job_req.Body)
	err = decoder.Decode(&job)
	if err != nil {
		return err
	}

	d.SetId(job.Uuid)
	d.Set("uuid", job.Uuid)

	return resourceJobRead(d, m)
}

func resourceJobRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*ShieldClient)
	job_req, err := client.Get(fmt.Sprintf("v1/jobs"))
	if err != nil {
		return err
	}

	decoder := json.NewDecoder(job_req.Body)
	decoder.Token()

	for decoder.More() {
		var job Job

		err := decoder.Decode(&job)
		if err != nil {
			return err
		}
		if job.Uuid == d.Get("uuid") {
			d.Set("uuid", job.Uuid)
			d.Set("name", job.Name)
			d.Set("summary", job.Summary)
			d.Set("store", job.Store)
			d.Set("target", job.Target)
			d.Set("retention", job.Retention)
			d.Set("schedule", job.Schedule)
			d.Set("paused", job.Paused)
			d.Set("uuid", job.Uuid)
			break
		}
	}

	return nil
}

func resourceJobUpdate(d *schema.ResourceData, m interface{}) error {
	client := m.(*ShieldClient)
	job := createJob(d)

	var jsonbuffer []byte

	jsonpayload := bytes.NewBuffer(jsonbuffer)
	enc := json.NewEncoder(jsonpayload)
	enc.Encode(job)

	job_req, err := client.Put(fmt.Sprintf("v1/job/%s",
		d.Get("uuid").(string),
	), jsonpayload)

	if err != nil {
		return err
	}

	decoder := json.NewDecoder(job_req.Body)
	err = decoder.Decode(&job)
	if err != nil {
		return err
	}

	return resourceJobRead(d, m)
}

func resourceJobExists(d *schema.ResourceData, m interface{}) (bool, error) {
	client := m.(*ShieldClient)
	if _, okay := d.GetOk("uuid"); okay {
		job_req, err := client.Get(fmt.Sprintf("v1/job/%s",
			d.Get("uuid").(string),
		))

		if err != nil {
			panic(err)
		}

		if job_req.StatusCode != 200 {
			d.SetId("")
			return false, nil
		}

		return true, nil
	} else {
		return false, nil
	}

}

func resourceJobDelete(d *schema.ResourceData, m interface{}) error {
	client := m.(*ShieldClient)
	_, err := client.Delete(fmt.Sprintf("v1/job/%s",
		d.Get("uuid").(string),
	))

	if err != nil {
		return err
	}
	return nil
}
