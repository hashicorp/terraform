package rabbitmq

import (
	"fmt"
	"log"
	"strings"

	"github.com/michaelklishin/rabbit-hole"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourcePermissions() *schema.Resource {
	return &schema.Resource{
		Create: CreatePermissions,
		Update: UpdatePermissions,
		Read:   ReadPermissions,
		Delete: DeletePermissions,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"user": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"vhost": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "/",
				ForceNew: true,
			},

			"permissions": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"configure": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"write": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"read": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func CreatePermissions(d *schema.ResourceData, meta interface{}) error {
	rmqc := meta.(*rabbithole.Client)

	user := d.Get("user").(string)
	vhost := d.Get("vhost").(string)
	permsList := d.Get("permissions").([]interface{})

	permsMap, ok := permsList[0].(map[string]interface{})
	if !ok {
		return fmt.Errorf("Unable to parse permissions")
	}

	if err := setPermissionsIn(rmqc, vhost, user, permsMap); err != nil {
		return err
	}

	id := fmt.Sprintf("%s@%s", user, vhost)
	d.SetId(id)

	return ReadPermissions(d, meta)
}

func ReadPermissions(d *schema.ResourceData, meta interface{}) error {
	rmqc := meta.(*rabbithole.Client)

	permissionId := strings.Split(d.Id(), "@")
	if len(permissionId) < 2 {
		return fmt.Errorf("Unable to determine Permission ID")
	}

	user := permissionId[0]
	vhost := permissionId[1]

	userPerms, err := rmqc.GetPermissionsIn(vhost, user)
	if err != nil {
		return checkDeleted(d, err)
	}

	log.Printf("[DEBUG] RabbitMQ: Permission retrieved for %s: %#v", d.Id(), userPerms)

	d.Set("user", userPerms.User)
	d.Set("vhost", userPerms.Vhost)

	perms := make([]map[string]interface{}, 1)
	p := make(map[string]interface{})
	p["configure"] = userPerms.Configure
	p["write"] = userPerms.Write
	p["read"] = userPerms.Read
	perms[0] = p
	d.Set("permissions", perms)

	return nil
}

func UpdatePermissions(d *schema.ResourceData, meta interface{}) error {
	rmqc := meta.(*rabbithole.Client)

	permissionId := strings.Split(d.Id(), "@")
	if len(permissionId) < 2 {
		return fmt.Errorf("Unable to determine Permission ID")
	}

	user := permissionId[0]
	vhost := permissionId[1]

	if d.HasChange("permissions") {
		_, newPerms := d.GetChange("permissions")

		newPermsList := newPerms.([]interface{})
		permsMap, ok := newPermsList[0].(map[string]interface{})
		if !ok {
			return fmt.Errorf("Unable to parse permissions")
		}

		if err := setPermissionsIn(rmqc, vhost, user, permsMap); err != nil {
			return err
		}
	}

	return ReadPermissions(d, meta)
}

func DeletePermissions(d *schema.ResourceData, meta interface{}) error {
	rmqc := meta.(*rabbithole.Client)

	permissionId := strings.Split(d.Id(), "@")
	if len(permissionId) < 2 {
		return fmt.Errorf("Unable to determine Permission ID")
	}

	user := permissionId[0]
	vhost := permissionId[1]

	log.Printf("[DEBUG] RabbitMQ: Attempting to delete permission for %s", d.Id())

	resp, err := rmqc.ClearPermissionsIn(vhost, user)
	log.Printf("[DEBUG] RabbitMQ: Permission delete response: %#v", resp)
	if err != nil {
		return err
	}

	if resp.StatusCode == 404 {
		// The permissions were already deleted
		return nil
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Error deleting RabbitMQ permission: %s", resp.Status)
	}

	return nil
}

func setPermissionsIn(rmqc *rabbithole.Client, vhost string, user string, permsMap map[string]interface{}) error {
	perms := rabbithole.Permissions{}

	if v, ok := permsMap["configure"].(string); ok {
		perms.Configure = v
	}

	if v, ok := permsMap["write"].(string); ok {
		perms.Write = v
	}

	if v, ok := permsMap["read"].(string); ok {
		perms.Read = v
	}

	log.Printf("[DEBUG] RabbitMQ: Attempting to set permissions for %s@%s: %#v", user, vhost, perms)

	resp, err := rmqc.UpdatePermissionsIn(vhost, user, perms)
	log.Printf("[DEBUG] RabbitMQ: Permission response: %#v", resp)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Error setting permissions: %s", resp.Status)
	}

	return nil
}
