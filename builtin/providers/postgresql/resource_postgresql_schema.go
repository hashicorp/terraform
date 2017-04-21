package postgresql

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/lib/pq"
	"github.com/sean-/postgresql-acl"
)

const (
	schemaNameAttr    = "name"
	schemaOwnerAttr   = "owner"
	schemaPolicyAttr  = "policy"
	schemaIfNotExists = "if_not_exists"

	schemaPolicyCreateAttr          = "create"
	schemaPolicyCreateWithGrantAttr = "create_with_grant"
	schemaPolicyRoleAttr            = "role"
	schemaPolicyUsageAttr           = "usage"
	schemaPolicyUsageWithGrantAttr  = "usage_with_grant"
)

func resourcePostgreSQLSchema() *schema.Resource {
	return &schema.Resource{
		Create: resourcePostgreSQLSchemaCreate,
		Read:   resourcePostgreSQLSchemaRead,
		Update: resourcePostgreSQLSchemaUpdate,
		Delete: resourcePostgreSQLSchemaDelete,
		Exists: resourcePostgreSQLSchemaExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			schemaNameAttr: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the schema",
			},
			schemaOwnerAttr: {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The ROLE name who owns the schema",
			},
			schemaIfNotExists: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "When true, use the existing schema if it exsts",
			},
			schemaPolicyAttr: &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						schemaPolicyCreateAttr: {
							Type:          schema.TypeBool,
							Optional:      true,
							Default:       false,
							Description:   "If true, allow the specified ROLEs to CREATE new objects within the schema(s)",
							ConflictsWith: []string{schemaPolicyAttr + "." + schemaPolicyCreateWithGrantAttr},
						},
						schemaPolicyCreateWithGrantAttr: {
							Type:          schema.TypeBool,
							Optional:      true,
							Default:       false,
							Description:   "If true, allow the specified ROLEs to CREATE new objects within the schema(s) and GRANT the same CREATE privilege to different ROLEs",
							ConflictsWith: []string{schemaPolicyAttr + "." + schemaPolicyCreateAttr},
						},
						schemaPolicyRoleAttr: {
							Type:        schema.TypeString,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Optional:    true,
							Default:     "",
							Description: "ROLE who will receive this policy (default: PUBLIC)",
						},
						schemaPolicyUsageAttr: {
							Type:          schema.TypeBool,
							Optional:      true,
							Default:       false,
							Description:   "If true, allow the specified ROLEs to use objects within the schema(s)",
							ConflictsWith: []string{schemaPolicyAttr + "." + schemaPolicyUsageWithGrantAttr},
						},
						schemaPolicyUsageWithGrantAttr: {
							Type:          schema.TypeBool,
							Optional:      true,
							Default:       false,
							Description:   "If true, allow the specified ROLEs to use objects within the schema(s) and GRANT the same USAGE privilege to different ROLEs",
							ConflictsWith: []string{schemaPolicyAttr + "." + schemaPolicyUsageAttr},
						},
					},
				},
			},
		},
	}
}

func resourcePostgreSQLSchemaCreate(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*Client)

	queries := []string{}

	schemaName := d.Get(schemaNameAttr).(string)
	{
		b := bytes.NewBufferString("CREATE SCHEMA ")
		if v := d.Get(schemaIfNotExists); v.(bool) {
			fmt.Fprint(b, "IF NOT EXISTS ")
		}
		fmt.Fprint(b, pq.QuoteIdentifier(schemaName))

		switch v, ok := d.GetOk(schemaOwnerAttr); {
		case ok:
			fmt.Fprint(b, " AUTHORIZATION ", pq.QuoteIdentifier(v.(string)))
		}
		queries = append(queries, b.String())
	}

	// ACL objects that can generate the necessary SQL
	type RoleKey string
	var schemaPolicies map[RoleKey]acl.Schema

	if policiesRaw, ok := d.GetOk(schemaPolicyAttr); ok {
		policiesList := policiesRaw.(*schema.Set).List()

		// NOTE: len(policiesList) doesn't take into account multiple
		// roles per policy.
		schemaPolicies = make(map[RoleKey]acl.Schema, len(policiesList))

		for _, policyRaw := range policiesList {
			policyMap := policyRaw.(map[string]interface{})
			rolePolicy := schemaPolicyToACL(policyMap)

			roleKey := RoleKey(strings.ToLower(rolePolicy.Role))
			if existingRolePolicy, ok := schemaPolicies[roleKey]; ok {
				schemaPolicies[roleKey] = existingRolePolicy.Merge(rolePolicy)
			} else {
				schemaPolicies[roleKey] = rolePolicy
			}
		}
	}

	for _, policy := range schemaPolicies {
		queries = append(queries, policy.Grants(schemaName)...)
	}

	c.catalogLock.Lock()
	defer c.catalogLock.Unlock()

	conn, err := c.Connect()
	if err != nil {
		return errwrap.Wrapf("Error connecting to PostgreSQL: {{err}}", err)
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
			return errwrap.Wrapf(fmt.Sprintf("Error creating schema %s: {{err}}", schemaName), err)
		}
	}

	if err := txn.Commit(); err != nil {
		return errwrap.Wrapf("Error committing schema: {{err}}", err)
	}

	d.SetId(schemaName)

	return resourcePostgreSQLSchemaReadImpl(d, meta)
}

