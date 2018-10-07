package types

import (
	"github.com/hashicorp/terraform/helper/schema"
)

const (
	ansiblePlaybookAttributeForceHandlers = "force_handlers"
	ansiblePlaybookAttributeSkipTags      = "skip_tags"
	ansiblePlaybookAttributeStartAtTask   = "start_at_task"
	ansiblePlaybookAttributeTags          = "tags"
	ansiblePlaybookAttributeFilePath      = "file_path"
	ansiblePlaybookAttributeRolesPath     = "roles_path"
)

// Playbook represents playbook settings.
type Playbook struct {
	forceHandlers bool
	skipTags      []string
	startAtTask   string
	tags          []string
	filePath      string
	rolesPath     []string

	// when running a remote provisioner, the path will changed to the remote path:
	overrideFilePath  string
	overrideRolesPath []string
}

// NewPlaybookSchema returns a new Ansible playbook schema.
func NewPlaybookSchema() *schema.Schema {
	return &schema.Schema{
		Type:          schema.TypeSet,
		Optional:      true,
		ConflictsWith: []string{"plays.module"},
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				// Ansible parameters:
				ansiblePlaybookAttributeForceHandlers: &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
				},
				ansiblePlaybookAttributeSkipTags: &schema.Schema{
					Type:     schema.TypeList,
					Elem:     &schema.Schema{Type: schema.TypeString},
					Optional: true,
				},
				ansiblePlaybookAttributeStartAtTask: &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
				ansiblePlaybookAttributeTags: &schema.Schema{
					Type:     schema.TypeList,
					Elem:     &schema.Schema{Type: schema.TypeString},
					Optional: true,
				},
				// operational:
				ansiblePlaybookAttributeFilePath: &schema.Schema{
					Type:         schema.TypeString,
					Required:     true,
					ValidateFunc: vfPath,
				},
				ansiblePlaybookAttributeRolesPath: &schema.Schema{
					Type:     schema.TypeList,
					Elem:     &schema.Schema{Type: schema.TypeString},
					Optional: true,
				},
			},
		},
	}
}

// NewPlaybookFromInterface reads Playbook configuration from Terraform schema.
func NewPlaybookFromInterface(i interface{}) *Playbook {
	vals := mapFromTypeSetList(i.(*schema.Set).List())
	return &Playbook{
		filePath:      vals[ansiblePlaybookAttributeFilePath].(string),
		forceHandlers: vals[ansiblePlaybookAttributeForceHandlers].(bool),
		skipTags:      listOfInterfaceToListOfString(vals[ansiblePlaybookAttributeSkipTags].([]interface{})),
		startAtTask:   vals[ansiblePlaybookAttributeStartAtTask].(string),
		tags:          listOfInterfaceToListOfString(vals[ansiblePlaybookAttributeTags].([]interface{})),
		rolesPath:     listOfInterfaceToListOfString(vals[ansiblePlaybookAttributeRolesPath].([]interface{})),
	}
}

// FilePath represents a path to the Ansible playbook to be executed.
func (v *Playbook) FilePath() string {
	if v.overrideFilePath == "" {
		return v.filePath
	}
	return v.overrideFilePath
}

// ForceHandlers represents Ansible Playbook --force-handlers flag.
func (v *Playbook) ForceHandlers() bool {
	return v.forceHandlers
}

// SkipTags represents Ansible Playbook --skip-tags flag.
func (v *Playbook) SkipTags() []string {
	return v.skipTags
}

// StartAtTask represents Ansible Playbook --start-at-task flag.
func (v *Playbook) StartAtTask() string {
	return v.startAtTask
}

// Tags represents Ansible Playbook --tags flag.
func (v *Playbook) Tags() []string {
	return v.tags
}

// RolesPath appends role directories to ANSIBLE_ROLES_PATH environment variable,
// as documented in https://docs.ansible.com/ansible/2.5/reference_appendices/config.html#envvar-ANSIBLE_ROLES_PATH
func (v *Playbook) RolesPath() []string {
	if len(v.overrideRolesPath) == 0 {
		return v.rolesPath
	}
	return v.overrideRolesPath
}

// SetOverrideFilePath is used by the remote provisioner to reference the correct
// playbook location after the upload to the provisioned machine.
func (v *Playbook) SetOverrideFilePath(path string) {
	v.overrideFilePath = path
}

// SetOverrideRolesPath is used by the remote provisioner to reference the correct
// role locations after the upload to the provisioned machine.
func (v *Playbook) SetOverrideRolesPath(path []string) {
	v.overrideRolesPath = path
}
