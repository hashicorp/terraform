package types

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
)

// Play return a new Ansible item to play.
type Play struct {
	defaults                  *Defaults
	enabled                   bool
	entity                    interface{}
	hosts                     []string
	groups                    []string
	become                    bool
	becomeMethod              string
	becomeUser                string
	diff                      bool
	extraVars                 map[string]interface{}
	forks                     int
	inventoryFile             string
	limit                     string
	vaultID                   []string
	vaultPasswordFile         string
	verbose                   bool
	overrideInventoryFile     string
	overrideVaultID           []string
	overrideVaultPasswordFile string
}

var (
	defaultRolesPath = []string{"~/.ansible/roles", "/usr/share/ansible/roles", "/etc/ansible/roles"}
)

const (
	// default values:
	playDefaultBecomeMethod = "sudo"
	playDefaultBecomeUser   = "root"
	playDefaultForks        = 5
	// environment variable names:
	ansibleEnvVarForceColor       = "ANSIBLE_FORCE_COLOR"
	ansibleEnvVarRolesPath        = "ANSIBLE_ROLES_PATH"
	ansibleEnvVarDefaultRolesPath = "DEFAULT_ROLES_PATH"
	// attribute names:
	playAttributeEnabled           = "enabled"
	playAttributePlaybook          = "playbook"
	playAttributeModule            = "module"
	playAttributeHosts             = "hosts"
	playAttributeGroups            = "groups"
	playAttributeBecome            = "become"
	playAttributeBecomeMethod      = "become_method"
	playAttributeBecomeUser        = "become_user"
	playAttributeDiff              = "diff"
	playAttributeExtraVars         = "extra_vars"
	playAttributeForks             = "forks"
	playAttributeInventoryFile     = "inventory_file"
	playAttributeLimit             = "limit"
	playAttributeVaultID           = "vault_id"
	playAttributeVaultPasswordFile = "vault_password_file"
	playAttributeVerbose           = "verbose"
)

// NewPlaySchema returns a new play schema.
func NewPlaySchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				playAttributeEnabled: &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
					Default:  true,
				},
				playAttributePlaybook: NewPlaybookSchema(),
				playAttributeModule:   NewModuleSchema(),
				playAttributeHosts: &schema.Schema{
					Type:     schema.TypeList,
					Elem:     &schema.Schema{Type: schema.TypeString},
					Optional: true,
				},
				playAttributeGroups: &schema.Schema{
					Type:     schema.TypeList,
					Elem:     &schema.Schema{Type: schema.TypeString},
					Optional: true,
				},
				playAttributeBecome: &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
				},
				playAttributeBecomeMethod: &schema.Schema{
					Type:         schema.TypeString,
					Optional:     true,
					Default:      playDefaultBecomeMethod,
					ValidateFunc: vfBecomeMethod,
				},
				playAttributeBecomeUser: &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
					Default:  playDefaultBecomeUser,
				},
				playAttributeDiff: &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
				},
				playAttributeExtraVars: &schema.Schema{
					Type:     schema.TypeMap,
					Optional: true,
					Computed: true,
				},
				playAttributeForks: &schema.Schema{
					Type:     schema.TypeInt,
					Optional: true,
					Default:  playDefaultForks,
				},
				playAttributeInventoryFile: &schema.Schema{
					Type:         schema.TypeString,
					Optional:     true,
					ValidateFunc: vfPath,
				},
				playAttributeLimit: &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
				playAttributeVaultID: &schema.Schema{
					Type:          schema.TypeList,
					Elem:          &schema.Schema{Type: schema.TypeString},
					Optional:      true,
					ConflictsWith: []string{"plays.vault_password_file"},
				},
				playAttributeVaultPasswordFile: &schema.Schema{
					Type:          schema.TypeString,
					Optional:      true,
					ValidateFunc:  vfPath,
					ConflictsWith: []string{"plays.vault_id"},
				},
				playAttributeVerbose: &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
				},
			},
		},
	}
}

