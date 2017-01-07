package circonus

import (
	"github.com/circonus-labs/circonus-gometrics/api"
	"github.com/hashicorp/terraform/helper/schema"
)

const (
	collectorCIDAttr          = "cid"
	collectorCNAttr           = "cn"
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
			collectorCIDAttr: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
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
			collectorTagsAttr: &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validateTag,
				},
				Description: collectorDescription[collectorTagsAttr],
			},
		},
	}
}

func dataSourceCirconusCollectorRead(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*providerContext)

	var b *api.Broker
	var err error
	if cidRaw, ok := d.GetOk(collectorCIDAttr); ok {
		cid := cidRaw.(string)
		b, err = c.client.FetchBroker(api.CIDType(&cid))
		if err != nil {
			return err
		}
	}

	d.Set("cid", b.CID)
	d.Set("details", b.Details)
	d.Set("latitude", b.Latitude)
	d.Set("longitude", b.Longitude)
	d.Set("name", b.Name)
	d.Set("tags", b.Tags)
	d.Set("type", b.Type)

	d.SetId(b.CID)

	return nil
}
