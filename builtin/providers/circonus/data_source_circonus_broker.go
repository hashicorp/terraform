package circonus

import (
	"github.com/circonus-labs/circonus-gometrics/api"
	"github.com/hashicorp/terraform/helper/schema"
)

const (
	brokerCIDAttr          = "cid"
	brokerCNAttr           = "cn"
	brokerDetailsAttr      = "details"
	brokerExternalHostAttr = "external_host"
	brokerExternalPortAttr = "external_port"
	brokerIPAttr           = "ip"
	brokerLatitudeAttr     = "latitude"
	brokerLongitudeAttr    = "longitude"
	brokerMinVersionAttr   = "min_version"
	brokerModulesAttr      = "modules"
	brokerNameAttr         = "name"
	brokerPortAttr         = "port"
	brokerSkewAttr         = "skew"
	brokerStatusAttr       = "status"
	brokerTagsAttr         = "tags"
	brokerTypeAttr         = "type"
	brokerVersionAttr      = "version"
)

func dataSourceCirconusBroker() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceCirconusBrokerRead,

		Schema: map[string]*schema.Schema{
			brokerCIDAttr: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			brokerDetailsAttr: &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				Description: "Details associated with a broker in the broker group",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						brokerCNAttr: &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
						brokerExternalHostAttr: &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
						brokerExternalPortAttr: &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
						},
						brokerIPAttr: &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
						brokerMinVersionAttr: &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
						},
						brokerModulesAttr: &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Computed: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						brokerPortAttr: &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
						},
						brokerSkewAttr: &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
						brokerStatusAttr: &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
						brokerVersionAttr: &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
						},
					},
				},
			},
			brokerLatitudeAttr: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			brokerLongitudeAttr: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			brokerNameAttr: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			brokerTagsAttr: &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				Description: "Tags assigned to a broker",
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validateTag,
				},
			},
		},
	}
}

func dataSourceCirconusBrokerRead(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*api.API)

	var b *api.Broker
	var err error
	if cidRaw, ok := d.GetOk(brokerCIDAttr); ok {
		cid := cidRaw.(string)
		b, err = c.FetchBroker(api.CIDType(&cid))
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
