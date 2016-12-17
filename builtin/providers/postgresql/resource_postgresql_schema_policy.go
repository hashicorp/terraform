package postgresql

import (
	"bytes"
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/errwrap"
	uuid "github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/lib/pq"
	"github.com/sean-/pgacl"
)

const (
	schemaPolicyCreateAttr          = "create"
	schemaPolicyCreateWithGrantAttr = "create_with_grant"
	schemaPolicyRoleAttr            = "role"
	schemaPolicySchemaAttr          = "schema"
	schemaPolicyUsageAttr           = "usage"
	schemaPolicyUsageWithGrantAttr  = "usage_with_grant"
	// schemaPolicyRolesAttr        = "roles"
	// schemaPolicySchemasAttr      = "schemas"
)

func resourcePostgreSQLSchemaPolicy() *schema.Resource {
	return &schema.Resource{
		Create: resourcePostgreSQLSchemaPolicyCreate,
		Read:   resourcePostgreSQLSchemaPolicyRead,
		Update: resourcePostgreSQLSchemaPolicyUpdate,
		Delete: resourcePostgreSQLSchemaPolicyDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			schemaPolicyCreateAttr: {
				Type:          schema.TypeBool,
				Optional:      true,
				Default:       false,
				Description:   "If true, allow the specified ROLEs to CREATE new objects within the schema(s)",
				ConflictsWith: []string{schemaPolicyCreateWithGrantAttr},
			},
			schemaPolicyCreateWithGrantAttr: {
				Type:          schema.TypeBool,
				Optional:      true,
				Default:       false,
				Description:   "If true, allow the specified ROLEs to CREATE new objects within the schema(s) and GRANT the same CREATE privilege to different ROLEs",
				ConflictsWith: []string{schemaPolicyCreateAttr},
			},
			// schemaPolicyRolesAttr: {
			// 	Type:        schema.TypeList,
			// 	Elem:        &schema.Schema{Type: schema.TypeString},
			// 	Description: "List of ROLEs who will receive this policy",
			// },
			schemaPolicyRoleAttr: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "ROLE who will receive this policy",
			},
			schemaPolicySchemaAttr: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Target SCHEMA who will have this policy applied to it",
			},
			schemaPolicyUsageAttr: {
				Type:          schema.TypeBool,
				Optional:      true,
				Default:       false,
				Description:   "If true, allow the specified ROLEs to use objects within the schema(s)",
				ConflictsWith: []string{schemaPolicyUsageWithGrantAttr},
			},
			schemaPolicyUsageWithGrantAttr: {
				Type:          schema.TypeBool,
				Optional:      true,
				Default:       false,
				Description:   "If true, allow the specified ROLEs to use objects within the schema(s) and GRANT the same USAGE privilege to different ROLEs",
				ConflictsWith: []string{schemaPolicyUsageAttr},
			},
		},
	}
}

func resourcePostgreSQLSchemaPolicyCreate(d *schema.ResourceData, meta interface{}) error {
	id, err := uuid.GenerateUUID()
	if err != nil {
		return errwrap.Wrapf("Unable to generate schema policy ID: {{err}}", err)
	}
	d.SetId(id)

	acl := pgacl.Schema{}
	role := d.Get(schemaPolicyRoleAttr).(string)
	if strings.ToUpper(role) != "PUBLIC" {
		role = pq.QuoteIdentifier(role)
	}

	acl.CreateGrant = d.Get(schemaPolicyCreateWithGrantAttr).(bool)
	if acl.CreateGrant {
		acl.Create = true
	} else {
		acl.Create = d.Get(schemaPolicyCreateAttr).(bool)
	}

	acl.UsageGrant = d.Get(schemaPolicyUsageWithGrantAttr).(bool)
	if acl.UsageGrant {
		acl.Usage = true
	} else {
		acl.Usage = d.Get(schemaPolicyUsageAttr).(bool)
	}
	schema := d.Get(schemaPolicySchemaAttr).(string)

	queries := make([]string, 0, 2)
	if acl.Create {
		b := bytes.NewBufferString("GRANT CREATE ON SCHEMA ")
		fmt.Fprintf(b, "%s TO %s", pq.QuoteIdentifier(schema), role)

		if acl.CreateGrant {
			fmt.Fprint(b, " WITH GRANT OPTION")
		}
		queries = append(queries, b.String())
	}

	if acl.Usage {
		b := bytes.NewBufferString("GRANT USAGE ON SCHEMA ")
		fmt.Fprintf(b, "%s TO %s", pq.QuoteIdentifier(schema), role)
		if acl.UsageGrant {
			fmt.Fprint(b, " WITH GRANT OPTION")
		}
		queries = append(queries, b.String())
	}

	c := meta.(*Client)
	conn, err := c.Connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	txn, err := conn.Begin()
	if err != nil {
		return err
	}
	defer txn.Rollback()

	for _, query := range queries {
		_, err = txn.Query(query)
		if err != nil {
			return errwrap.Wrapf(fmt.Sprintf("Error applying policy on schema (%+q): {{err}}", query), err)
		}
	}

	if err := txn.Commit(); err != nil {
		return errwrap.Wrapf("Error committing schema policy: {{err}}", err)
	}

	return resourcePostgreSQLSchemaPolicyRead(d, meta)
}

