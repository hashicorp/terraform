package consul

import (
	"strings"

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

			"datacenter": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
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

			"stored_token": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"service": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"tags": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"near": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"only_passing": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},

			"failover": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"nearest_n": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},
						"datacenters": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},

			"dns": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
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
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"regexp": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func resourceConsulPreparedQueryCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*consulapi.Client)
	wo := &consulapi.WriteOptions{
		Datacenter: d.Get("datacenter").(string),
		Token:      d.Get("token").(string),
	}

	pq := preparedQueryDefinitionFromResourceData(d)

	id, _, err := client.PreparedQuery().Create(pq, wo)
	if err != nil {
		return err
	}

	d.SetId(id)
	return resourceConsulPreparedQueryRead(d, meta)
}

func resourceConsulPreparedQueryUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*consulapi.Client)
	wo := &consulapi.WriteOptions{
		Datacenter: d.Get("datacenter").(string),
		Token:      d.Get("token").(string),
	}

	pq := preparedQueryDefinitionFromResourceData(d)

	if _, err := client.PreparedQuery().Update(pq, wo); err != nil {
		return err
	}

	return resourceConsulPreparedQueryRead(d, meta)
}

func resourceConsulPreparedQueryRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*consulapi.Client)
	qo := &consulapi.QueryOptions{
		Datacenter: d.Get("datacenter").(string),
		Token:      d.Get("token").(string),
	}

	queries, _, err := client.PreparedQuery().Get(d.Id(), qo)
	if err != nil {
		// Check for a 404/not found, these are returned as errors.
		if strings.Contains(err.Error(), "not found") {
			d.SetId("")
			return nil
		}
		return err
	}

	if len(queries) != 1 {
		d.SetId("")
		return nil
	}
	pq := queries[0]

	d.Set("name", pq.Name)
	d.Set("session", pq.Session)
	d.Set("stored_token", pq.Token)
	d.Set("service", pq.Service.Service)
	d.Set("near", pq.Service.Near)
	d.Set("only_passing", pq.Service.OnlyPassing)
	d.Set("tags", pq.Service.Tags)

	if pq.Service.Failover.NearestN > 0 {
		d.Set("failover.0.nearest_n", pq.Service.Failover.NearestN)
	}
	if len(pq.Service.Failover.Datacenters) > 0 {
		d.Set("failover.0.datacenters", pq.Service.Failover.Datacenters)
	}

	if pq.DNS.TTL != "" {
		d.Set("dns.0.ttl", pq.DNS.TTL)
	}

	if pq.Template.Type != "" {
		d.Set("template.0.type", pq.Template.Type)
		d.Set("template.0.regexp", pq.Template.Regexp)
	}

	return nil
}

func resourceConsulPreparedQueryDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*consulapi.Client)
	writeOpts := &consulapi.WriteOptions{
		Datacenter: d.Get("datacenter").(string),
		Token:      d.Get("token").(string),
	}

	if _, err := client.PreparedQuery().Delete(d.Id(), writeOpts); err != nil {
		return err
	}

	d.SetId("")
	return nil
}

func preparedQueryDefinitionFromResourceData(d *schema.ResourceData) *consulapi.PreparedQueryDefinition {
	pq := &consulapi.PreparedQueryDefinition{
		ID:      d.Id(),
		Name:    d.Get("name").(string),
		Session: d.Get("session").(string),
		Token:   d.Get("stored_token").(string),
		Service: consulapi.ServiceQuery{
			Service:     d.Get("service").(string),
			Near:        d.Get("near").(string),
			OnlyPassing: d.Get("only_passing").(bool),
		},
	}

	tags := d.Get("tags").(*schema.Set).List()
	pq.Service.Tags = make([]string, len(tags))
	for i, v := range tags {
		pq.Service.Tags[i] = v.(string)
	}

	if _, ok := d.GetOk("failover.0"); ok {
		failover := consulapi.QueryDatacenterOptions{
			NearestN: d.Get("failover.0.nearest_n").(int),
		}

		dcs := d.Get("failover.0.datacenters").([]interface{})
		failover.Datacenters = make([]string, len(dcs))
		for i, v := range dcs {
			failover.Datacenters[i] = v.(string)
		}

		pq.Service.Failover = failover
	}

	if _, ok := d.GetOk("template.0"); ok {
		pq.Template = consulapi.QueryTemplate{
			Type:   d.Get("template.0.type").(string),
			Regexp: d.Get("template.0.regexp").(string),
		}
	}

	if _, ok := d.GetOk("dns.0"); ok {
		pq.DNS = consulapi.QueryDNSOptions{
			TTL: d.Get("dns.0.ttl").(string),
		}
	}

	return pq
}