func resourcePostgreSQLSchemaDelete(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*Client)
	c.catalogLock.Lock()
	defer c.catalogLock.Unlock()

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

	schemaName := d.Get(schemaNameAttr).(string)

	// NOTE(sean@): Deliberately not performing a cascading drop.
	query := fmt.Sprintf("DROP SCHEMA %s", pq.QuoteIdentifier(schemaName))
	_, err = txn.Query(query)
	if err != nil {
		return errwrap.Wrapf("Error deleting schema: {{err}}", err)
	}

	if err := txn.Commit(); err != nil {
		return errwrap.Wrapf("Error committing schema: {{err}}", err)
	}

	d.SetId("")

	return nil
}

func resourcePostgreSQLSchemaExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	c := meta.(*Client)
	c.catalogLock.RLock()
	defer c.catalogLock.RUnlock()

	conn, err := c.Connect()
	if err != nil {
		return false, err
	}
	defer conn.Close()

	var schemaName string
	err = conn.QueryRow("SELECT n.nspname FROM pg_catalog.pg_namespace n WHERE n.nspname=$1", d.Id()).Scan(&schemaName)
	switch {
	case err == sql.ErrNoRows:
		return false, nil
	case err != nil:
		return false, errwrap.Wrapf("Error reading schema: {{err}}", err)
	}

	return true, nil
}

func resourcePostgreSQLSchemaRead(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*Client)
	c.catalogLock.RLock()
	defer c.catalogLock.RUnlock()

	return resourcePostgreSQLSchemaReadImpl(d, meta)
}

func resourcePostgreSQLSchemaReadImpl(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*Client)
	conn, err := c.Connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	schemaId := d.Id()
	var schemaName, schemaOwner string
	var schemaACLs []string
	err = conn.QueryRow("SELECT n.nspname, pg_catalog.pg_get_userbyid(n.nspowner), COALESCE(n.nspacl, '{}'::aclitem[])::TEXT[] FROM pg_catalog.pg_namespace n WHERE n.nspname=$1", schemaId).Scan(&schemaName, &schemaOwner, pq.Array(&schemaACLs))
	switch {
	case err == sql.ErrNoRows:
		log.Printf("[WARN] PostgreSQL schema (%s) not found", schemaId)
		d.SetId("")
		return nil
	case err != nil:
		return errwrap.Wrapf("Error reading schema: {{err}}", err)
	default:
		type RoleKey string
		schemaPolicies := make(map[RoleKey]acl.Schema, len(schemaACLs))
		for _, aclStr := range schemaACLs {
			aclItem, err := acl.Parse(aclStr)
			if err != nil {
				return errwrap.Wrapf("Error parsing aclitem: {{err}}", err)
			}

			schemaACL, err := acl.NewSchema(aclItem)
			if err != nil {
				return errwrap.Wrapf("invalid perms for schema: {{err}}", err)
			}

			roleKey := RoleKey(strings.ToLower(schemaACL.Role))
			var mergedPolicy acl.Schema
			if existingRolePolicy, ok := schemaPolicies[roleKey]; ok {
				mergedPolicy = existingRolePolicy.Merge(schemaACL)
			} else {
				mergedPolicy = schemaACL
			}
			schemaPolicies[roleKey] = mergedPolicy
		}

		d.Set(schemaNameAttr, schemaName)
		d.Set(schemaOwnerAttr, schemaOwner)
		d.SetId(schemaName)
		return nil
	}
}

