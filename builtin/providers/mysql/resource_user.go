package mysql

import (
	"fmt"
	"log"

	mysqlc "github.com/ziutek/mymysql/mysql"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceUser() *schema.Resource {
	return &schema.Resource{
		Create: CreateUser,
		Update: UpdateUser,
		Read:   ReadUser,
		Delete: DeleteUser,

		Schema: map[string]*schema.Schema{
			"user": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"host": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "localhost",
			},

			"password": &schema.Schema{
				Type:      schema.TypeString,
				Optional:  true,
				Sensitive: true,
			},
		},
	}
}

func CreateUser(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(mysqlc.Conn)

	stmtSQL := fmt.Sprintf("CREATE USER '%s'@'%s'",
		d.Get("user").(string),
		d.Get("host").(string))

	password := d.Get("password").(string)
	if password != "" {
		stmtSQL = stmtSQL + fmt.Sprintf(" IDENTIFIED BY '%s'", password)
	}

	log.Println("Executing statement:", stmtSQL)
	_, _, err := conn.Query(stmtSQL)
	if err != nil {
		return err
	}

	user := fmt.Sprintf("%s@%s", d.Get("user").(string), d.Get("host").(string))
	d.SetId(user)

	return nil
}

func UpdateUser(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(mysqlc.Conn)

	if d.HasChange("password") {
		_, newpw := d.GetChange("password")
		stmtSQL := fmt.Sprintf("ALTER USER '%s'@'%s' IDENTIFIED BY '%s'",
			d.Get("user").(string),
			d.Get("host").(string),
			newpw.(string))

		log.Println("Executing query:", stmtSQL)
		_, _, err := conn.Query(stmtSQL)
		if err != nil {
			return err
		}
	}

	return nil
}

func ReadUser(d *schema.ResourceData, meta interface{}) error {
	// At this time, all attributes are supplied by the user
	return nil
}

func DeleteUser(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(mysqlc.Conn)

	stmtSQL := fmt.Sprintf("DROP USER '%s'@'%s'",
		d.Get("user").(string),
		d.Get("host").(string))

	log.Println("Executing statement:", stmtSQL)

	_, _, err := conn.Query(stmtSQL)
	if err == nil {
		d.SetId("")
	}
	return err
}
