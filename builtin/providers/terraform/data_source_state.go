package terraform

import (
	"fmt"
	"log"
	"time"

	multierror "github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/backend"
	backendInit "github.com/hashicorp/terraform/backend/init"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func dataSourceRemoteState() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceRemoteStateRead,

		Schema: map[string]*schema.Schema{
			"backend": {
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					if vStr, ok := v.(string); ok && vStr == "_local" {
						ws = append(ws, "Use of the %q backend is now officially "+
							"supported as %q. Please update your configuration to ensure "+
							"compatibility with future versions of Terraform.",
							"_local", "local")
					}

					return
				},
			},

			// This field now contains all possible attributes that are supported
			// by any of the existing backends. When merging this into 0.12 this
			// should be reverted and instead the new 'cty.DynamicPseudoType' type
			// should be used to make this work with any future backends as well.
			"config": {
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"path": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"hostname": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"organization": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"token": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"workspaces": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem:     &schema.Schema{Type: schema.TypeMap},
						},
						"username": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"password": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"url": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"repo": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"subpath": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"storage_account_name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"container_name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"key": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"access_key": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"environment": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"resource_group_name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"arm_subscription_id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"arm_client_id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"arm_client_secret": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"arm_tenant_id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"access_token": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"address": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"scheme": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"datacenter": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"http_auth": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"gzip": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"lock": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"ca_file": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"cert_file": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"key_file": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"endpoints": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"prefix": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"cacert_path": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"cert_path": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"key_path": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"bucket": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"credentials": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"project": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"region": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"encryption_key": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"update_method": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"lock_address": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"lock_method": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"unlock_address": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"unlock_method": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"skip_cert_verification": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"account": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"user": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"key_material": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"key_id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"insecure_skip_tls_verify": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"object_name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"endpoint": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"encrypt": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"acl": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"secret_key": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"kms_key_id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"lock_table": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"dynamodb_table": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"profile": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"shared_credentials_file": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"role_arn": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"assume_role_policy": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"external_id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"session_name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"workspace_key_prefix": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"skip_credentials_validation": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"skip_get_ec2_platforms": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"skip_region_validation": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"skip_requesting_account_id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"skip_metadata_api_check": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"auth_url": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"container": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"user_name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"user_id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"region_name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"tenant_id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"tenant_name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"domain_id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"domain_name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"insecure": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"cacert_file": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"cert": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"archive_container": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"archive_path": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"expire_after": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},

			"defaults": {
				Type:     schema.TypeMap,
				Optional: true,
			},

			"environment": {
				Type:       schema.TypeString,
				Optional:   true,
				Default:    backend.DefaultStateName,
				Deprecated: "Terraform environments are now called workspaces. Please use the workspace key instead.",
			},

			"workspace": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  backend.DefaultStateName,
			},

			"__has_dynamic_attributes": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func dataSourceRemoteStateRead(d *schema.ResourceData, meta interface{}) error {
	backendType := d.Get("backend").(string)

	// Get the configuration in a type we want. This is a bit of a hack but makes
	// things work for the 'remote' backend as well. This can simply be deleted or
	// reverted when merging this 0.12.
	raw := make(map[string]interface{})
	if cfg, ok := d.GetOk("config"); ok {
		if raw, ok = cfg.(*schema.Set).List()[0].(map[string]interface{}); ok {
			for k, v := range raw {
				switch v := v.(type) {
				case string:
					if v == "" {
						delete(raw, k)
					}
				case []interface{}:
					if len(v) == 0 {
						delete(raw, k)
					}
				}
			}
		}
	}

	rawConfig, err := config.NewRawConfig(raw)
	if err != nil {
		return fmt.Errorf("error initializing backend: %s", err)
	}

	// Don't break people using the old _local syntax - but note warning above
	if backendType == "_local" {
		log.Println(`[INFO] Switching old (unsupported) backend "_local" to "local"`)
		backendType = "local"
	}

	// Create the client to access our remote state
	log.Printf("[DEBUG] Initializing remote state backend: %s", backendType)
	f := backendInit.Backend(backendType)
	if f == nil {
		return fmt.Errorf("Unknown backend type: %s", backendType)
	}
	b := f()

	warns, errs := b.Validate(terraform.NewResourceConfig(rawConfig))
	for _, warning := range warns {
		log.Printf("[DEBUG] Warning validating backend config: %s", warning)
	}
	if len(errs) > 0 {
		return fmt.Errorf("error validating backend config: %s", multierror.Append(nil, errs...))
	}

	// Configure the backend
	if err := b.Configure(terraform.NewResourceConfig(rawConfig)); err != nil {
		return fmt.Errorf("error initializing backend: %s", err)
	}

	// environment is deprecated in favour of workspace.
	// If both keys are set workspace should win.
	name := d.Get("environment").(string)
	if ws, ok := d.GetOk("workspace"); ok && ws != backend.DefaultStateName {
		name = ws.(string)
	}

	state, err := b.State(name)
	if err != nil {
		return fmt.Errorf("error loading the remote state: %s", err)
	}
	if err := state.RefreshState(); err != nil {
		return err
	}
	d.SetId(time.Now().UTC().String())

	outputMap := make(map[string]interface{})

	defaults := d.Get("defaults").(map[string]interface{})
	for key, val := range defaults {
		outputMap[key] = val
	}

	remoteState := state.State()
	if remoteState.Empty() {
		log.Println("[DEBUG] empty remote state")
	} else {
		for key, val := range remoteState.RootModule().Outputs {
			if val.Value != nil {
				outputMap[key] = val.Value
			}
		}
	}

	mappedOutputs := remoteStateFlatten(outputMap)

	for key, val := range mappedOutputs {
		d.UnsafeSetFieldRaw(key, val)
	}

	return nil
}