func resourcePostgreSQLSchemaPolicyDelete(d *schema.ResourceData, meta interface{}) error {
	role := d.Get(schemaPolicyRoleAttr).(string)
	schema := d.Get(schemaPolicySchemaAttr).(string)

	acl := pgacl.Schema{}
	if strings.ToUpper(role) != "PUBLIC" {
		role = pq.QuoteIdentifier(role)
	}

	acl.CreateGrant = d.Get(schemaPolicyCreateWithGrantAttr).(bool)
	if acl.CreateGrant {
		acl.Create = true
	} else {
		acl.Create = d.Get(schemaPolicyCreateAttr).(bool)
	}

	acl.UsageGrant = d.Get(schemaPolicyUsageWithGrantAttr).(bool)
	if acl.UsageGrant {
		acl.Usage = true
	} else {
		acl.Usage = d.Get(schemaPolicyUsageAttr).(bool)
	}

	queries := make([]string, 0, 2)
	switch {
	case !acl.Create:
		b := bytes.NewBufferString("REVOKE CREATE ON SCHEMA ")
		fmt.Fprintf(b, "%s FROM %s", pq.QuoteIdentifier(schema), role)
		queries = append(queries, b.String())
	case acl.Create && !acl.CreateGrant:
		b := bytes.NewBufferString("REVOKE GRANT OPTION FOR CREATE ON SCHEMA ")
		fmt.Fprintf(b, "%s FROM %s", pq.QuoteIdentifier(schema), role)
		queries = append(queries, b.String())
	}

	switch {
	case !acl.Usage:
		b := bytes.NewBufferString("REVOKE USAGE ON SCHEMA ")
		fmt.Fprintf(b, "%s FROM %s", pq.QuoteIdentifier(schema), role)
		queries = append(queries, b.String())
	case acl.Usage && !acl.UsageGrant:
		b := bytes.NewBufferString("REVOKE GRANT OPTION FOR USAGE ON SCHEMA ")
		fmt.Fprintf(b, "%s FROM %s", pq.QuoteIdentifier(schema), role)
		queries = append(queries, b.String())
	}

	c := meta.(*Client)
	conn, err := c.Connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	txn, err := conn.Begin()
	if err != nil {
		return err
	}
	defer txn.Rollback()

	for _, query := range queries {
		_, err = txn.Query(query)
		if err != nil {
			return errwrap.Wrapf(fmt.Sprintf("Error removing policy on schema (%+q): {{err}}", query), err)
		}
	}
	txn.Commit()

	d.SetId("")

	return resourcePostgreSQLSchemaPolicyRead(d, meta)
}

func resourcePostgreSQLSchemaPolicyRead(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*Client)
	conn, err := c.Connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	oraw, _ := d.GetChange(schemaPolicySchemaAttr)
	o := oraw.(string)

	var schemaName, schemaOwner, schemaACL string
	err = conn.QueryRow("SELECT n.nspname, pg_catalog.pg_get_userbyid(n.nspowner), n.nspacl FROM pg_catalog.pg_namespace n WHERE n.nspname = $1", o).Scan(&schemaName, &schemaOwner, &schemaACL)
	switch {
	case err == sql.ErrNoRows:
		log.Printf("[WARN] PostgreSQL schema (%s) not found", o)
		d.SetId("")
		return nil
	case err != nil:
		return errwrap.Wrapf("Error reading schema ACLs: {{err}}", err)
	default:
		acl, err := pgacl.NewSchema(schemaACL)
		if err != nil {
			return err
		}
		switch {
		case acl.CreateGrant:
			d.Set(schemaPolicyCreateWithGrantAttr, true)
			d.Set(schemaPolicyCreateAttr, false)
		case acl.Create:
			d.Set(schemaPolicyCreateAttr, true)
			d.Set(schemaPolicyCreateWithGrantAttr, false)
		default:
			d.Set(schemaPolicyCreateWithGrantAttr, false)
			d.Set(schemaPolicyCreateAttr, false)
		}

		switch {
		case acl.UsageGrant:
			d.Set(schemaPolicyUsageWithGrantAttr, true)
			d.Set(schemaPolicyUsageAttr, false)
		case acl.Usage:
			d.Set(schemaPolicyUsageAttr, true)
			d.Set(schemaPolicyUsageWithGrantAttr, false)
		default:
			d.Set(schemaPolicyUsageWithGrantAttr, false)
			d.Set(schemaPolicyUsageAttr, false)
		}
		d.Set(schemaPolicySchemaAttr, acl.Role)
		d.Set(schemaPolicySchemaAttr, schemaName)
		return nil
	}
}

