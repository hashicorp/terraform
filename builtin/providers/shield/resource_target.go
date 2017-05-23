package shield

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
)

type Target struct {
	Name     string `json:"name,omitempty"`
	Summary  string `json:"summary,omitempty"`
	Plugin   string `json:"plugin,omitempty"`
	Agent    string `json:"agent,omitempty"`
	Endpoint string `json:"endpoint,omitempty"`
	Uuid     string `json:"uuid,omitempty"`
}

func resourceTarget() *schema.Resource {
	return &schema.Resource{
		Create: resourceTargetCreate,
		Read:   resourceTargetRead,
		Update: resourceTargetUpdate,
		Delete: resourceTargetDelete,
		Exists: resourceTargetExists,

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

			"plugin": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"agent": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"endpoint": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: ValidateTargetJSON,
			},

			"uuid": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func createTarget(d *schema.ResourceData) *Target {

	return &Target{
		Name:     d.Get("name").(string),
		Summary:  d.Get("summary").(string),
		Plugin:   d.Get("plugin").(string),
		Agent:    d.Get("agent").(string),
		Endpoint: d.Get("endpoint").(string),
	}

}

func resourceTargetCreate(d *schema.ResourceData, m interface{}) error {
	client := m.(*ShieldClient)
	target := createTarget(d)

	var jsonbuffer []byte

	jsonpayload := bytes.NewBuffer(jsonbuffer)
	enc := json.NewEncoder(jsonpayload)
	enc.Encode(target)

	target_req, err := client.Post(fmt.Sprintf("v1/targets"), jsonpayload)

	decoder := json.NewDecoder(target_req.Body)
	err = decoder.Decode(&target)
	if err != nil {
		return err
	}

	d.SetId(target.Uuid)
	d.Set("uuid", target.Uuid)

	return resourceTargetRead(d, m)
}

func resourceTargetRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*ShieldClient)
	target_req, err := client.Get(fmt.Sprintf("v1/targets"))
	if err != nil {
		return err
	}

	decoder := json.NewDecoder(target_req.Body)
	decoder.Token()

	for decoder.More() {
		var target Target

		err := decoder.Decode(&target)
		if err != nil {
			return err
		}
		if target.Uuid == d.Get("uuid") {
			d.Set("uuid", target.Uuid)
			d.Set("name", target.Name)
			d.Set("summary", target.Summary)
			d.Set("plugin", target.Plugin)
			d.Set("endpoint", target.Endpoint)
			d.Set("agent", target.Agent)
			break
		}
	}

	return nil
}

func resourceTargetUpdate(d *schema.ResourceData, m interface{}) error {
	client := m.(*ShieldClient)
	target := createTarget(d)

	var jsonbuffer []byte

	jsonpayload := bytes.NewBuffer(jsonbuffer)
	enc := json.NewEncoder(jsonpayload)
	enc.Encode(target)

	target_req, err := client.Put(fmt.Sprintf("v1/target/%s",
		d.Get("uuid").(string),
	), jsonpayload)

	if err != nil {
		return err
	}

	decoder := json.NewDecoder(target_req.Body)
	err = decoder.Decode(&target)
	if err != nil {
		return err
	}

	return resourceTargetRead(d, m)
}

func resourceTargetExists(d *schema.ResourceData, m interface{}) (bool, error) {
	client := m.(*ShieldClient)
	if _, okay := d.GetOk("uuid"); okay {
		target_req, err := client.Get(fmt.Sprintf("v1/target/%s",
			d.Get("uuid").(string),
		))

		if err != nil {
			panic(err)
		}

		if target_req.StatusCode != 200 {
			d.SetId("")
			return false, nil
		}

		return true, nil
	} else {
		return false, nil
	}

}

func resourceTargetDelete(d *schema.ResourceData, m interface{}) error {
	client := m.(*ShieldClient)
	_, err := client.Delete(fmt.Sprintf("v1/target/%s",
		d.Get("uuid").(string),
	))

	if err != nil {
		return err
	}
	return nil
}

func ValidateTargetJSON(configI interface{}, k string) ([]string, []error) {
	configJSON := configI.(string)
	configMap := map[string]interface{}{}
	err := json.Unmarshal([]byte(configJSON), &configMap)
	if err != nil {
		return nil, []error{err}
	}
	return nil, nil
}