// NewPlayFromInterface reads Play configuration from Terraform schema.
func NewPlayFromInterface(i interface{}, defaults *Defaults) *Play {
	vals := mapFromTypeSetList(i.(*schema.Set).List())
	v := &Play{
		defaults:          defaults,
		enabled:           vals[playAttributeEnabled].(bool),
		become:            vals[playAttributeBecome].(bool),
		becomeMethod:      vals[playAttributeBecomeMethod].(string),
		becomeUser:        vals[playAttributeBecomeUser].(string),
		diff:              vals[playAttributeDiff].(bool),
		extraVars:         mapFromTypeMap(vals[playAttributeExtraVars]),
		forks:             vals[playAttributeForks].(int),
		inventoryFile:     vals[playAttributeInventoryFile].(string),
		limit:             vals[playAttributeLimit].(string),
		vaultID:           listOfInterfaceToListOfString(vals[playAttributeVaultID].([]interface{})),
		vaultPasswordFile: vals[playAttributeVaultPasswordFile].(string),
		verbose:           vals[playAttributeVerbose].(bool),
	}

	emptySet := "*Set(map[string]interface {}(nil))"

	if vals[playAttributePlaybook].(*schema.Set).GoString() != emptySet {
		v.entity = NewPlaybookFromInterface(vals[playAttributePlaybook])
	} else if vals[playAttributeModule].(*schema.Set).GoString() != emptySet {
		v.entity = NewModuleFromInterface(vals[playAttributeModule])
	}

	if val, ok := vals[playAttributeHosts]; ok {
		v.hosts = listOfInterfaceToListOfString(val.([]interface{}))
	}
	if val, ok := vals[playAttributeGroups]; ok {
		v.groups = listOfInterfaceToListOfString(val.([]interface{}))
	}

	return v
}

// Enabled controls the execution of a play.
// Play will be skipped if this value is false.
func (v *Play) Enabled() bool {
	return v.enabled
}

// Entity to run. A Playbook or Module.
func (v *Play) Entity() interface{} {
	return v.entity
}

// Hosts to include in the auto-generated inventory file.
func (v *Play) Hosts() []string {
	if len(v.hosts) > 0 {
		return v.hosts
	}
	if v.defaults.hostsIsSet {
		return v.defaults.hosts
	}
	return make([]string, 0)
}

// Groups to include in the auto-generated inventory file.
func (v *Play) Groups() []string {
	if len(v.groups) > 0 {
		return v.groups
	}
	if v.defaults.groupsIsSet {
		return v.defaults.groups
	}
	return make([]string, 0)
}

// Become represents Ansible --become flag.
func (v *Play) Become() bool {
	return v.become
}

// BecomeMethod represents Ansible --become-method flag.
func (v *Play) BecomeMethod() string {
	if v.becomeMethod != "" {
		return v.becomeMethod
	}
	if v.defaults.becomeMethodIsSet {
		return v.defaults.becomeMethod
	}
	return playDefaultBecomeMethod
}

// BecomeUser represents Ansible --become-user flag.
func (v *Play) BecomeUser() string {
	if v.becomeUser != "" {
		return v.becomeUser
	}
	if v.defaults.becomeUserIsSet {
		return v.defaults.becomeUser
	}
	return "" // will be obtained from connection info
}

// Diff represents Ansible --diff flag.
func (v *Play) Diff() bool {
	return v.diff
}

// ExtraVars represents Ansible --extra-vars flag.
func (v *Play) ExtraVars() map[string]interface{} {
	if len(v.extraVars) > 0 {
		return v.extraVars
	}
	if v.defaults.extraVarsIsSet {
		return v.defaults.extraVars
	}
	return make(map[string]interface{})
}

// Forks represents Ansible --forks flag.
func (v *Play) Forks() int {
	if v.forks > 0 {
		return v.forks
	}
	if v.defaults.forksIsSet {
		return v.defaults.forks
	}
	return playDefaultForks
}

// InventoryFile represents Ansible --inventory-file flag.
func (v *Play) InventoryFile() string {
	if v.overrideInventoryFile != "" {
		return v.overrideInventoryFile
	}
	if v.inventoryFile != "" {
		return v.inventoryFile
	}
	if v.defaults.inventoryFileIsSet {
		return v.defaults.inventoryFile
	}
	return ""
}

// Limit represents Ansible --limit flag.
func (v *Play) Limit() string {
	if v.limit != "" {
		return v.limit
	}
	if v.defaults.limitIsSet {
		return v.defaults.limit
	}
	return ""
}

