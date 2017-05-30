package icanhazip

import (
	"fmt"
	"io/ioutil"
	"log"
	"strconv"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"strings"
)

func dataSourceIcanhazipIPAddress() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceIcanhazipIPAddressRead,

		Schema: map[string]*schema.Schema{
			"ip_address": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceIcanhazipIPAddressRead(d *schema.ResourceData, meta interface{}) error {

	conn := cleanhttp.DefaultClient()

	version := d.Get("version").(string)

	log.Printf("[DEBUG] Fetching %s IP address", version)

	res, err := conn.Get(fmt.Sprintf("https://%s.icanhazip.com/", version))

	if err != nil {
		return fmt.Errorf("Failed to retrieve %s IP address: %s", version, err)
	}

	defer res.Body.Close()

	data, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return fmt.Errorf("Error reading response body: %s", err)
	}

	d.SetId(strconv.Itoa(hashcode.String(string(data))))

	result := strings.Trim(string(data), "\n")

	if err := d.Set("ip_address", result); err != nil {
		return fmt.Errorf("Error setting ip address: %s", err)
	}

	return nil
}
