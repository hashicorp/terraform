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
	// Datasource predicates
	catalogServicesServiceName = "name"

	// Out parameters
	catalogServicesDatacenter  = "datacenter"
	catalogServicesNames       = "names"
	catalogServicesServices    = "services"
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
			catalogNodesQueryOpts: schemaQueryOpts,

			// Out parameters
			catalogServicesNames: &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			catalogServicesServices: &schema.Schema{
				Computed: true,
				Type:     schema.TypeMap,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						catalogServiceServiceTags: &schema.Schema{
							Type:     schema.TypeList,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
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

	catalogServices := make(map[string]interface{}, len(services))
	for name, tags := range services {
		tagList := make([]string, 0, len(tags))
		for _, tag := range tags {
			tagList = append(tagList, tag)
		}

		sort.Strings(tagList)
		catalogServices[name] = strings.Join(tagList, " ")
	}

	serviceNames := make([]interface{}, 0, len(services))
	for k := range catalogServices {
		serviceNames = append(serviceNames, k)
	}

	const idKeyFmt = "catalog-services-%s"
	d.SetId(fmt.Sprintf(idKeyFmt, queryOpts.Datacenter))

	d.Set(catalogServicesDatacenter, queryOpts.Datacenter)
	if err := d.Set(catalogServicesServices, catalogServices); err != nil {
		return errwrap.Wrapf("Unable to store services: {{err}}", err)
	}

	if err := d.Set(catalogServicesNames, serviceNames); err != nil {
		return errwrap.Wrapf("Unable to store service names: {{err}}", err)
	}

	return nil
}