// VaultPasswordFile represents Ansible --vault-password-file flag.
func (v *Play) VaultPasswordFile() string {
	if v.overrideVaultPasswordFile != "" {
		return v.overrideVaultPasswordFile
	}
	if v.vaultPasswordFile != "" {
		return v.vaultPasswordFile
	}
	if v.defaults.vaultPasswordFileIsSet {
		return v.defaults.vaultPasswordFile
	}
	return ""
}

// VaultID represents Ansible --vault-id flag.
func (v *Play) VaultID() []string {
	if len(v.overrideVaultID) > 0 {
		return v.overrideVaultID
	}
	if len(v.vaultID) > 0 {
		return v.vaultID
	}
	if v.defaults.vaultIDIsSet {
		return v.defaults.vaultID
	}
	return make([]string, 0)
}

// Verbose represents Ansible --verbose flag.
func (v *Play) Verbose() bool {
	return v.verbose
}

// SetOverrideInventoryFile is used by the provisioner in the following cases:
// - remote provisioner not given an inventory_file, a generated temporary file used
// - local mode always writes a temporary inventory file, such file has to be removed after provisioning
func (v *Play) SetOverrideInventoryFile(path string) {
	v.overrideInventoryFile = path
}

// SetOverrideVaultID is used by remote provisioner when vault id files are defined.
// After uploading the files to the machine, the paths are updated to the remote paths, such that Ansible
// can be given correct remote locations.
func (v *Play) SetOverrideVaultID(paths []string) {
	v.overrideVaultID = paths
}

// SetOverrideVaultPasswordPath is used by remote provisioner when a vault password file is defined.
// After uploading the file to the machine, the path is updated to the remote path, such that Ansible
// can be given the correct remote location.
func (v *Play) SetOverrideVaultPasswordPath(path string) {
	v.overrideVaultPasswordFile = path
}

func (v *Play) defaultRolePaths() []string {
	if val, ok := os.LookupEnv(ansibleEnvVarRolesPath); ok {
		return strings.Split(val, ":")
	}
	if val, ok := os.LookupEnv(ansibleEnvVarDefaultRolesPath); ok {
		return strings.Split(val, ":")
	}
	return defaultRolesPath
}

// ToCommand serializes the play to an executable Ansible command.
func (v *Play) ToCommand(ansibleArgs LocalModeAnsibleArgs) (string, error) {

	command := ""
	// entity to call:
	switch entity := v.Entity().(type) {
	case *Playbook:
		command = fmt.Sprintf("%s=true", ansibleEnvVarForceColor)

		// handling role directories:
		rolePaths := v.defaultRolePaths()
		for _, rp := range entity.RolesPath() {
			rolePaths = append(rolePaths, filepath.Clean(rp))
		}

		command = fmt.Sprintf("%s %s=%s", command, ansibleEnvVarRolesPath, strings.Join(rolePaths, ":"))
		command = fmt.Sprintf("%s ansible-playbook %s", command, entity.FilePath())

		// force handlers:
		if entity.ForceHandlers() {
			command = fmt.Sprintf("%s --force-handlers", command)
		}
		// skip tags:
		if len(entity.SkipTags()) > 0 {
			command = fmt.Sprintf("%s --skip-tags='%s'", command, strings.Join(entity.SkipTags(), ","))
		}
		// start at task:
		if entity.StartAtTask() != "" {
			command = fmt.Sprintf("%s --start-at-task='%s'", command, entity.StartAtTask())
		}
		// tags:
		if len(entity.Tags()) > 0 {
			command = fmt.Sprintf("%s --tags='%s'", command, strings.Join(entity.Tags(), ","))
		}

	case *Module:
		hostPattern := entity.HostPattern()
		if hostPattern == "" {
			hostPattern = ansibleModuleDefaultHostPattern
		}
		command = fmt.Sprintf("ansible %s --module-name='%s'", hostPattern, entity.module)

		if entity.Background() > 0 {
			command = fmt.Sprintf("%s --background=%d", command, entity.Background())
			if entity.Poll() > 0 {
				command = fmt.Sprintf("%s --poll=%d", command, entity.Poll())
			}
		}
		// module args:
		if len(entity.Args()) > 0 {
			args := make([]string, 0)
			for mak, mav := range entity.Args() {
				args = append(args, fmt.Sprintf("%s=%+v", mak, mav))
			}
			command = fmt.Sprintf("%s --args=\"%s\"", command, strings.Join(args, " "))
		}
		// one line:
		if entity.OneLine() {
			command = fmt.Sprintf("%s --one-line", command)
		}
	}

	// inventory file:
	command = fmt.Sprintf("%s --inventory-file='%s'", command, v.InventoryFile())

	// shared arguments:

	// become:
	if v.Become() {
		command = fmt.Sprintf("%s --become", command)
		command = fmt.Sprintf("%s --become-method='%s'", command, v.BecomeMethod())
		if v.BecomeUser() != "" {
			command = fmt.Sprintf("%s --become-user='%s'", command, v.BecomeUser())
		} else {
			command = fmt.Sprintf("%s --become-user='%s'", command, ansibleArgs.Username)
		}
	}
	// diff:
	if v.Diff() {
		command = fmt.Sprintf("%s --diff", command)
	}
	// extra vars:
	if len(v.ExtraVars()) > 0 {
		extraVars, err := json.Marshal(v.ExtraVars())
		if err != nil {
			return "", err
		}
		command = fmt.Sprintf("%s --extra-vars='%s'", command, string(extraVars))
	}
	// forks:
	if v.Forks() > 0 {
		command = fmt.Sprintf("%s --forks=%d", command, v.Forks())
	}
	// limit
	if v.Limit() != "" {
		command = fmt.Sprintf("%s --limit='%s'", command, v.Limit())
	}

	if len(v.VaultID()) > 0 {
		for _, vaultID := range v.VaultID() {
			command = fmt.Sprintf("%s --vault-id='%s'", command, filepath.Clean(vaultID))
		}
	} else {
		// vault password file:
		if v.VaultPasswordFile() != "" {
			command = fmt.Sprintf("%s --vault-password-file='%s'", command, v.VaultPasswordFile())
		}
	}

	// verbose:
	if v.Verbose() {
		command = fmt.Sprintf("%s --verbose", command)
	}

	return command, nil
}

