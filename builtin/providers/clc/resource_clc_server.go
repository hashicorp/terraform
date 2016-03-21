package clc

import (
	"fmt"
	"log"
	"strings"

	clc "github.com/CenturyLinkCloud/clc-sdk"
	"github.com/CenturyLinkCloud/clc-sdk/api"
	"github.com/CenturyLinkCloud/clc-sdk/server"
	"github.com/CenturyLinkCloud/clc-sdk/status"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceCLCServer() *schema.Resource {
	return &schema.Resource{
		Create: resourceCLCServerCreate,
		Read:   resourceCLCServerRead,
		Update: resourceCLCServerUpdate,
		Delete: resourceCLCServerDelete,
		Schema: map[string]*schema.Schema{
			"name_template": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"group_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"source_server_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"cpu": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},
			"memory_mb": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},
			"password": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			// optional
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "standard",
			},
			"network_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"custom_fields": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeMap},
			},
			"additional_disks": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeMap},
			},

			// optional: misc state storage. non-CLC field
			"metadata": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
			},

			// optional
			"storage_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "standard",
			},

			// sorta computed
			"private_ip_address": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				Default:  nil,
			},
			"power_state": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				Default:  nil,
			},

			// computed
			"created_date": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"modified_date": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"public_ip_address": &schema.Schema{
				// RO: if a public_ip is on this server, populate it
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceCLCServerCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clc.Client)
	spec := server.Server{
		Name:           d.Get("name_template").(string),
		Password:       d.Get("password").(string),
		Description:    d.Get("description").(string),
		GroupID:        d.Get("group_id").(string),
		CPU:            d.Get("cpu").(int),
		MemoryGB:       d.Get("memory_mb").(int) / 1024,
		SourceServerID: d.Get("source_server_id").(string),
		Type:           d.Get("type").(string),
		IPaddress:      d.Get("private_ip_address").(string),
		NetworkID:      d.Get("network_id").(string),
		Storagetype:    d.Get("storage_type").(string),
	}

	var err error
	disks, err := parseAdditionalDisks(d)
	if err != nil {
		return fmt.Errorf("Failed parsing disks: %v", err)
	}
	spec.Additionaldisks = disks
	fields, err := parseCustomFields(d)
	if err != nil {
		return fmt.Errorf("Failed setting customfields: %v", err)
	}
	spec.Customfields = fields

	resp, err := client.Server.Create(spec)
	if err != nil || !resp.IsQueued {
		return fmt.Errorf("Failed creating server: %v", err)
	}
	// server's UUID returned under rel=self link
	_, uuid := resp.Links.GetID("self")

	ok, st := resp.GetStatusID()
	if !ok {
		return fmt.Errorf("Failed extracting status to poll on %v: %v", resp, err)
	}
	err = waitStatus(client, st)
	if err != nil {
		return err
	}

	s, err := client.Server.Get(uuid)
	d.SetId(strings.ToUpper(s.Name))
	log.Printf("[INFO] Server created. id: %v", s.Name)
	return resourceCLCServerRead(d, meta)
}

func resourceCLCServerRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clc.Client)
	s, err := client.Server.Get(d.Id())
	if err != nil {
		log.Printf("[INFO] Failed finding server: %v. Marking destroyed", d.Id())
		d.SetId("")
		return nil
	}
	if len(s.Details.IPaddresses) > 0 {
		d.Set("private_ip_address", s.Details.IPaddresses[0].Internal)
		if "" != s.Details.IPaddresses[0].Public {
			d.Set("public_ip_address", s.Details.IPaddresses[0].Public)
		}
	}

	d.Set("name", s.Name)
	d.Set("groupId", s.GroupID)
	d.Set("status", s.Status)
	d.Set("power_state", s.Details.Powerstate)
	d.Set("cpu", s.Details.CPU)
	d.Set("memory_mb", s.Details.MemoryMB)
	d.Set("disk_gb", s.Details.Storagegb)
	d.Set("status", s.Status)
	d.Set("storage_type", s.Storagetype)
	d.Set("created_date", s.ChangeInfo.CreatedDate)
	d.Set("modified_date", s.ChangeInfo.ModifiedDate)
	return nil
}

func resourceCLCServerUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clc.Client)
	id := d.Id()

	var err error
	var edits []api.Update
	var updates []api.Update
	var i int

	poll := make(chan *status.Response, 1)
	d.Partial(true)
	s, err := client.Server.Get(id)
	if err != nil {
		return fmt.Errorf("Failed fetching server: %v - %v", d.Id(), err)
	}
	// edits happen synchronously
	if delta, orig := d.Get("description").(string), s.Description; delta != orig {
		d.SetPartial("description")
		edits = append(edits, server.UpdateDescription(delta))
	}
	if delta, orig := d.Get("group_id").(string), s.GroupID; delta != orig {
		d.SetPartial("group_id")
		edits = append(edits, server.UpdateGroup(delta))
	}
	if len(edits) > 0 {
		err = client.Server.Edit(id, edits...)
		if err != nil {
			return fmt.Errorf("Failed saving edits: %v", err)
		}
	}
	// updates are queue processed
	if d.HasChange("password") {
		d.SetPartial("password")
		o, _ := d.GetChange("password")
		old := o.(string)
		pass := d.Get("password").(string)
		updates = append(updates, server.UpdateCredentials(old, pass))
	}
	if i = d.Get("cpu").(int); i != s.Details.CPU {
		d.SetPartial("cpu")
		updates = append(updates, server.UpdateCPU(i))
	}
	if i = d.Get("memory_mb").(int); i != s.Details.MemoryMB {
		d.SetPartial("memory_mb")
		updates = append(updates, server.UpdateMemory(i/1024)) // takes GB
	}

	if d.HasChange("custom_fields") {
		d.SetPartial("custom_fields")
		fields, err := parseCustomFields(d)
		if err != nil {
			return fmt.Errorf("Failed setting customfields: %v", err)
		}
		updates = append(updates, server.UpdateCustomfields(fields))
	}
	if d.HasChange("additional_disks") {
		d.SetPartial("additional_disks")
		disks, err := parseAdditionalDisks(d)
		if err != nil {
			return fmt.Errorf("Failed parsing disks: %v", err)
		}
		updates = append(updates, server.UpdateAdditionaldisks(disks))
	}

	if len(updates) > 0 {
		resp, err := client.Server.Update(id, updates...)
		if err != nil {
			return fmt.Errorf("Failed saving updates: %v", err)
		}

		err = client.Status.Poll(resp.ID, poll)
		if err != nil {
			return err
		}
		status := <-poll
		if status.Failed() {
			return fmt.Errorf("Update failed")
		}
		log.Printf("[INFO] Server updated! status: %v", status.Status)
	}

	if d.HasChange("power_state") {
		st := d.Get("power_state").(string)
		log.Printf("[DEBUG] POWER: %v => %v", s.Details.Powerstate, st)
		newst := stateFromString(st)
		servers, err := client.Server.PowerState(newst, s.Name)
		if err != nil {
			return fmt.Errorf("Failed setting power state to: %v", newst)
		}
		ok, id := servers[0].GetStatusID()
		if !ok {
			return fmt.Errorf("Failed extracting power state queue status from: %v", servers[0])
		}
		err = client.Status.Poll(id, poll)
		if err != nil {
			return err
		}
		status := <-poll
		if status.Failed() {
			return fmt.Errorf("Update failed")
		}
		log.Printf("[INFO] state updated: %v", status)
	}

	d.Partial(false)
	return nil
}

func resourceCLCServerDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clc.Client)
	id := d.Id()
	resp, err := client.Server.Delete(id)
	if err != nil || !resp.IsQueued {
		return fmt.Errorf("Failed queueing delete of %v - %v", id, err)
	}

	ok, st := resp.GetStatusID()
	if !ok {
		return fmt.Errorf("Failed extracting status to poll on %v: %v", resp, err)
	}
	err = waitStatus(client, st)
	if err != nil {
		return err
	}
	log.Printf("[INFO] Server sucessfully deleted: %v", st)
	return nil
}
