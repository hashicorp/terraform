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
	wo := &consulapi.WriteOptions{Token: d.Get("token").(string)}

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
	wo := &consulapi.WriteOptions{Token: d.Get("token").(string)}

	pq := preparedQueryDefinitionFromResourceData(d)

	if _, err := client.PreparedQuery().Update(pq, wo); err != nil {
		return err
	}

	return resourceConsulPreparedQueryRead(d, meta)
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
		return nil
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

	if pq.Template.Type != "" {
		d.Set("template.0.type", pq.Template.Type)
		d.Set("template.0.regexp", pq.Template.Regexp)
	}

	return nil
}

func resourceConsulPreparedQueryDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*consulapi.Client)
	qo := &consulapi.QueryOptions{Token: d.Get("token").(string)}
	if _, err := client.PreparedQuery().Delete(d.Id(), qo); err != nil {
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
		Token:   d.Get("token").(string),
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

	if v, ok := d.GetOk("template"); ok {
		tpl := v.([]interface{})[0].(map[string]interface{})
		pq.Template = consulapi.QueryTemplate{
			Type:   tpl["type"].(string),
			Regexp: tpl["regexp"].(string),
		}
	}

	return pq
}
