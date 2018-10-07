package types

import (
	"github.com/hashicorp/terraform/helper/schema"
)

// Defaults represents default settings for each consequent play.
type Defaults struct {
	hosts             []string
	groups            []string
	becomeMethod      string
	becomeUser        string
	extraVars         map[string]interface{}
	forks             int
	inventoryFile     string
	limit             string
	vaultID           []string
	vaultPasswordFile string
	//
	hostsIsSet             bool
	groupsIsSet            bool
	becomeMethodIsSet      bool
	becomeUserIsSet        bool
	extraVarsIsSet         bool
	forksIsSet             bool
	inventoryFileIsSet     bool
	limitIsSet             bool
	vaultIDIsSet           bool
	vaultPasswordFileIsSet bool
}

const (
	// attribute names:
	defaultsAttributeHosts             = "hosts"
	defaultsAttributeGroups            = "groups"
	defaultsAttributeBecomeMethod      = "become_method"
	defaultsAttributeBecomeUser        = "become_user"
	defaultsAttributeExtraVars         = "extra_vars"
	defaultsAttributeForks             = "forks"
	defaultsAttributeInventoryFile     = "inventory_file"
	defaultsAttributeLimit             = "limit"
	defaultsAttributeVaultID           = "vault_id"
	defaultsAttributeVaultPasswordFile = "vault_password_file"
)

// NewDefaultsSchema returns a new defaults schema.
func NewDefaultsSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				defaultsAttributeHosts: &schema.Schema{
					Type:     schema.TypeList,
					Elem:     &schema.Schema{Type: schema.TypeString},
					Optional: true,
				},
				defaultsAttributeGroups: &schema.Schema{
					Type:     schema.TypeList,
					Elem:     &schema.Schema{Type: schema.TypeString},
					Optional: true,
				},
				defaultsAttributeBecomeMethod: &schema.Schema{
					Type:         schema.TypeString,
					Optional:     true,
					ValidateFunc: vfBecomeMethod,
				},
				defaultsAttributeBecomeUser: &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
				defaultsAttributeExtraVars: &schema.Schema{
					Type:     schema.TypeMap,
					Optional: true,
					Computed: true,
				},
				defaultsAttributeForks: &schema.Schema{
					Type:     schema.TypeInt,
					Optional: true,
				},
				defaultsAttributeInventoryFile: &schema.Schema{
					Type:         schema.TypeString,
					Optional:     true,
					ValidateFunc: vfPath,
				},
				defaultsAttributeLimit: &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
				defaultsAttributeVaultID: &schema.Schema{
					Type:          schema.TypeList,
					Elem:          &schema.Schema{Type: schema.TypeString},
					Optional:      true,
					ConflictsWith: []string{"defaults.vault_password_file"},
				},
				defaultsAttributeVaultPasswordFile: &schema.Schema{
					Type:          schema.TypeString,
					Optional:      true,
					ValidateFunc:  vfPath,
					ConflictsWith: []string{"defaults.vault_id"},
				},
			},
		},
	}
}

// NewDefaultsFromInterface reads Defaults configuration from Terraform schema.
func NewDefaultsFromInterface(i interface{}, ok bool) *Defaults {
	v := &Defaults{}
	if ok {
		vals := mapFromTypeSetList(i.(*schema.Set).List())
		if val, ok := vals[defaultsAttributeHosts]; ok {
			v.hosts = listOfInterfaceToListOfString(val.([]interface{}))
			v.hostsIsSet = len(v.hosts) > 0
		}
		if val, ok := vals[defaultsAttributeGroups]; ok {
			v.groups = listOfInterfaceToListOfString(val.([]interface{}))
			v.groupsIsSet = len(v.groups) > 0
		}
		if val, ok := vals[defaultsAttributeBecomeMethod]; ok {
			v.becomeMethod = val.(string)
			v.becomeMethodIsSet = v.becomeMethod != ""
		}
		if val, ok := vals[defaultsAttributeBecomeUser]; ok {
			v.becomeUser = val.(string)
			v.becomeUserIsSet = v.becomeUser != ""
		}
		if val, ok := vals[defaultsAttributeExtraVars]; ok {
			v.extraVars = mapFromTypeMap(val)
			v.extraVarsIsSet = len(v.extraVars) > 0
		}
		if val, ok := vals[defaultsAttributeForks]; ok {
			v.forks = val.(int)
			v.forksIsSet = v.forks > 0
		}
		if val, ok := vals[defaultsAttributeInventoryFile]; ok {
			v.inventoryFile = val.(string)
			v.inventoryFileIsSet = v.inventoryFile != ""
		}
		if val, ok := vals[defaultsAttributeLimit]; ok {
			v.limit = val.(string)
			v.limitIsSet = v.limit != ""
		}
		if val, ok := vals[defaultsAttributeVaultID]; ok {
			v.vaultID = listOfInterfaceToListOfString(val.([]interface{}))
			v.vaultIDIsSet = len(v.vaultID) > 0
		}
		if val, ok := vals[defaultsAttributeVaultPasswordFile]; ok {
			v.vaultPasswordFile = val.(string)
			v.vaultPasswordFileIsSet = v.vaultPasswordFile != ""
		}
	}
	return v
}

// Hosts returns default hosts.
func (v *Defaults) Hosts() []string {
	return v.hosts
}
