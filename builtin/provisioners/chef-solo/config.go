package chefsolo

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

type provisioner struct {
	Environment                string
	ConfigTemplate             string
	CookbookPaths              []string
	DataBagsPath               string
	EncryptedDataBagSecretPath string
	EnvironmentsPath           string
	ExecuteCommand             string
	InstallCommand             string
	GuestOSType                string
	JSON                       map[string]interface{}
	KeepLog                    bool
	PreventSudo                bool
	RemoteCookbookPaths        []string
	RolesPath                  string
	RunList                    []string
	SkipInstall                bool
	StagingDirectory           string
	Version                    string

	createDirCommand string
}

// Provisioner returns a Chef Solo provisioner
func Provisioner() terraform.ResourceProvisioner {
	return &schema.Provisioner{
		Schema: map[string]*schema.Schema{
			"environment": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"config_template": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  defaultSoloRbTemplate,
			},
			"cookbook_paths": &schema.Schema{
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
			},
			"data_bags_path": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"encrypted_data_bag_secret_path": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"environments_path": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"execute_command": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"guest_os_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"install_command": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"json": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"keep_log": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"prevent_sudo": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"remote_cookbook_paths": &schema.Schema{
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
			},
			"roles_path": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"run_list": &schema.Schema{
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
			},
			"skip_install": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"staging_directory": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"version": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
		},
		ApplyFunc:    applyFn,
		ValidateFunc: func(c *terraform.ResourceConfig) (ws []string, es []error) { return },
	}
}

// takes the data from the provisioner schema
func decodeConfig(d *schema.ResourceData) (*provisioner, error) {
	p := &provisioner{
		ConfigTemplate:             d.Get("config_template").(string),
		CookbookPaths:              getStringList(d.Get("cookbook_paths")),
		DataBagsPath:               d.Get("data_bags_path").(string),
		EncryptedDataBagSecretPath: d.Get("encrypted_data_bag_secret_path").(string),
		Environment:                d.Get("environment").(string),
		EnvironmentsPath:           d.Get("environments_path").(string),
		ExecuteCommand:             d.Get("execute_command").(string),
		GuestOSType:                d.Get("guest_os_type").(string),
		InstallCommand:             d.Get("install_command").(string),
		KeepLog:                    d.Get("keep_log").(bool),
		PreventSudo:                d.Get("prevent_sudo").(bool),
		RemoteCookbookPaths:        getStringList(d.Get("remote_cookbook_paths")),
		RunList:                    getStringList(d.Get("run_list")),
		RolesPath:                  d.Get("roles_path").(string),
		StagingDirectory:           d.Get("staging_directory").(string),
		SkipInstall:                d.Get("skip_install").(bool),
		Version:                    d.Get("version").(string),
	}
	if unparsed, ok := d.GetOk("json"); ok {
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(unparsed.(string)), &parsed); err != nil {
			return nil, fmt.Errorf("Error parsing `json`: %#v", err)
		}
		p.JSON = parsed
	}
	return p, nil
}
