package clc

import (
	"fmt"
	"log"
	"time"

	clc "github.com/CenturyLinkCloud/clc-sdk"
	"github.com/CenturyLinkCloud/clc-sdk/lb"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceCLCLoadBalancer() *schema.Resource {
	return &schema.Resource{
		Create: resourceCLCLoadBalancerCreate,
		Read:   resourceCLCLoadBalancerRead,
		Update: resourceCLCLoadBalancerUpdate,
		Delete: resourceCLCLoadBalancerDelete,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"data_center": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			// optional
			"status": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "enabled",
			},
			// computed
			"ip_address": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceCLCLoadBalancerCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clc.Client)
	dc := d.Get("data_center").(string)
	name := d.Get("name").(string)
	desc := d.Get("description").(string)
	status := d.Get("status").(string)
	r1 := lb.LoadBalancer{
		Name:        name,
		Description: desc,
		Status:      status,
	}
	l, err := client.LB.Create(dc, r1)
	if err != nil {
		return fmt.Errorf("Failed creating load balancer under %v/%v: %v", dc, name, err)
	}
	d.SetId(l.ID)
	return resource.Retry(1*time.Minute, func() *resource.RetryError {
		_, err := client.LB.Get(dc, l.ID)
		if err != nil {
			return resource.RetryableError(err)
		}
		err = resourceCLCLoadBalancerRead(d, meta)
		if err != nil {
			return resource.NonRetryableError(err)
		}
		return nil
	})
}

func resourceCLCLoadBalancerRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clc.Client)
	dc := d.Get("data_center").(string)
	id := d.Id()
	resp, err := client.LB.Get(dc, id)
	if err != nil {
		log.Printf("[INFO] Failed finding load balancer %v/%v. Marking destroyed", dc, id)
		d.SetId("")
		return nil
	}
	d.Set("description", resp.Description)
	d.Set("ip_address", resp.IPaddress)
	d.Set("status", resp.Status)
	d.Set("pools", resp.Pools)
	d.Set("links", resp.Links)
	return nil
}

func resourceCLCLoadBalancerUpdate(d *schema.ResourceData, meta interface{}) error {
	update := lb.LoadBalancer{}
	client := meta.(*clc.Client)
	dc := d.Get("data_center").(string)
	id := d.Id()

	if d.HasChange("name") {
		update.Name = d.Get("name").(string)
	}
	if d.HasChange("description") {
		update.Description = d.Get("description").(string)
	}
	if d.HasChange("status") {
		update.Status = d.Get("status").(string)
	}
	if update.Name != "" || update.Description != "" || update.Status != "" {
		update.Name = d.Get("name").(string) // required on every PUT
		err := client.LB.Update(dc, id, update)
		if err != nil {
			return fmt.Errorf("Failed updating load balancer under %v/%v: %v", dc, id, err)
		}
	}
	return resourceCLCLoadBalancerRead(d, meta)
}

func resourceCLCLoadBalancerDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clc.Client)
	dc := d.Get("data_center").(string)
	id := d.Id()
	err := client.LB.Delete(dc, id)
	if err != nil {
		return fmt.Errorf("Failed deleting loadbalancer %v: %v", id, err)
	}
	return nil
}
