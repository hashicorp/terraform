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
	CreateDate   string
	Prefixes     []dataSourceAwsIPRangesPrefix
	Ipv6Prefixes []dataSourceAwsIPRangesIpv6Prefix `json:"ipv6_prefixes"`
	SyncToken    string
}

type dataSourceAwsIPRangesPrefix struct {
	IpPrefix string `json:"ip_prefix"`
	Region   string
	Service  string
}

type dataSourceAwsIPRangesIpv6Prefix struct {
	Ipv6Prefix string `json:"ipv6_prefix"`
	Region     string
	Service    string
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
			"ipv6_cidr_blocks": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
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
		ipPrefixes     []string
		ipv6Prefixes   []string
	)

	matchFilter := func(region, service string) bool {
		matchRegion := noRegionFilter || regions.Contains(strings.ToLower(region))
		matchService := services.Contains(strings.ToLower(service))
		return matchRegion && matchService
	}

	for _, e := range result.Prefixes {
		if matchFilter(e.Region, e.Service) {
			ipPrefixes = append(ipPrefixes, e.IpPrefix)
		}
	}

	for _, e := range result.Ipv6Prefixes {
		if matchFilter(e.Region, e.Service) {
			ipv6Prefixes = append(ipv6Prefixes, e.Ipv6Prefix)
		}
	}

	if len(ipPrefixes) == 0 && len(ipv6Prefixes) == 0 {
		return fmt.Errorf("No IP ranges result from filters")
	}

	sort.Strings(ipPrefixes)

	if err := d.Set("cidr_blocks", ipPrefixes); err != nil {
		return fmt.Errorf("Error setting cidr_blocks: %s", err)
	}

	sort.Strings(ipv6Prefixes)

	if err := d.Set("ipv6_cidr_blocks", ipv6Prefixes); err != nil {
		return fmt.Errorf("Error setting ipv6_cidr_blocks: %s", err)
	}

	return nil

}