// ToLocalCommand serializes the play to an executable local provisioning Ansible command.
func (v *Play) ToLocalCommand(ansibleArgs LocalModeAnsibleArgs, ansibleSSHSettings *AnsibleSSHSettings) (string, error) {
	baseCommand, err := v.ToCommand(ansibleArgs)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s %s", baseCommand, v.toCommandArguments(ansibleArgs, ansibleSSHSettings)), nil
}

func (v *Play) toCommandArguments(ansibleArgs LocalModeAnsibleArgs, ansibleSSHSettings *AnsibleSSHSettings) string {
	args := fmt.Sprintf("--user='%s'", ansibleArgs.Username)
	if ansibleArgs.PemFile != "" {
		args = fmt.Sprintf("%s --private-key='%s'", args, ansibleArgs.PemFile)
	}

	sshExtraAgrsOptions := make([]string, 0)
	sshExtraAgrsOptions = append(sshExtraAgrsOptions, fmt.Sprintf("-p %d", ansibleArgs.Port))
	sshExtraAgrsOptions = append(sshExtraAgrsOptions, fmt.Sprintf("-o UserKnownHostsFile=%s", ansibleArgs.KnownHostsFile))
	sshExtraAgrsOptions = append(sshExtraAgrsOptions, fmt.Sprintf("-o ConnectTimeout=%d", ansibleSSHSettings.ConnectTimeoutSeconds()))
	sshExtraAgrsOptions = append(sshExtraAgrsOptions, fmt.Sprintf("-o ConnectionAttempts=%d", ansibleSSHSettings.ConnectAttempts()))
	if ansibleArgs.BastionHost != "" {
		sshExtraAgrsOptions = append(
			sshExtraAgrsOptions,
			fmt.Sprintf(
				"-o ProxyCommand=\"ssh -p %d -W %%h:%%p %s@%s -o UserKnownHostsFile=%s\"",
				ansibleArgs.BastionPort,
				ansibleArgs.BastionUsername,
				ansibleArgs.BastionHost,
				ansibleArgs.KnownHostsFile))
		if ansibleArgs.BastionPemFile == "" && os.Getenv("SSH_AUTH_SOCK") != "" {
			sshExtraAgrsOptions = append(sshExtraAgrsOptions, "-o ForwardAgent=yes")
		}
	}

	args = fmt.Sprintf("%s --ssh-extra-args='%s'", args, strings.Join(sshExtraAgrsOptions, " "))

	return args
}