func resourcePostgreSQLSchemaPolicyUpdate(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*Client)
	conn, err := c.Connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	txn, err := conn.Begin()
	if err != nil {
		return err
	}
	defer txn.Rollback()

	if err := setSchemaPolicyCreate(txn, d); err != nil {
		return err
	}

	if err := setSchemaPolicyUsage(txn, d); err != nil {
		return err
	}

	txn.Commit()

	return resourcePostgreSQLSchemaRead(d, meta)
}

func setSchemaPolicyCreate(txn *sql.Tx, d *schema.ResourceData) error {
	if !d.HasChange(schemaPolicyCreateAttr) && !d.HasChange(schemaPolicyCreateWithGrantAttr) {
		return nil
	}

	oldCreateRaw, newCreateRaw := d.GetChange(schemaPolicyCreateAttr)
	oldCreate := oldCreateRaw.(bool)
	newCreate := newCreateRaw.(bool)

	oldGrantRaw, newGrantRaw := d.GetChange(schemaPolicyCreateWithGrantAttr)
	oldGrant := oldGrantRaw.(bool)
	newGrant := newGrantRaw.(bool)

	var grant, revoke, withGrant bool
	switch {
	case oldCreate == newCreate:
		// nothing changed
	case oldCreate && !newCreate:
		// Lost create privs
		revoke = true
	case !oldCreate && newCreate:
		// Gaining create privs
		grant = true
	}

	switch {
	case newGrant == oldGrant:
		// Nothing changed
	case newGrant && !oldGrant, // Getting WITH GRANT OPTION priv
		!newGrant && oldGrant: // Loosing WITH GRANT OPTION priv
		withGrant = true
	}

	role := d.Get(schemaPolicyRoleAttr).(string)
	if strings.ToUpper(role) != "PUBLIC" {
		role = pq.QuoteIdentifier(role)
	}

	schema := d.Get(schemaPolicySchemaAttr).(string)

	b := &bytes.Buffer{}
	switch {
	case grant:
		b = bytes.NewBufferString("GRANT CREATE ON SCHEMA ")
		fmt.Fprintf(b, "%s TO %s", pq.QuoteIdentifier(schema), role)
		if withGrant {
			fmt.Fprint(b, " WITH GRANT OPTION")
		}
	case revoke:
		b = bytes.NewBufferString("REVOKE")
		if withGrant {
			fmt.Fprint(b, " GRANT OPTION FOR")
		}

		fmt.Fprintf(b, " CREATE ON SCHEMA %s FROM %s", pq.QuoteIdentifier(schema), role)
	}

	query := b.String()
	if _, err := txn.Query(query); err != nil {
		return errwrap.Wrapf("Error updating schema create privileges: {{err}}", err)
	}

	return nil
}

func setSchemaPolicyUsage(txn *sql.Tx, d *schema.ResourceData) error {
	if !d.HasChange(schemaPolicyUsageAttr) && !d.HasChange(schemaPolicyUsageWithGrantAttr) {
		return nil
	}

	oldUsageRaw, newUsageRaw := d.GetChange(schemaPolicyUsageAttr)
	oldUsage := oldUsageRaw.(bool)
	newUsage := newUsageRaw.(bool)

	oldGrantRaw, newGrantRaw := d.GetChange(schemaPolicyUsageWithGrantAttr)
	oldGrant := oldGrantRaw.(bool)
	newGrant := newGrantRaw.(bool)

	var grant, revoke, withGrant bool
	switch {
	case oldUsage == newUsage:
		// nothing changed
	case oldUsage && !newUsage:
		// Lost usage privs
		revoke = true
	case !oldUsage && newUsage:
		// Gaining usage privs
		grant = true
	}

	switch {
	case newGrant == oldGrant:
		// Nothing changed
	case newGrant && !oldGrant, // Getting WITH GRANT OPTION priv
		!newGrant && oldGrant: // Loosing WITH GRANT OPTION priv
		withGrant = true
	}

	role := d.Get(schemaPolicyRoleAttr).(string)
	if strings.ToUpper(role) != "PUBLIC" {
		role = pq.QuoteIdentifier(role)
	}

	schema := d.Get(schemaPolicySchemaAttr).(string)

	b := &bytes.Buffer{}
	switch {
	case grant:
		b = bytes.NewBufferString("GRANT USAGE ON SCHEMA ")
		fmt.Fprintf(b, "%s TO %s", pq.QuoteIdentifier(schema), role)
		if withGrant {
			fmt.Fprint(b, " WITH GRANT OPTION")
		}
	case revoke:
		b = bytes.NewBufferString("REVOKE")
		if withGrant {
			fmt.Fprint(b, " GRANT OPTION FOR")
		}

		fmt.Fprintf(b, " USAGE ON SCHEMA %s FROM %s", pq.QuoteIdentifier(schema), role)
	}

	query := b.String()
	if _, err := txn.Query(query); err != nil {
		return errwrap.Wrapf("Error updating schema usage privileges: {{err}}", err)
	}

	return nil
}
