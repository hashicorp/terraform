package cloudfoundry

import (
	"fmt"

	"github.com/hashicorp/terraform/builtin/providers/cf/cfapi"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDefaultAsg() *schema.Resource {

	return &schema.Resource{

		Create: resourceDefaultAsgCreate,
		Read:   resourceDefaultAsgRead,
		Update: resourceDefaultAsgUpdate,
		Delete: resourceDefaultAsgDelete,

		Schema: map[string]*schema.Schema{

			"name": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateDefaultRunningStagingName,
			},
			"asgs": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      resourceStringHash,
			},
		},
	}
}

func resourceDefaultAsgCreate(d *schema.ResourceData, meta interface{}) (err error) {

	session := meta.(*cfapi.Session)
	if session == nil {
		return fmt.Errorf("client is nil")
	}

	name := d.Get("name").(string)
	asgs := d.Get("asgs").(*schema.Set).List()

	am := session.ASGManager()
	switch name {
	case "running":
		err = am.UnbindAllFromRunning()
		if err != nil {
			return
		}
		for _, g := range asgs {
			err = am.BindToRunning(g.(string))
			if err != nil {
				return
			}
		}
	case "staging":
		err = am.UnbindAllFromStaging()
		if err != nil {
			return
		}
		for _, g := range asgs {
			err = am.BindToStaging(g.(string))
			if err != nil {
				return
			}
		}
	default:
		err = fmt.Errorf("default security group name must be one of 'running' or 'staging'")
	}
	d.SetId(name)

	return
}

func resourceDefaultAsgRead(d *schema.ResourceData, meta interface{}) (err error) {

	session := meta.(*cfapi.Session)
	if session == nil {
		return fmt.Errorf("client is nil")
	}

	var asgs []string

	am := session.ASGManager()
	switch d.Get("name").(string) {
	case "running":
		if asgs, err = am.Running(); err != nil {
			return
		}
	case "staging":
		if asgs, err = am.Staging(); err != nil {
			return
		}
	}

	tfAsgs := []interface{}{}
	for _, s := range asgs {
		tfAsgs = append(tfAsgs, s)
	}
	d.Set("asgs", schema.NewSet(resourceStringHash, tfAsgs))
	return
}

func resourceDefaultAsgUpdate(d *schema.ResourceData, meta interface{}) (err error) {

	session := meta.(*cfapi.Session)
	if session == nil {
		return fmt.Errorf("client is nil")
	}

	var asgs []string

	tfAsgs := d.Get("asgs").(*schema.Set).List()

	am := session.ASGManager()
	switch d.Get("name").(string) {
	case "running":
		if asgs, err = am.Running(); err != nil {
			return
		}
		for _, s := range tfAsgs {
			asg := s.(string)
			if !isStringInList(asgs, asg) {
				if err = am.BindToRunning(asg); err != nil {
					return
				}
			}
		}
		for _, s := range asgs {
			if !isStringInInterfaceList(tfAsgs, s) {
				if err = am.UnbindFromRunning(s); err != nil {
					return
				}
			}
		}
	case "staging":
		if asgs, err = am.Staging(); err != nil {
			return
		}
		for _, s := range tfAsgs {
			asg := s.(string)
			if !isStringInList(asgs, asg) {
				err = am.BindToStaging(asg)
				if err != nil {
					return
				}
			}
		}
		for _, s := range asgs {
			if !isStringInInterfaceList(tfAsgs, s) {
				if err = am.UnbindFromStaging(s); err != nil {
					return
				}
			}
		}
	}
	return
}

func resourceDefaultAsgDelete(d *schema.ResourceData, meta interface{}) (err error) {

	session := meta.(*cfapi.Session)
	if session == nil {
		return fmt.Errorf("client is nil")
	}

	am := session.ASGManager()
	switch d.Get("name").(string) {
	case "running":
		err = am.UnbindAllFromRunning()
		if err != nil {
			return
		}
	case "staging":
		err = am.UnbindAllFromStaging()
		if err != nil {
			return
		}
	}
	return
}
