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
			"blocks": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"create_date": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"regions": &schema.Schema{
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
			},
			"services": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"sync_token": &schema.Schema{
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

	var (
		regions         = d.Get("regions").(*schema.Set)
		services        = d.Get("services").(*schema.Set)
		noRegionFilter  = regions.Len() == 0
		noServiceFilter = services.Len() == 0
		prefixes        []string
	)

	for _, e := range result.Prefixes {

		var (
			matchRegion  = noRegionFilter || regions.Contains(strings.ToLower(e.Region))
			matchService = noServiceFilter || services.Contains(strings.ToLower(e.Service))
		)

		if matchRegion && matchService {
			prefixes = append(prefixes, e.IpPrefix)
		}

	}

	if len(prefixes) == 0 {
		log.Printf("[WARN] No ip ranges result from filters")
	}

	sort.Strings(prefixes)

	if err := d.Set("blocks", prefixes); err != nil {
		return fmt.Errorf("Error setting ip ranges: %s", err)
	}

	return nil

}