func resourcePostgreSQLSchemaUpdate(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*Client)
	c.catalogLock.Lock()
	defer c.catalogLock.Unlock()

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

	if err := setSchemaName(txn, d); err != nil {
		return err
	}

	if err := setSchemaOwner(txn, d); err != nil {
		return err
	}

	if err := setSchemaPolicy(txn, d); err != nil {
		return err
	}

	if err := txn.Commit(); err != nil {
		return errwrap.Wrapf("Error committing schema: {{err}}", err)
	}

	return resourcePostgreSQLSchemaReadImpl(d, meta)
}

func setSchemaName(txn *sql.Tx, d *schema.ResourceData) error {
	if !d.HasChange(schemaNameAttr) {
		return nil
	}

	oraw, nraw := d.GetChange(schemaNameAttr)
	o := oraw.(string)
	n := nraw.(string)
	if n == "" {
		return errors.New("Error setting schema name to an empty string")
	}

	query := fmt.Sprintf("ALTER SCHEMA %s RENAME TO %s", pq.QuoteIdentifier(o), pq.QuoteIdentifier(n))
	if _, err := txn.Query(query); err != nil {
		return errwrap.Wrapf("Error updating schema NAME: {{err}}", err)
	}
	d.SetId(n)

	return nil
}

func setSchemaOwner(txn *sql.Tx, d *schema.ResourceData) error {
	if !d.HasChange(schemaOwnerAttr) {
		return nil
	}

	oraw, nraw := d.GetChange(schemaOwnerAttr)
	o := oraw.(string)
	n := nraw.(string)
	if n == "" {
		return errors.New("Error setting schema owner to an empty string")
	}

	query := fmt.Sprintf("ALTER SCHEMA %s OWNER TO %s", pq.QuoteIdentifier(o), pq.QuoteIdentifier(n))
	if _, err := txn.Query(query); err != nil {
		return errwrap.Wrapf("Error updating schema OWNER: {{err}}", err)
	}

	return nil
}

func setSchemaPolicy(txn *sql.Tx, d *schema.ResourceData) error {
	if !d.HasChange(schemaPolicyAttr) {
		return nil
	}

	schemaName := d.Get(schemaNameAttr).(string)

	oraw, nraw := d.GetChange(schemaPolicyAttr)
	oldList := oraw.(*schema.Set).List()
	newList := nraw.(*schema.Set).List()
	queries := make([]string, 0, len(oldList)+len(newList))
	dropped, added, updated, _ := schemaChangedPolicies(oldList, newList)

	for _, p := range dropped {
		pMap := p.(map[string]interface{})
		rolePolicy := schemaPolicyToACL(pMap)

		// The PUBLIC role can not be DROP'ed, therefore we do not need
		// to prevent revoking against it not existing.
		if rolePolicy.Role != "" {
			var foundUser bool
			err := txn.QueryRow(`SELECT TRUE FROM pg_catalog.pg_user WHERE usename = $1`, rolePolicy.Role).Scan(&foundUser)
			switch {
			case err == sql.ErrNoRows:
				// Don't execute this role's REVOKEs because the role
				// was dropped first and therefore doesn't exist.
			case err != nil:
				return errwrap.Wrapf("Error reading schema: {{err}}", err)
			default:
				queries = append(queries, rolePolicy.Revokes(schemaName)...)
			}
		}
	}

	for _, p := range added {
		pMap := p.(map[string]interface{})
		rolePolicy := schemaPolicyToACL(pMap)
		queries = append(queries, rolePolicy.Grants(schemaName)...)
	}

	for _, p := range updated {
		policies := p.([]interface{})
		if len(policies) != 2 {
			panic("expected 2 policies, old and new")
		}

		{
			oldPolicies := policies[0].(map[string]interface{})
			rolePolicy := schemaPolicyToACL(oldPolicies)
			queries = append(queries, rolePolicy.Revokes(schemaName)...)
		}

		{
			newPolicies := policies[1].(map[string]interface{})
			rolePolicy := schemaPolicyToACL(newPolicies)
			queries = append(queries, rolePolicy.Grants(schemaName)...)
		}
	}

	for _, query := range queries {
		if _, err := txn.Query(query); err != nil {
			return errwrap.Wrapf("Error updating schema DCL: {{err}}", err)
		}
	}

	return nil
}

