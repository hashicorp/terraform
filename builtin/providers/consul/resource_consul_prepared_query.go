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

			"failover_nearest_n": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},

			"failover_datacenters": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"dns_ttl": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
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
			Service:     d.Get("service").(string),
			Near:        d.Get("near").(string),
			OnlyPassing: d.Get("only_passing").(bool),
			Failover: consulapi.QueryDatacenterOptions{
				NearestN: d.Get("failover_nearest_n").(int),
			},
		},
		DNS: consulapi.QueryDNSOptions{
			TTL: d.Get("dns_ttl").(string),
		},
	}

	tags := d.Get("tags").(*schema.Set).List()
	pq.Service.Tags = make([]string, len(tags))
	for i, v := range tags {
		pq.Service.Tags[i] = v.(string)
	}

	failoverDatacenters := d.Get("failover_datacenters").(*schema.Set).List()
	pq.Service.Failover.Datacenters = make([]string, len(failoverDatacenters))
	for i, v := range failoverDatacenters {
		pq.Service.Failover.Datacenters[i] = v.(string)
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
	d.Set("service", pq.Service.Service)
	d.Set("near", pq.Service.Near)
	d.Set("only_passing", pq.Service.OnlyPassing)
	d.Set("tags", pq.Service.Tags)
	d.Set("failover_nearest_n", pq.Service.Failover.NearestN)
	d.Set("failover_datacenters", pq.Service.Failover.Datacenters)
	d.Set("dns_ttl", pq.DNS.TTL)

	return nil
}

func resourceConsulPreparedQueryDelete(d *schema.ResourceData, meta interface{}) error {
	return nil
}
