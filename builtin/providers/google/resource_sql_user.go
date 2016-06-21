package google

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"

	"google.golang.org/api/googleapi"
	"google.golang.org/api/sqladmin/v1beta4"
)

func resourceSqlUser() *schema.Resource {
	return &schema.Resource{
		Create: resourceSqlUserCreate,
		Read:   resourceSqlUserRead,
		Update: resourceSqlUserUpdate,
		Delete: resourceSqlUserDelete,

		Schema: map[string]*schema.Schema{
			"host": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"instance": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"password": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"project": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceSqlUserCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	name := d.Get("name").(string)
	instance := d.Get("instance").(string)
	password := d.Get("password").(string)
	host := d.Get("host").(string)

	user := &sqladmin.User{
		Name:     name,
		Instance: instance,
		Password: password,
		Host:     host,
	}

	op, err := config.clientSqlAdmin.Users.Insert(project, instance,
		user).Do()

	if err != nil {
		return fmt.Errorf("Error, failed to insert "+
			"user %s into instance %s: %s", name, instance, err)
	}

	err = sqladminOperationWait(config, op, "Insert User")

	if err != nil {
		return fmt.Errorf("Error, failure waiting for insertion of %s "+
			"into %s: %s", name, instance, err)
	}

	return resourceSqlUserRead(d, meta)
}

func resourceSqlUserRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	name := d.Get("name").(string)
	instance := d.Get("instance").(string)

	users, err := config.clientSqlAdmin.Users.List(project, instance).Do()

	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			log.Printf("[WARN] Removing SQL User %q because it's gone", d.Get("name").(string))
			d.SetId("")

			return nil
		}

		return fmt.Errorf("Error, failed to get user %s in instance %s: %s", name, instance, err)
	}

	found := false
	for _, user := range users.Items {
		if user.Name == name {
			found = true
			break
		}
	}

	if !found {
		log.Printf("[WARN] Removing SQL User %q because it's gone", d.Get("name").(string))
		d.SetId("")

		return nil
	}

	d.SetId(name)

	return nil
}

func resourceSqlUserUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	if d.HasChange("password") {
		project, err := getProject(d, config)
		if err != nil {
			return err
		}

		name := d.Get("name").(string)
		instance := d.Get("instance").(string)
		host := d.Get("host").(string)
		password := d.Get("password").(string)

		user := &sqladmin.User{
			Name:     name,
			Instance: instance,
			Password: password,
			Host:     host,
		}

		op, err := config.clientSqlAdmin.Users.Update(project, instance, host, name,
			user).Do()

		if err != nil {
			return fmt.Errorf("Error, failed to update"+
				"user %s into user %s: %s", name, instance, err)
		}

		err = sqladminOperationWait(config, op, "Insert User")

		if err != nil {
			return fmt.Errorf("Error, failure waiting for update of %s "+
				"in %s: %s", name, instance, err)
		}

		return resourceSqlUserRead(d, meta)
	}

	return nil
}

func resourceSqlUserDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	name := d.Get("name").(string)
	instance := d.Get("instance").(string)
	host := d.Get("host").(string)

	op, err := config.clientSqlAdmin.Users.Delete(project, instance, host, name).Do()

	if err != nil {
		return fmt.Errorf("Error, failed to delete"+
			"user %s in instance %s: %s", name,
			instance, err)
	}

	err = sqladminOperationWait(config, op, "Delete User")

	if err != nil {
		return fmt.Errorf("Error, failure waiting for deletion of %s "+
			"in %s: %s", name, instance, err)
	}

	return nil
}
