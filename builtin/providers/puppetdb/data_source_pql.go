package puppetdb

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
)

var PQLCertnames []string

func dataSourcePQL() *schema.Resource {
	return &schema.Resource{
		Read: dataSourcePQLRead,

		Schema: map[string]*schema.Schema{
			"query": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"data_json": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"data": &schema.Schema{
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeMap},
				Computed: true,
			},
		},
	}
}

func dataSourcePQLRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Refreshing PuppetDB Node: %s", d.Id())

	query := d.Get("query").(string)
	client := meta.(*PuppetDBClient)

	queryMap := map[string]string{
		"query": query,
	}
	queryStr, err := json.Marshal(&queryMap)
	if err != nil {
		return err
	}
	resp, err := client.Query("query/v4", "POST", string(queryStr))
	if err != nil {
		return err
	}

	md5sum := fmt.Sprintf("%x", md5.Sum(queryStr))
	d.SetId(md5sum)

	d.Set("data_json", string(resp))

	var respPdb []map[string]interface{}
	err = json.Unmarshal(resp, &respPdb)
	if err != nil {
		log.Printf("[RANCHER] unmarshaling failed: %v\n", err)
		d.Set("data", string(resp))
	} else {
		dataList := flattenMapList(respPdb)
		d.Set("data", dataList)
	}
	return nil
}

func flattenMapList(list []map[string]interface{}) []interface{} {
	vs := make([]interface{}, 0, len(list))
	for _, v := range list {
		// Cast everything as strings: UGLY!!
		m := make(map[string]string)
		for k, v := range v {
			m[k] = fmt.Sprintf("%v", v)
		}
		vs = append(vs, m)
	}
	return vs
}
