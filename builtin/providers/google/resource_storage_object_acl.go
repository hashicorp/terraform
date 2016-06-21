package google

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"

	"google.golang.org/api/googleapi"
	"google.golang.org/api/storage/v1"
)

func resourceStorageObjectAcl() *schema.Resource {
	return &schema.Resource{
		Create: resourceStorageObjectAclCreate,
		Read:   resourceStorageObjectAclRead,
		Update: resourceStorageObjectAclUpdate,
		Delete: resourceStorageObjectAclDelete,

		Schema: map[string]*schema.Schema{
			"bucket": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"object": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"predefined_acl": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"role_entity": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func getObjectAclId(object string) string {
	return object + "-acl"
}

func resourceStorageObjectAclCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	bucket := d.Get("bucket").(string)
	object := d.Get("object").(string)

	predefined_acl := ""
	role_entity := make([]interface{}, 0)

	if v, ok := d.GetOk("predefined_acl"); ok {
		predefined_acl = v.(string)
	}

	if v, ok := d.GetOk("role_entity"); ok {
		role_entity = v.([]interface{})
	}

	if len(predefined_acl) > 0 {
		if len(role_entity) > 0 {
			return fmt.Errorf("Error, you cannot specify both " +
				"\"predefined_acl\" and \"role_entity\"")
		}

		res, err := config.clientStorage.Objects.Get(bucket, object).Do()

		if err != nil {
			return fmt.Errorf("Error reading object %s: %v", bucket, err)
		}

		res, err = config.clientStorage.Objects.Update(bucket, object,
			res).PredefinedAcl(predefined_acl).Do()

		if err != nil {
			return fmt.Errorf("Error updating object %s: %v", bucket, err)
		}

		return resourceStorageBucketAclRead(d, meta)
	} else if len(role_entity) > 0 {
		for _, v := range role_entity {
			pair, err := getRoleEntityPair(v.(string))

			objectAccessControl := &storage.ObjectAccessControl{
				Role:   pair.Role,
				Entity: pair.Entity,
			}

			log.Printf("[DEBUG]: setting role = %s, entity = %s", pair.Role, pair.Entity)

			_, err = config.clientStorage.ObjectAccessControls.Insert(bucket,
				object, objectAccessControl).Do()

			if err != nil {
				return fmt.Errorf("Error setting ACL for %s on object %s: %v", pair.Entity, object, err)
			}
		}

		return resourceStorageObjectAclRead(d, meta)
	}

	return fmt.Errorf("Error, you must specify either " +
		"\"predefined_acl\" or \"role_entity\"")
}

func resourceStorageObjectAclRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	bucket := d.Get("bucket").(string)
	object := d.Get("object").(string)

	// Predefined ACLs cannot easily be parsed once they have been processed
	// by the GCP server
	if _, ok := d.GetOk("predefined_acl"); !ok {
		role_entity := make([]interface{}, 0)
		re_local := d.Get("role_entity").([]interface{})
		re_local_map := make(map[string]string)
		for _, v := range re_local {
			res, err := getRoleEntityPair(v.(string))

			if err != nil {
				return fmt.Errorf(
					"Old state has malformed Role/Entity pair: %v", err)
			}

			re_local_map[res.Entity] = res.Role
		}

		res, err := config.clientStorage.ObjectAccessControls.List(bucket, object).Do()

		if err != nil {
			if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
				log.Printf("[WARN] Removing Storage Object ACL for Bucket %q because it's gone", d.Get("bucket").(string))
				// The resource doesn't exist anymore
				d.SetId("")

				return nil
			}

			return err
		}

		for _, v := range res.Items {
			role := ""
			entity := ""
			for key, val := range v.(map[string]interface{}) {
				if key == "role" {
					role = val.(string)
				} else if key == "entity" {
					entity = val.(string)
				}
			}
			if _, in := re_local_map[entity]; in {
				role_entity = append(role_entity, fmt.Sprintf("%s:%s", role, entity))
				log.Printf("[DEBUG]: saving re %s-%s", role, entity)
			}
		}

		d.Set("role_entity", role_entity)
	}

	d.SetId(getObjectAclId(object))
	return nil
}

func resourceStorageObjectAclUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	bucket := d.Get("bucket").(string)
	object := d.Get("object").(string)

	if d.HasChange("role_entity") {
		o, n := d.GetChange("role_entity")
		old_re, new_re := o.([]interface{}), n.([]interface{})

		old_re_map := make(map[string]string)
		for _, v := range old_re {
			res, err := getRoleEntityPair(v.(string))

			if err != nil {
				return fmt.Errorf(
					"Old state has malformed Role/Entity pair: %v", err)
			}

			old_re_map[res.Entity] = res.Role
		}

		for _, v := range new_re {
			pair, err := getRoleEntityPair(v.(string))

			objectAccessControl := &storage.ObjectAccessControl{
				Role:   pair.Role,
				Entity: pair.Entity,
			}

			// If the old state is missing this entity, it needs to
			// be created. Otherwise it is updated
			if _, ok := old_re_map[pair.Entity]; ok {
				_, err = config.clientStorage.ObjectAccessControls.Update(
					bucket, object, pair.Entity, objectAccessControl).Do()
			} else {
				_, err = config.clientStorage.ObjectAccessControls.Insert(
					bucket, object, objectAccessControl).Do()
			}

			// Now we only store the keys that have to be removed
			delete(old_re_map, pair.Entity)

			if err != nil {
				return fmt.Errorf("Error updating ACL for object %s: %v", bucket, err)
			}
		}

		for entity, _ := range old_re_map {
			log.Printf("[DEBUG]: removing entity %s", entity)
			err := config.clientStorage.ObjectAccessControls.Delete(bucket, object, entity).Do()

			if err != nil {
				return fmt.Errorf("Error updating ACL for object %s: %v", bucket, err)
			}
		}

		return resourceStorageObjectAclRead(d, meta)
	}

	return nil
}

func resourceStorageObjectAclDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	bucket := d.Get("bucket").(string)
	object := d.Get("object").(string)

	re_local := d.Get("role_entity").([]interface{})
	for _, v := range re_local {
		res, err := getRoleEntityPair(v.(string))
		if err != nil {
			return err
		}

		entity := res.Entity

		log.Printf("[DEBUG]: removing entity %s", entity)

		err = config.clientStorage.ObjectAccessControls.Delete(bucket, object,
			entity).Do()

		if err != nil {
			return fmt.Errorf("Error deleting entity %s ACL: %s",
				entity, err)
		}
	}

	return nil
}
