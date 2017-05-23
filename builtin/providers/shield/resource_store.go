package shield

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
)

type Store struct {
	Name     string `json:"name,omitempty"`
	Summary  string `json:"summary,omitempty"`
	Plugin   string `json:"plugin,omitempty"`
	Endpoint string `json:"endpoint,omitempty"`
	Uuid     string `json:"uuid,omitempty"`
}

func resourceStore() *schema.Resource {
	return &schema.Resource{
		Create: resourceStoreCreate,
		Read:   resourceStoreRead,
		Update: resourceStoreUpdate,
		Delete: resourceStoreDelete,
		Exists: resourceStoreExists,

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

			"endpoint": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: ValidateStoreJSON,
			},

			"uuid": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func createStore(d *schema.ResourceData) *Store {

	return &Store{
		Name:     d.Get("name").(string),
		Summary:  d.Get("summary").(string),
		Plugin:   d.Get("plugin").(string),
		Endpoint: d.Get("endpoint").(string),
	}

}

func resourceStoreCreate(d *schema.ResourceData, m interface{}) error {
	client := m.(*ShieldClient)
	store := createStore(d)

	var jsonbuffer []byte

	jsonpayload := bytes.NewBuffer(jsonbuffer)
	enc := json.NewEncoder(jsonpayload)
	enc.Encode(store)

	store_req, err := client.Post(fmt.Sprintf("v1/stores"), jsonpayload)

	decoder := json.NewDecoder(store_req.Body)
	err = decoder.Decode(&store)
	if err != nil {
		return err
	}

	d.SetId(store.Uuid)
	d.Set("uuid", store.Uuid)

	return resourceStoreRead(d, m)
}

func resourceStoreRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*ShieldClient)
	store_req, err := client.Get(fmt.Sprintf("v1/stores"))
	if err != nil {
		return err
	}

	decoder := json.NewDecoder(store_req.Body)
	decoder.Token()

	for decoder.More() {
		var store Store

		err := decoder.Decode(&store)
		if err != nil {
			return err
		}
		if store.Uuid == d.Get("uuid") {
			d.Set("uuid", store.Uuid)
			d.Set("name", store.Name)
			d.Set("summary", store.Summary)
			d.Set("plugin", store.Plugin)
			d.Set("endpoint", store.Endpoint)
			break
		}
	}

	return nil
}

func resourceStoreUpdate(d *schema.ResourceData, m interface{}) error {
	client := m.(*ShieldClient)
	store := createStore(d)

	var jsonbuffer []byte

	jsonpayload := bytes.NewBuffer(jsonbuffer)
	enc := json.NewEncoder(jsonpayload)
	enc.Encode(store)

	store_req, err := client.Put(fmt.Sprintf("v1/store/%s",
		d.Get("uuid").(string),
	), jsonpayload)

	if err != nil {
		return err
	}

	decoder := json.NewDecoder(store_req.Body)
	err = decoder.Decode(&store)
	if err != nil {
		return err
	}

	return resourceStoreRead(d, m)
}

func resourceStoreExists(d *schema.ResourceData, m interface{}) (bool, error) {
	client := m.(*ShieldClient)
	if _, okay := d.GetOk("uuid"); okay {
		store_req, err := client.Get(fmt.Sprintf("v1/store/%s",
			d.Get("uuid").(string),
		))

		if err != nil {
			panic(err)
		}

		if store_req.StatusCode != 200 {
			d.SetId("")
			return false, nil
		}

		return true, nil
	} else {
		return false, nil
	}

}

func resourceStoreDelete(d *schema.ResourceData, m interface{}) error {
	client := m.(*ShieldClient)
	_, err := client.Delete(fmt.Sprintf("v1/store/%s",
		d.Get("uuid").(string),
	))

	if err != nil {
		return err
	}
	return nil
}

func ValidateStoreJSON(configI interface{}, k string) ([]string, []error) {
	configJSON := configI.(string)
	configMap := map[string]interface{}{}
	err := json.Unmarshal([]byte(configJSON), &configMap)
	if err != nil {
		return nil, []error{err}
	}
	return nil, nil
}
