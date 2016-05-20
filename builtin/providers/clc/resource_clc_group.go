package clc

import (
	"fmt"
	"log"
	"time"

	"github.com/CenturyLinkCloud/clc-sdk"
	"github.com/CenturyLinkCloud/clc-sdk/api"
	"github.com/CenturyLinkCloud/clc-sdk/group"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceCLCGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceCLCGroupCreate,
		Read:   resourceCLCGroupRead,
		Update: resourceCLCGroupUpdate,
		Delete: resourceCLCGroupDelete,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"parent": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"location_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"parent_group_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"custom_fields": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeMap},
			},
		},
	}
}

func resourceCLCGroupCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clc.Client)
	name := d.Get("name").(string)
	desc := d.Get("description").(string)
	parent := d.Get("parent").(string)
	dc := d.Get("location_id").(string)

	// clc doesn't enforce uniqueness by name
	// so skip the trad'l error we'd raise
	e, err := resolveGroupByNameOrId(name, dc, client)
	if e != "" {
		log.Printf("[INFO] Resolved existing group: %v => %v", name, e)
		d.SetId(e)
		return nil
	}

	var pgid string
	p, err := resolveGroupByNameOrId(parent, dc, client)
	if p != "" {
		log.Printf("[INFO] Resolved parent group: %v => %v", parent, p)
		pgid = p
	} else {
		return fmt.Errorf("Failed resolving parent group %s - %s err:%s", parent, p, err)
	}

	d.Set("parent_group_id", pgid)
	spec := group.Group{
		Name:          name,
		Description:   desc,
		ParentGroupID: pgid,
	}
	resp, err := client.Group.Create(spec)
	if err != nil {
		return fmt.Errorf("Failed creating group: %s", err)
	}
	log.Println("[INFO] Group created")
	d.SetId(resp.ID)
	return nil
}

func resourceCLCGroupRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clc.Client)
	id := d.Id()
	g, err := client.Group.Get(id)
	if err != nil {
		log.Printf("[INFO] Failed finding group: %s -  %s. Marking destroyed", id, err)
		d.SetId("")
		return nil
	}
	d.Set("name", g.Name)
	d.Set("description", g.Description)
	d.Set("parent_group_id", g.ParentGroupID())
	return nil
}

func resourceCLCGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clc.Client)
	id := d.Id()
	var err error
	var patches []api.Update

	g, err := client.Group.Get(id)
	if err != nil {
		return fmt.Errorf("Failed fetching group: %v - %v", id, err)
	}

	if delta, orig := d.Get("name").(string), g.Name; delta != orig {
		patches = append(patches, group.UpdateName(delta))
	}
	if delta, orig := d.Get("description").(string), g.Description; delta != orig {
		patches = append(patches, group.UpdateDescription(delta))
	}
	newParent := d.Get("parent").(string)
	pgid, err := resolveGroupByNameOrId(newParent, g.Locationid, client)
	log.Printf("[DEBUG] PARENT current:%v new:%v resolved:%v", g.ParentGroupID(), newParent, pgid)
	if pgid == "" {
		return fmt.Errorf("Unable to resolve parent group %v: %v", newParent, err)
	} else if newParent != g.ParentGroupID() {
		patches = append(patches, group.UpdateParentGroupID(pgid))
	}

	if len(patches) == 0 {
		return nil
	}
	err = client.Group.Update(id, patches...)
	if err != nil {
		return fmt.Errorf("Failed updating group %v: %v", id, err)
	}
	return resource.Retry(1*time.Minute, func() *resource.RetryError {
		_, err := client.Group.Get(id)
		if err != nil {
			return resource.RetryableError(err)
		}
		err = resourceCLCGroupRead(d, meta)
		if err != nil {
			return resource.NonRetryableError(err)
		}
		return nil
	})
}

func resourceCLCGroupDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clc.Client)
	id := d.Id()
	log.Printf("[INFO] Deleting group %v", id)
	st, err := client.Group.Delete(id)
	if err != nil {
		return fmt.Errorf("Failed deleting group: %v with err: %v", id, err)
	}
	waitStatus(client, st.ID)
	return nil
}
