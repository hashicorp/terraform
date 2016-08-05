package fastly

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"sort"
	"strconv"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

type dataSourceFastlyIPRangesResult struct {
	Addresses []string
}

func dataSourceFastlyIPRanges() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceFastlyIPRangesRead,

		Schema: map[string]*schema.Schema{
			"cidr_blocks": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func dataSourceFastlyIPRangesRead(d *schema.ResourceData, meta interface{}) error {

	conn := cleanhttp.DefaultClient()

	log.Printf("[DEBUG] Reading IP ranges")

	res, err := conn.Get("https://api.fastly.com/public-ip-list")

	if err != nil {
		return fmt.Errorf("Error listing IP ranges: %s", err)
	}

	defer res.Body.Close()

	data, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return fmt.Errorf("Error reading response body: %s", err)
	}

	d.SetId(strconv.Itoa(hashcode.String(string(data))))

	result := new(dataSourceFastlyIPRangesResult)

	if err := json.Unmarshal(data, result); err != nil {
		return fmt.Errorf("Error parsing result: %s", err)
	}

	sort.Strings(result.Addresses)

	if err := d.Set("cidr_blocks", result.Addresses); err != nil {
		return fmt.Errorf("Error setting ip ranges: %s", err)
	}

	return nil

}
