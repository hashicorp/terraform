package main

import (
	"errors"
	"time"

	cobbler "github.com/ContainerSolutions/cobblerclient"
	"github.com/hashicorp/terraform/helper/schema"
)

var (
	tts            time.Time // Time to Sync
	syncCheckpoint bool
)

func resourceCobblerSystem() *schema.Resource {
	return &schema.Resource{
		SchemaVersion: 1,
		Create:        resourceCobblerSystemCreate,
		Read:          resourceCobblerSystemRead,
		Update:        resourceCobblerSystemUpdate,
		Delete:        resourceCobblerSystemDelete,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"profile": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"hostname": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"gateway": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"nameservers": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"network": &schema.Schema{
				Type:     schema.TypeMap,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"mac": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"ip": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"static": &schema.Schema{
							Type:     schema.TypeBool,
							Required: true,
						},
						"netmask": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"dnsname": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func resourceCobblerSystemCreate(d *schema.ResourceData, meta interface{}) error {

	client := meta.(*cobbler.Client)
	ok, err := client.Login()
	if err != nil {
		return err
	}

	if !ok {
		return errors.New("Invalid Cobbler credentials")
	}

	network := d.Get("network").(map[string]interface{})
	sysConfig := cobbler.SystemConfig{
		Name:        d.Get("name").(string),
		Profile:     d.Get("profile").(string),
		Hostname:    d.Get("hostname").(string),
		Nameservers: d.Get("nameservers").(string),
		Network: cobbler.NetworkConfig{
			Mac:     network["mac"].(string),
			Ip:      network["ip"].(string),
			DNSName: network["dnsname"].(string),
		},
	}

	system, err := client.CreateSystem(sysConfig)
	if err != nil {
		return err
	}

	tts = time.Now().Add(time.Second)

	// This block runs only once.
	// check sync method's comment.
	if !syncCheckpoint {
		syncCheckpoint = true
		channel := make(chan bool)
		go sync(channel, client)
		<-channel
	}

	if !ok {
		return errors.New("Something went wrong creating the system. Please try again.")
	}

	d.SetId(system.Id)

	return resourceCobblerSystemRead(d, meta)
}

func resourceCobblerSystemRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceCobblerSystemUpdate(d *schema.ResourceData, meta interface{}) error {

	return resourceCobblerSystemRead(d, meta)
}

func resourceCobblerSystemDelete(d *schema.ResourceData, meta interface{}) error {
	var returnValue bool

	//create client
	client := meta.(*cobbler.Client)
	ok, err := client.Login()
	if err != nil {
		return err
	}

	if !ok {
		return errors.New("Invalid Cobbler credentials")
	}

	// get name of system
	name := d.Get("name").(string)

	//delete
	returnValue, err = client.DeleteSystem(name)
	if err != nil {
		return err
	}

	if !returnValue {
		return errors.New("Delete System failed.")
	}

	// tell Terraform that the resource has been deleted
	d.SetId("")
	return nil
}

// Every time a system is created in Cobbler, the sync method must be called.
// See the following links for more information
// https://cobbler.github.io/manuals/2.6.0/3/2/2_-_Sync.html
// https://fedorahosted.org/cobbler/wiki/CobblerXmlrpc#Anexamplehowtoaddanewhost
// We only want to call sync once after all systems have been created.
// Calling sync every time after we create a system causes errors.
// See also https://github.com/cobbler/cobbler/issues/1570
func sync(channel chan bool, client *cobbler.Client) {
	// Block until we reach the time to sync
	for {
		if time.Now().UnixNano() > tts.UnixNano() {
			break
		}
	}

	client.Sync()
	channel <- true
}
