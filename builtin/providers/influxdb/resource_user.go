package influxdb

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/influxdata/influxdb/client"
)

func resourceUser() *schema.Resource {
	return &schema.Resource{
		Create: createUser,
		Read:   readUser,
		Update: updateUser,
		Delete: deleteUser,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"password": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"admin": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},
			"grant": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"database": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"privilege": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func createUser(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*client.Client)

	name := d.Get("name").(string)
	password := d.Get("password").(string)

	is_admin := d.Get("admin").(bool)
	admin_privileges := ""
	if is_admin {
		admin_privileges = "WITH ALL PRIVILEGES"
	}

	queryStr := fmt.Sprintf("CREATE USER %s WITH PASSWORD '%s' %s", name, password, admin_privileges)
	query := client.Query{
		Command: queryStr,
	}

	resp, err := conn.Query(query)
	if err != nil {
		return err
	}
	if resp.Err != nil {
		return resp.Err
	}

	d.SetId(fmt.Sprintf("influxdb-user:%s", name))

	if v, ok := d.GetOk("grant"); ok {
		grants := v.([]interface{})
		for _, vv := range grants {
			grant := vv.(map[string]interface{})
			if err := grantPrivilegeOn(conn, grant["privilege"].(string), grant["database"].(string), name); err != nil {
				return err
			}
		}
	}

	return readUser(d, meta)
}

func exec(conn *client.Client, query string) error {
	resp, err := conn.Query(client.Query{
		Command: query,
	})
	if err != nil {
		return err
	}
	if resp.Err != nil {
		return resp.Err
	}
	return nil
}

func grantPrivilegeOn(conn *client.Client, privilege, database, user string) error {
	return exec(conn, fmt.Sprintf("GRANT %s ON %s TO %s", privilege, quoteIdentifier(database), user))
}

func revokePrivilegeOn(conn *client.Client, privilege, database, user string) error {
	return exec(conn, fmt.Sprintf("REVOKE %s ON %s FROM %s", privilege, quoteIdentifier(database), user))
}

func grantAllOn(conn *client.Client, user string) error {
	return exec(conn, fmt.Sprintf("GRANT ALL PRIVILEGES TO %s", user))
}

func revokeAllOn(conn *client.Client, user string) error {
	return exec(conn, fmt.Sprintf("REVOKE ALL PRIVILEGES FROM %s", user))
}

func readUser(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*client.Client)
	name := d.Get("name").(string)

	// InfluxDB doesn't have a command to check the existence of a single
	// User, so we instead must read the list of all Users and see
	// if ours is present in it.
	query := client.Query{
		Command: "SHOW USERS",
	}

	resp, err := conn.Query(query)
	if err != nil {
		return err
	}
	if resp.Err != nil {
		return resp.Err
	}

	var found = false
	for _, result := range resp.Results[0].Series[0].Values {
		if result[0] == name {
			found = true
			d.Set("admin", result[1].(bool))
			break
		}
	}

	if !found {
		// If we fell out here then we didn't find our User in the list.
		d.SetId("")

		return nil
	}

	return readGrants(d, meta)
}

func readGrants(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*client.Client)
	name := d.Get("name").(string)

	query := client.Query{
		Command: fmt.Sprintf("SHOW GRANTS FOR %s", name),
	}

	resp, err := conn.Query(query)
	if err != nil {
		return err
	}

	if resp.Err != nil {
		return resp.Err
	}

	var grants = []map[string]string{}
	for _, result := range resp.Results[0].Series[0].Values {
		if result[1].(string) != "NO PRIVILEGES" {
			var grant = map[string]string{
				"database":  result[0].(string),
				"privilege": strings.ToLower(result[1].(string)),
			}
			grants = append(grants, grant)
		}
	}
	d.Set("grant", grants)
	return nil
}

func updateUser(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*client.Client)
	name := d.Get("name").(string)

	if d.HasChange("admin") {
		if !d.Get("admin").(bool) {
			revokeAllOn(conn, name)
		} else {
			grantAllOn(conn, name)
		}
	}

	if d.HasChange("grant") {
		oldGrantV, newGrantV := d.GetChange("grant")
		oldGrant := oldGrantV.([]interface{})
		newGrant := newGrantV.([]interface{})

		for _, oGV := range oldGrant {
			oldGrant := oGV.(map[string]interface{})

			exists := false
			privilege := oldGrant["privilege"].(string)
			for _, nGV := range newGrant {
				newGrant := nGV.(map[string]interface{})

				if newGrant["database"].(string) == oldGrant["database"].(string) {
					exists = true
					privilege = newGrant["privilege"].(string)
				}
			}

			if !exists {
				revokePrivilegeOn(conn, oldGrant["privilege"].(string), oldGrant["database"].(string), name)
			} else {
				if privilege != oldGrant["privilege"].(string) {
					grantPrivilegeOn(conn, privilege, oldGrant["database"].(string), name)
				}
			}
		}

		for _, nGV := range newGrant {
			newGrant := nGV.(map[string]interface{})

			exists := false
			for _, oGV := range oldGrant {
				oldGrant := oGV.(map[string]interface{})

				exists = exists || (newGrant["database"].(string) == oldGrant["database"].(string) && newGrant["privilege"].(string) == oldGrant["privilege"].(string))
			}

			if !exists {
				grantPrivilegeOn(conn, newGrant["privilege"].(string), newGrant["database"].(string), name)
			}
		}
	}

	return readUser(d, meta)
}

func deleteUser(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*client.Client)
	name := d.Get("name").(string)

	queryStr := fmt.Sprintf("DROP USER %s", name)
	query := client.Query{
		Command: queryStr,
	}

	resp, err := conn.Query(query)
	if err != nil {
		return err
	}
	if resp.Err != nil {
		return resp.Err
	}

	d.SetId("")

	return nil
}
