package chef

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"

	chefc "github.com/go-chef/chef"
)

func resourceChefDataBagItem() *schema.Resource {
	return &schema.Resource{
		Create: CreateDataBagItem,
		Read:   ReadDataBagItem,
		Delete: DeleteDataBagItem,

		Schema: map[string]*schema.Schema{
			"data_bag_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"content_json": &schema.Schema{
				Type:      schema.TypeString,
				Required:  true,
				ForceNew:  true,
				StateFunc: jsonStateFunc,
			},
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func CreateDataBagItem(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*chefc.Client)

	dataBagName := d.Get("data_bag_name").(string)
	itemId, itemContent, err := prepareDataBagItemContent(d.Get("content_json").(string))
	if err != nil {
		return err
	}

	err = client.DataBags.CreateItem(dataBagName, itemContent)
	if err != nil {
		return err
	}

	d.SetId(itemId)
	d.Set("id", itemId)
	return nil
}

func ReadDataBagItem(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*chefc.Client)

	// The Chef API provides no API to read a data bag's metadata,
	// but we can try to read its items and use that as a proxy for
	// whether it still exists.

	itemId := d.Id()
	dataBagName := d.Get("data_bag_name").(string)

	value, err := client.DataBags.GetItem(dataBagName, itemId)
	if err != nil {
		if errRes, ok := err.(*chefc.ErrorResponse); ok {
			if errRes.Response.StatusCode == 404 {
				d.SetId("")
				return nil
			}
		} else {
			return err
		}
	}

	jsonContent, err := json.Marshal(value)
	if err != nil {
		return err
	}

	d.Set("content_json", string(jsonContent))

	return nil
}

func DeleteDataBagItem(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*chefc.Client)

	itemId := d.Id()
	dataBagName := d.Get("data_bag_name").(string)

	err := client.DataBags.DeleteItem(dataBagName, itemId)
	if err == nil {
		d.SetId("")
		d.Set("id", "")
	}
	return err
}

func prepareDataBagItemContent(contentJson string) (string, interface{}, error) {
	var value map[string]interface{}
	err := json.Unmarshal([]byte(contentJson), &value)
	if err != nil {
		return "", nil, err
	}

	var itemId string
	if itemIdI, ok := value["id"]; ok {
		itemId, _ = itemIdI.(string)
	}

	if itemId == "" {
		return "", nil, fmt.Errorf("content_json must have id attribute, set to a string")
	}

	return itemId, value, nil
}
