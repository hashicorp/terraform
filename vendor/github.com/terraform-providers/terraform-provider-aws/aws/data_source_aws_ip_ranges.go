package aws

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/terraform/helper/schema"
)

type dataSourceAwsIPRangesResult struct {
	CreateDate string
	Prefixes   []dataSourceAwsIPRangesPrefix
	SyncToken  string
}

type dataSourceAwsIPRangesPrefix struct {
	IpPrefix string `json:"ip_prefix"`
	Region   string
	Service  string
}

func dataSourceAwsIPRanges() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsIPRangesRead,

		Schema: map[string]*schema.Schema{
			"cidr_blocks": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"create_date": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"regions": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
			},
			"services": {
				Type:     schema.TypeSet,
				Required: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"sync_token": {
				Type:     schema.TypeInt,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsIPRangesRead(d *schema.ResourceData, meta interface{}) error {

	conn := cleanhttp.DefaultClient()

	log.Printf("[DEBUG] Reading IP ranges")

	res, err := conn.Get("https://ip-ranges.amazonaws.com/ip-ranges.json")

	if err != nil {
		return fmt.Errorf("Error listing IP ranges: %s", err)
	}

	defer res.Body.Close()

	data, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return fmt.Errorf("Error reading response body: %s", err)
	}

	result := new(dataSourceAwsIPRangesResult)

	if err := json.Unmarshal(data, result); err != nil {
		return fmt.Errorf("Error parsing result: %s", err)
	}

	if err := d.Set("create_date", result.CreateDate); err != nil {
		return fmt.Errorf("Error setting create date: %s", err)
	}

	syncToken, err := strconv.Atoi(result.SyncToken)

	if err != nil {
		return fmt.Errorf("Error while converting sync token: %s", err)
	}

	d.SetId(result.SyncToken)

	if err := d.Set("sync_token", syncToken); err != nil {
		return fmt.Errorf("Error setting sync token: %s", err)
	}

	get := func(key string) *schema.Set {

		set := d.Get(key).(*schema.Set)

		for _, e := range set.List() {

			s := e.(string)

			set.Remove(s)
			set.Add(strings.ToLower(s))

		}

		return set

	}

	var (
		regions        = get("regions")
		services       = get("services")
		noRegionFilter = regions.Len() == 0
		prefixes       []string
	)

	for _, e := range result.Prefixes {

		var (
			matchRegion  = noRegionFilter || regions.Contains(strings.ToLower(e.Region))
			matchService = services.Contains(strings.ToLower(e.Service))
		)

		if matchRegion && matchService {
			prefixes = append(prefixes, e.IpPrefix)
		}

	}

	if len(prefixes) == 0 {
		return fmt.Errorf(" No IP ranges result from filters")
	}

	sort.Strings(prefixes)

	if err := d.Set("cidr_blocks", prefixes); err != nil {
		return fmt.Errorf("Error setting ip ranges: %s", err)
	}

	return nil

}
