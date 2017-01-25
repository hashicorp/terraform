package circonus

import (
	"github.com/circonus-labs/circonus-gometrics/api"
	"github.com/circonus-labs/circonus-gometrics/api/config"
	"github.com/hashicorp/terraform/helper/schema"
)

const (
	collectorCNAttr           = "cn"
	collectorIDAttr           = "id"
	collectorDetailsAttr      = "details"
	collectorExternalHostAttr = "external_host"
	collectorExternalPortAttr = "external_port"
	collectorIPAttr           = "ip"
	collectorLatitudeAttr     = "latitude"
	collectorLongitudeAttr    = "longitude"
	collectorMinVersionAttr   = "min_version"
	collectorModulesAttr      = "modules"
	collectorNameAttr         = "name"
	collectorPortAttr         = "port"
	collectorSkewAttr         = "skew"
	collectorStatusAttr       = "status"
	collectorTagsAttr         = "tags"
	collectorTypeAttr         = "type"
	collectorVersionAttr      = "version"
)

func dataSourceCirconusCollector() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceCirconusCollectorRead,

		Schema: map[string]*schema.Schema{
			collectorIDAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: _ValidateRegexp(collectorIDAttr, config.BrokerCIDRegex),
				Description:  collectorDescription[collectorIDAttr],
			},
			collectorDetailsAttr: &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				Description: collectorDescription[collectorDetailsAttr],
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						collectorCNAttr: &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							Description: collectorDescription[collectorCNAttr],
						},
						collectorExternalHostAttr: &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							Description: collectorDescription[collectorExternalHostAttr],
						},
						collectorExternalPortAttr: &schema.Schema{
							Type:        schema.TypeInt,
							Optional:    true,
							Computed:    true,
							Description: collectorDescription[collectorExternalPortAttr],
						},
						collectorIPAttr: &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							Description: collectorDescription[collectorIPAttr],
						},
						collectorMinVersionAttr: &schema.Schema{
							Type:        schema.TypeInt,
							Optional:    true,
							Computed:    true,
							Description: collectorDescription[collectorMinVersionAttr],
						},
						collectorModulesAttr: &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Computed: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Description: collectorDescription[collectorModulesAttr],
						},
						collectorPortAttr: &schema.Schema{
							Type:        schema.TypeInt,
							Optional:    true,
							Computed:    true,
							Description: collectorDescription[collectorPortAttr],
						},
						collectorSkewAttr: &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							Description: collectorDescription[collectorSkewAttr],
						},
						collectorStatusAttr: &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							Description: collectorDescription[collectorStatusAttr],
						},
						collectorVersionAttr: &schema.Schema{
							Type:        schema.TypeInt,
							Optional:    true,
							Computed:    true,
							Description: collectorDescription[collectorVersionAttr],
						},
					},
				},
			},
			collectorLatitudeAttr: &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: collectorDescription[collectorLatitudeAttr],
			},
			collectorLongitudeAttr: &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: collectorDescription[collectorLongitudeAttr],
			},
			collectorNameAttr: &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: collectorDescription[collectorNameAttr],
			},

			// OUT parameters
			collectorTypeAttr: &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: collectorDescription[collectorTypeAttr],
			},
			collectorTagsAttr: _TagMakeConfigSchema(collectorTagsAttr),
		},
	}
}

func dataSourceCirconusCollectorRead(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*_ProviderContext)

	var collector *api.Broker
	var err error
	cid := d.Id()
	if cidRaw, ok := d.GetOk(collectorIDAttr); ok {
		cid = cidRaw.(string)
	}
	collector, err = ctxt.client.FetchBroker(api.CIDType(&cid))
	if err != nil {
		return err
	}

	_StateSet(d, collectorDetailsAttr, collectorDetailsToState(collector))
	_StateSet(d, collectorLatitudeAttr, collector.Latitude)
	_StateSet(d, collectorLongitudeAttr, collector.Longitude)
	_StateSet(d, collectorNameAttr, collector.Name)
	_StateSet(d, collectorTagsAttr, collector.Tags)
	_StateSet(d, collectorTypeAttr, collector.Type)

	d.SetId(collector.CID)

	return nil
}

func collectorDetailsToState(c *api.Broker) []interface{} {
	details := make([]interface{}, 0, len(c.Details))

	for _, collector := range c.Details {
		collectorDetails := make(map[string]interface{}, defaultCollectorDetailAttrs)

		collectorDetails[collectorCNAttr] = collector.CN

		if collector.ExternalHost != nil {
			collectorDetails[collectorExternalHostAttr] = *collector.ExternalHost
		}

		if collector.ExternalPort != 0 {
			collectorDetails[collectorExternalPortAttr] = collector.ExternalPort
		}

		if collector.IP != nil {
			collectorDetails[collectorIPAttr] = *collector.IP
		}

		if collector.MinVer != 0 {
			collectorDetails[collectorMinVersionAttr] = collector.MinVer
		}

		if len(collector.Modules) > 0 {
			collectorDetails[collectorModulesAttr] = collector.Modules
		}

		if collector.Port != nil {
			collectorDetails[collectorPortAttr] = *collector.Port
		}

		if collector.Skew != nil {
			collectorDetails[collectorSkewAttr] = *collector.Skew
		}

		if collector.Status != "" {
			collectorDetails[collectorStatusAttr] = collector.Status
		}

		if collector.Version != nil {
			collectorDetails[collectorVersionAttr] = *collector.Version
		}

		details = append(details, collectorDetails)
	}

	return details
}