// schemaChangedPolicies walks old and new to create a set of queries that can
// be executed to enact each type of state change (roles that have been dropped
// from the policy, added to a policy, have updated privilges, or are
// unchanged).
func schemaChangedPolicies(old, new []interface{}) (dropped, added, update, unchanged map[string]interface{}) {
	type RoleKey string
	oldLookupMap := make(map[RoleKey]interface{}, len(old))
	for idx, _ := range old {
		v := old[idx]
		schemaPolicy := v.(map[string]interface{})
		if roleRaw, ok := schemaPolicy[schemaPolicyRoleAttr]; ok {
			role := roleRaw.(string)
			roleKey := strings.ToLower(role)
			oldLookupMap[RoleKey(roleKey)] = schemaPolicy
		}
	}

	newLookupMap := make(map[RoleKey]interface{}, len(new))
	for idx, _ := range new {
		v := new[idx]
		schemaPolicy := v.(map[string]interface{})
		if roleRaw, ok := schemaPolicy[schemaPolicyRoleAttr]; ok {
			role := roleRaw.(string)
			roleKey := strings.ToLower(role)
			newLookupMap[RoleKey(roleKey)] = schemaPolicy
		}
	}

	droppedRoles := make(map[string]interface{}, len(old))
	for kOld, vOld := range oldLookupMap {
		if _, ok := newLookupMap[kOld]; !ok {
			droppedRoles[string(kOld)] = vOld
		}
	}

	addedRoles := make(map[string]interface{}, len(new))
	for kNew, vNew := range newLookupMap {
		if _, ok := oldLookupMap[kNew]; !ok {
			addedRoles[string(kNew)] = vNew
		}
	}

	updatedRoles := make(map[string]interface{}, len(new))
	unchangedRoles := make(map[string]interface{}, len(new))
	for kOld, vOld := range oldLookupMap {
		if vNew, ok := newLookupMap[kOld]; ok {
			if reflect.DeepEqual(vOld, vNew) {
				unchangedRoles[string(kOld)] = vOld
			} else {
				updatedRoles[string(kOld)] = []interface{}{vOld, vNew}
			}
		}
	}

	return droppedRoles, addedRoles, updatedRoles, unchangedRoles
}

func schemaPolicyToHCL(s *acl.Schema) map[string]interface{} {
	return map[string]interface{}{
		schemaPolicyRoleAttr:            s.Role,
		schemaPolicyCreateAttr:          s.GetPrivilege(acl.Create),
		schemaPolicyCreateWithGrantAttr: s.GetGrantOption(acl.Create),
		schemaPolicyUsageAttr:           s.GetPrivilege(acl.Usage),
		schemaPolicyUsageWithGrantAttr:  s.GetGrantOption(acl.Usage),
	}
}

func schemaPolicyToACL(policyMap map[string]interface{}) acl.Schema {
	var rolePolicy acl.Schema

	if policyMap[schemaPolicyCreateAttr].(bool) {
		rolePolicy.Privileges |= acl.Create
	}

	if policyMap[schemaPolicyCreateWithGrantAttr].(bool) {
		rolePolicy.Privileges |= acl.Create
		rolePolicy.GrantOptions |= acl.Create
	}

	if policyMap[schemaPolicyUsageAttr].(bool) {
		rolePolicy.Privileges |= acl.Usage
	}

	if policyMap[schemaPolicyUsageWithGrantAttr].(bool) {
		rolePolicy.Privileges |= acl.Usage
		rolePolicy.GrantOptions |= acl.Usage
	}

	if roleRaw, ok := policyMap[schemaPolicyRoleAttr]; ok {
		rolePolicy.Role = roleRaw.(string)
	}

	return rolePolicy
}
