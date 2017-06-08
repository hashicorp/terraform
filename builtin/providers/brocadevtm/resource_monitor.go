package brocadevtm

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/sky-uk/go-brocade-vtm"
	"github.com/sky-uk/go-brocade-vtm/api/monitor"
)

func resourceMonitor() *schema.Resource {
	return &schema.Resource{
		Create: resourceMonitorCreate,
		Read:   resourceMonitorRead,
		//Update: resourceMonitorUpdate,
		Delete: resourceMonitorDelete,

		// TODO when implementing update change ForceNew = false for all attributes apart from name.
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"delay": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},
			"timeout": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},
			"failures": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},
			"verbose": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},
			"use_ssl": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},
			"http_host_header": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"http_path": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"http_authentication": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"http_body_regex": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceMonitorCreate(d *schema.ResourceData, m interface{}) error {

	vtmClient := m.(*brocadevtm.VTMClient)
	var createMonitor monitor.Monitor
	var name string

	if v, ok := d.GetOk("name"); ok {
		name = v.(string)
	} else {
		return fmt.Errorf("Name argument required")
	}
	if v, ok := d.GetOk("delay"); ok {
		createMonitor.Properties.Basic.Delay = v.(int)
	}
	if v, ok := d.GetOk("timeout"); ok {
		createMonitor.Properties.Basic.Timeout = v.(int)
	}
	if v, ok := d.GetOk("failures"); ok {
		createMonitor.Properties.Basic.Failures = v.(int)
	}
	if v, ok := d.GetOk("verbose"); ok {
		createMonitor.Properties.Basic.Verbose = v.(*bool)
	}
	if v, ok := d.GetOk("use_ssl"); ok {
		createMonitor.Properties.Basic.UseSSL = v.(*bool)
	}
	if v, ok := d.GetOk("http_host_header"); ok {
		createMonitor.Properties.HTTP.HostHeader = v.(string)
	}
	if v, ok := d.GetOk("http_path"); ok {
		createMonitor.Properties.HTTP.URIPath = v.(string)
	}
	if v, ok := d.GetOk("http_authentication"); ok {
		createMonitor.Properties.HTTP.Authentication = v.(string)
	}
	if v, ok := d.GetOk("http_body_regex"); ok {
		createMonitor.Properties.HTTP.BodyRegex = v.(string)
	}

	createAPI := monitor.NewCreate(name, createMonitor)

	err := vtmClient.Do(createAPI)
	if err != nil {
		return fmt.Errorf("Error: %+v", err)
	}


	if createAPI.StatusCode() != 201 && createAPI.StatusCode() != 200 {

		return fmt.Errorf("Invalid HTTP response code %+v returned. Response object was %+v", createAPI.StatusCode(), createAPI.ResponseObject())
	}

	d.SetId(name)
	return resourceMonitorRead(d, m)

}

func resourceMonitorRead(d *schema.ResourceData, m interface{}) error {

	vtmClient := m.(*brocadevtm.VTMClient)
	var readName string
	var readMonitor monitor.Monitor

	if v, ok := d.GetOk("name"); ok {
		readName = v.(string)
	} else {
		return fmt.Errorf("Name argument required")
	}
	if v, ok := d.GetOk("delay"); ok {
		readMonitor.Properties.Basic.Delay = v.(int)
	}
	if v, ok := d.GetOk("timeout"); ok {
		readMonitor.Properties.Basic.Timeout = v.(int)
	}
	if v, ok := d.GetOk("failures"); ok {
		readMonitor.Properties.Basic.Failures = v.(int)
	}
	if v, ok := d.GetOk("verbose"); ok {
		readMonitor.Properties.Basic.Verbose = v.(*bool)
	}
	if v, ok := d.GetOk("use_ssl"); ok {
		readMonitor.Properties.Basic.UseSSL = v.(*bool)
	}
	if v, ok := d.GetOk("http_host_header"); ok {
		readMonitor.Properties.HTTP.HostHeader = v.(string)
	}
	if v, ok := d.GetOk("http_path"); ok {
		readMonitor.Properties.HTTP.URIPath = v.(string)
	}
	if v, ok := d.GetOk("http_authentication"); ok {
		readMonitor.Properties.HTTP.Authentication = v.(string)
	}
	if v, ok := d.GetOk("http_body_regex"); ok {
		readMonitor.Properties.HTTP.BodyRegex = v.(string)
	}

	getAllAPI := monitor.NewGetAll()
	err := vtmClient.Do(getAllAPI)
	if err != nil {
		return fmt.Errorf("Error: %+v", err)
	}
	getChildMonitor := getAllAPI.GetResponse().FilterByName(readName)
	if getChildMonitor.Name != readName {
		d.SetId("")
		return nil
	}
	getSingleMonitorAPI := monitor.NewGetSingleMonitor(getChildMonitor.Name)
	getMonitorProperties := getSingleMonitorAPI.GetResponse()
	err = vtmClient.Do(getSingleMonitorAPI)
	if err != nil {
		return fmt.Errorf("Error: %+v", err)
	}

	d.Set("name", getChildMonitor.Name)
	d.Set("delay", getMonitorProperties.Properties.Basic.Delay)
	d.Set("timeout", getMonitorProperties.Properties.Basic.Timeout)
	d.Set("failures", getMonitorProperties.Properties.Basic.Failures)
	d.Set("verbose", getMonitorProperties.Properties.Basic.Verbose)
	d.Set("use_ssl", getMonitorProperties.Properties.Basic.UseSSL)
	d.Set("http_host_header", getMonitorProperties.Properties.HTTP.HostHeader)
	d.Set("http_path", getMonitorProperties.Properties.HTTP.URIPath)
	d.Set("http_authentication", getMonitorProperties.Properties.HTTP.Authentication)
	d.Set("http_body_regex", getMonitorProperties.Properties.HTTP.BodyRegex)
	return nil
}

/*
func resourceMonitorUpdate(d *schema.ResourceData, m interface{}) error {
	return nil
}
*/

func resourceMonitorDelete(d *schema.ResourceData, m interface{}) error {

	vtmClient := m.(*brocadevtm.VTMClient)
	var readName string

	if v, ok := d.GetOk("name"); ok {
		readName = v.(string)
	} else {
		return fmt.Errorf("Name argument required")
	}

	deleteAPI := monitor.NewDelete(readName)
	err := vtmClient.Do(deleteAPI)
	if err != nil || deleteAPI.StatusCode() != 204 {
		return fmt.Errorf("Error deleting monitor %s. Return code != 204. Error: %+v", readName, err)
	}

	d.SetId("")
	return nil

}
