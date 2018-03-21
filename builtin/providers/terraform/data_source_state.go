package terraform

import (
        "fmt"
        "log"
        "time"

        "github.com/hashicorp/terraform/backend"
        backendinit "github.com/hashicorp/terraform/backend/init"
        "github.com/hashicorp/terraform/config/hcl2shim"
        "github.com/hashicorp/terraform/helper/schema"
        "github.com/hashicorp/terraform/tfdiags"
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

                        "config": {
                                Type:     schema.TypeMap,
                                Optional: true,
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

        // Don't break people using the old _local syntax - but note warning above
        if backendType == "_local" {
			log.Println(`[INFO] Switching old (unsupported) backend "_local" to "local"`)
			backendType = "local"
	}

	// Create the client to access our remote state
	log.Printf("[DEBUG] Initializing remote state backend: %s", backendType)
	f := backendinit.Backend(backendType)
	if f == nil {
			return fmt.Errorf("Unknown backend type: %s", backendType)
	}
	b := f()

	schema := b.ConfigSchema()
	rawConfig := d.Get("config")
	configVal := hcl2shim.HCL2ValueFromConfigValue(rawConfig)

	// Try to coerce the provided value into the desired configuration type.
	configVal, err := schema.CoerceValue(configVal)
	if err != nil {
			return fmt.Errorf("invalid %s backend configuration: %s", backendType, tfdiags.FormatError(err))
	}
	validateDiags := b.ValidateConfig(configVal)
	if validateDiags.HasErrors() {
			return validateDiags.Err()
	}
	configureDiags := b.Configure(configVal)
	if configureDiags.HasErrors() {
			return configureDiags.Err()
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
