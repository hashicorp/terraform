package consul

import (
	consulapi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceConsulPreparedQuery() *schema.Resource {
	return &schema.Resource{
		Create: resourceConsulPreparedQueryCreate,
		Update: resourceConsulPreparedQueryUpdate,
		Read:   resourceConsulPreparedQueryRead,
		Delete: resourceConsulPreparedQueryDelete,

		SchemaVersion: 0,

		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"session": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"token": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"service": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"service": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"near": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"failover": &schema.Schema{
							Type:     schema.TypeSet,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"nearest_n": &schema.Schema{
										Type:     schema.TypeInt,
										Optional: true,
									},

									"datacenters": &schema.Schema{
										Type:     schema.TypeSet,
										Optional: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
										Set:      schema.HashString,
									},
								},
							},
						},

						"only_passing": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
						},

						"tags": &schema.Schema{
							Type:     schema.TypeSet,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
					},
				},
			},

			"dns": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ttl": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},

			"template": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"regexp": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
		},
	}
}

func resourceConsulPreparedQueryCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*consulapi.Client)
	token := d.Get("token").(string)

	pq := &consulapi.PreparedQueryDefinition{
		Name:    d.Get("name").(string),
		Session: d.Get("session").(string),
		Token:   token,
		Service: consulapi.ServiceQuery{
			Service:     d.Get("service.service").(string),
			Near:        d.Get("service.near").(string),
			OnlyPassing: d.Get("service.only_passing").(bool),
			Tags:        d.Get("service.tags").([]string),
		},
	}

	if _, ok := d.GetOk("service.failover"); ok {
		pq.Service.Failover.NearestN = d.Get("service.failover.nearest_n").(int)
		pq.Service.Failover.Datacenters = d.Get("service.failover.datacenters").([]string)
	}

	if _, ok := d.GetOk("dns"); ok {
		if v, ok := d.GetOk("dns.ttl"); ok {
			pq.DNS.TTL = v.(string)
		}
	}

	id, _, err := client.PreparedQuery().Create(pq, &consulapi.WriteOptions{Token: token})
	if err != nil {
		return err
	}

	d.SetId(id)

	return resourceConsulPreparedQueryRead(d, meta)
}

func resourceConsulPreparedQueryUpdate(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceConsulPreparedQueryRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*consulapi.Client)
	token := d.Get("token").(string)

	queries, _, err := client.PreparedQuery().Get(d.Id(), &consulapi.QueryOptions{Token: token})
	if err != nil {
		return err
	}

	if len(queries) != 1 {
		d.SetId("")
	}
	pq := queries[0]

	d.Set("name", pq.Name)
	d.Set("session", pq.Session)
	d.Set("token", token)
	d.Set("service.service", pq.Service.Service)
	d.Set("service.near", pq.Service.Near)
	d.Set("service.only_passing", pq.Service.OnlyPassing)
	d.Set("service.tags", pq.Service.Tags)
	d.Set("service.failover.nearest_n", pq.Service.Failover.NearestN)
	d.Set("service.failover.datacenters", pq.Service.Failover.Datacenters)
	d.Set("dns.ttl", pq.DNS.TTL)

	return nil
}

func resourceConsulPreparedQueryDelete(d *schema.ResourceData, meta interface{}) error {
	return nil
}
