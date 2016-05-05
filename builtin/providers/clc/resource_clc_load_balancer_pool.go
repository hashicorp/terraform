package clc

import (
	"fmt"
	"log"
	"strconv"

	clc "github.com/CenturyLinkCloud/clc-sdk"
	"github.com/CenturyLinkCloud/clc-sdk/lb"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceCLCLoadBalancerPool() *schema.Resource {
	return &schema.Resource{
		Create: resourceCLCLoadBalancerPoolCreate,
		Read:   resourceCLCLoadBalancerPoolRead,
		Update: resourceCLCLoadBalancerPoolUpdate,
		Delete: resourceCLCLoadBalancerPoolDelete,
		Schema: map[string]*schema.Schema{
			// pool args
			"port": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},
			"data_center": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"load_balancer": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"method": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "roundRobin",
			},
			"persistence": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "standard",
			},
			"nodes": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				Elem:     &schema.Schema{Type: schema.TypeMap},
			},
		},
	}
}

func resourceCLCLoadBalancerPoolCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clc.Client)
	dc := d.Get("data_center").(string)
	lbid := d.Get("load_balancer").(string)

	s1 := d.Get("method").(string)
	m := lb.LeastConn
	if s1 == string(lb.RoundRobin) {
		m = lb.RoundRobin
	}
	s2 := d.Get("persistence").(string)
	p := lb.Standard
	if s2 == string(lb.Sticky) {
		p = lb.Sticky
	}
	r2 := lb.Pool{
		Port:        d.Get("port").(int),
		Method:      m,
		Persistence: p,
	}
	lbp, err := client.LB.CreatePool(dc, lbid, r2)
	if err != nil {
		return fmt.Errorf("Failed creating pool under %v/%v: %v", dc, lbid, err)
	}
	d.SetId(lbp.ID)
	return resourceCLCLoadBalancerPoolUpdate(d, meta)
}

func resourceCLCLoadBalancerPoolRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clc.Client)
	dc := d.Get("data_center").(string)
	lbid := d.Get("load_balancer").(string)
	id := d.Id()
	pool, err := client.LB.GetPool(dc, lbid, id)
	if err != nil {
		log.Printf("[INFO] Failed fetching pool %v/%v. Marking destroyed", lbid, d.Id())
		d.SetId("")
		return nil
	}
	nodes, err := client.LB.GetAllNodes(dc, lbid, id)
	nodes2 := make([]lb.Node, len(nodes))
	for i, n := range nodes {
		nodes2[i] = *n
	}
	pool.Nodes = nodes2
	d.Set("port", pool.Port)
	d.Set("method", pool.Method)
	d.Set("persistence", pool.Persistence)
	d.Set("nodes", pool.Nodes)
	d.Set("links", pool.Links)
	return nil
}

func resourceCLCLoadBalancerPoolUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clc.Client)
	dc := d.Get("data_center").(string)
	lbid := d.Get("load_balancer").(string)
	id := d.Id()
	pool, err := client.LB.GetPool(dc, lbid, d.Id())
	pool.Port = 0 // triggers empty value => omission from POST

	if d.HasChange("method") {
		d.SetPartial("method")
		pool.Method = lb.Method(d.Get("method").(string))
	}
	if d.HasChange("persistence") {
		d.SetPartial("persistence")
		pool.Persistence = lb.Persistence(d.Get("persistence").(string))
	}
	err = client.LB.UpdatePool(dc, lbid, id, *pool)
	if err != nil {
		return fmt.Errorf("Failed updating pool %v: %v", id, err)
	}

	if d.HasChange("nodes") {
		d.SetPartial("nodes")
		nodes, err := parseNodes(d)
		if err != nil {
			return err
		}
		err = client.LB.UpdateNodes(dc, lbid, id, nodes...)
		if err != nil {
			return fmt.Errorf("Failed updating pool nodes %v: %v", id, err)
		}
	}
	return resourceCLCLoadBalancerPoolRead(d, meta)
}

func resourceCLCLoadBalancerPoolDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clc.Client)
	dc := d.Get("data_center").(string)
	lbid := d.Get("load_balancer").(string)
	id := d.Id()
	err := client.LB.DeletePool(dc, lbid, id)
	if err != nil {
		return fmt.Errorf("Failed deleting pool %v: %v", id, err)
	}
	return nil
}

func parseNodes(d *schema.ResourceData) ([]lb.Node, error) {
	var nodes []lb.Node
	raw := d.Get("nodes")
	if raw == nil {
		log.Println("WARNING: pool missing nodes")
		return nil, nil
	}
	if arr, ok := raw.([]interface{}); ok {
		for _, v := range arr {
			m := v.(map[string]interface{})
			p, err := strconv.Atoi(m["privatePort"].(string))
			if err != nil {
				log.Printf("[WARN] Failed parsing port '%v'. skipping", m["privatePort"])
				continue
			}
			n := lb.Node{
				Status:      m["status"].(string),
				IPaddress:   m["ipAddress"].(string),
				PrivatePort: p,
			}
			nodes = append(nodes, n)
		}
	} else {
		return nil, fmt.Errorf("Failed parsing nodes from pool spec: %v", raw)
	}
	return nodes, nil
}
