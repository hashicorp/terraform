package consul

import (
	"fmt"
	"sort"
	"strings"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
)

const (
	catalogServicesDatacenter = "datacenter"
	catalogServicesNames      = "names"

	catalogServicesServiceName = "name"
	catalogServicesServiceTags = "tags"
)

func dataSourceConsulCatalogServices() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceConsulCatalogServicesRead,
		Schema: map[string]*schema.Schema{
			// Data Source Predicate(s)
			catalogServicesDatacenter: &schema.Schema{
				// Used in the query, must be stored and force a refresh if the value
				// changes.
				Computed: true,
				Type:     schema.TypeString,
				ForceNew: true,
			},
			queryOpts: schemaQueryOpts,

			// Out parameters
			catalogServicesNames: &schema.Schema{
				Computed: true,
				Type:     schema.TypeMap,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						catalogServiceServiceTags: &schema.Schema{
							// FIXME(sean@): Tags is currently a space separated list of tags.
							// The ideal structure should be map[string][]string instead.
							// When this is supported in the future this should be changed to
							// be a TypeList instead.
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceConsulCatalogServicesRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*consulapi.Client)

	// Parse out data source filters to populate Consul's query options
	queryOpts, err := getQueryOpts(d, client)
	if err != nil {
		return errwrap.Wrapf("unable to get query options for fetching catalog services: {{err}}", err)
	}

	services, meta, err := client.Catalog().Services(queryOpts)
	if err != nil {
		return err
	}

	m := make(map[string]interface{}, len(services))
	for name, tags := range services {
		tagList := make([]string, 0, len(tags))
		for _, tag := range tags {
			tagList = append(tagList, tag)
		}

		sort.Strings(tagList)
		m[name] = strings.Join(tagList, " ")
	}

	const idKeyFmt = "catalog-services-%s"
	d.SetId(fmt.Sprintf(idKeyFmt, queryOpts.Datacenter))

	d.Set(catalogServicesDatacenter, queryOpts.Datacenter)
	if err := d.Set(catalogServicesNames, m); err != nil {
		return errwrap.Wrapf("Unable to store services: {{err}}", err)
	}

	return nil
}
