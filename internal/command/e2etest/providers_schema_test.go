// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package e2etest

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/e2e"
	"github.com/hashicorp/terraform/internal/getproviders"
)

// TestProvidersSchema is a test for `provider schemas -json` subcommand
// which effectively tests much of the schema-related logic underneath
func TestProvidersSchema(t *testing.T) {
	if !canRunGoBuild {
		// We're running in a separate-build-then-run context, so we can't
		// currently execute this test which depends on being able to build
		// new executable at runtime.
		//
		// (See the comment on canRunGoBuild's declaration for more information.)
		t.Skip("can't run without building a new provider executable")
	}
	t.Parallel()

	tf := e2e.NewBinary(t, terraformBin, "testdata/provider-plugin")

	// In order to do a decent end-to-end test for this case we will need a real
	// enough provider plugin to try to run and make sure we are able to
	// actually run it. Here will build the simple and simple6 (built with
	// protocol v6) providers.
	simple6Provider := filepath.Join(tf.WorkDir(), "terraform-provider-simple6")
	simple6ProviderExe := e2e.GoBuild("github.com/hashicorp/terraform/internal/provider-simple-v6/main", simple6Provider)

	simpleProvider := filepath.Join(tf.WorkDir(), "terraform-provider-simple")
	simpleProviderExe := e2e.GoBuild("github.com/hashicorp/terraform/internal/provider-simple/main", simpleProvider)

	// Move the provider binaries into a directory that we will point terraform
	// to using the -plugin-dir cli flag.
	platform := getproviders.CurrentPlatform.String()
	hashiDir := "cache/registry.terraform.io/hashicorp/"
	if err := os.MkdirAll(tf.Path(hashiDir, "simple6/0.0.1/", platform), os.ModePerm); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(simple6ProviderExe, tf.Path(hashiDir, "simple6/0.0.1/", platform, "terraform-provider-simple6")); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(tf.Path(hashiDir, "simple/0.0.1/", platform), os.ModePerm); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(simpleProviderExe, tf.Path(hashiDir, "simple/0.0.1/", platform, "terraform-provider-simple")); err != nil {
		t.Fatal(err)
	}

	//// INIT
	_, stderr, err := tf.Run("init", "-plugin-dir=cache")
	if err != nil {
		t.Fatalf("unexpected init error: %s\nstderr:\n%s", err, stderr)
	}

	expectedRawOutput := `{
    "format_version": "1.0",
    "provider_schemas": {
        "registry.terraform.io/hashicorp/simple": {
            "provider": {
                "version": 0,
                "block": {
                    "description": "This is terraform-provider-simple v5",
                    "description_kind": "plain"
                }
            },
            "resource_schemas": {
                "simple_resource": {
                    "version": 0,
                    "block": {
                        "attributes": {
                            "id": {
                                "type": "string",
                                "description_kind": "plain",
                                "computed": true
                            },
                            "value": {
                                "type": "string",
                                "description_kind": "plain",
                                "optional": true
                            }
                        },
                        "description_kind": "plain"
                    }
                }
            },
            "data_source_schemas": {
                "simple_resource": {
                    "version": 0,
                    "block": {
                        "attributes": {
                            "id": {
                                "type": "string",
                                "description_kind": "plain",
                                "computed": true
                            },
                            "value": {
                                "type": "string",
                                "description_kind": "plain",
                                "optional": true
                            }
                        },
                        "description_kind": "plain"
                    }
                }
            },
            "ephemeral_resource_schemas": {
                "simple_resource": {
                    "version": 0,
                    "block": {
                        "attributes": {
                            "id": {
                                "type": "string",
                                "description_kind": "plain",
                                "computed": true
                            },
                            "value": {
                                "type": "string",
                                "description_kind": "plain",
                                "optional": true
                            }
                        },
                        "description_kind": "plain"
                    }
                }
            },
            "list_resource_schemas": {
                "simple_resource": {
                    "version": 0,
                    "block": {
                        "attributes": {
                            "value": {
                                "type": "string",
                                "description_kind": "plain",
                                "optional": true
                            }
                        },
                        "description_kind": "plain"
                    }
                }
            },
            "resource_identity_schemas": {
                "simple_resource": {
                    "version": 0,
                    "attributes": {
                        "id": {
                            "type": "string",
                            "required_for_import": true
                        }
                    }
                }
            }
        },
        "registry.terraform.io/hashicorp/simple6": {
            "provider": {
                "version": 0,
                "block": {
                    "description": "This is terraform-provider-simple v6",
                    "description_kind": "plain"
                }
            },
            "resource_schemas": {
                "simple_resource": {
                    "version": 0,
                    "block": {
                        "attributes": {
                            "id": {
                                "type": "string",
                                "description_kind": "plain",
                                "computed": true
                            },
                            "value": {
                                "type": "string",
                                "description_kind": "plain",
                                "optional": true
                            }
                        },
                        "description_kind": "plain"
                    }
                }
            },
            "data_source_schemas": {
                "simple_resource": {
                    "version": 0,
                    "block": {
                        "attributes": {
                            "id": {
                                "type": "string",
                                "description_kind": "plain",
                                "computed": true
                            },
                            "value": {
                                "type": "string",
                                "description_kind": "plain",
                                "optional": true
                            }
                        },
                        "description_kind": "plain"
                    }
                }
            },
            "ephemeral_resource_schemas": {
                "simple_resource": {
                    "version": 0,
                    "block": {
                        "attributes": {
                            "id": {
                                "type": "string",
                                "description_kind": "plain",
                                "computed": true
                            },
                            "value": {
                                "type": "string",
                                "description_kind": "plain",
                                "optional": true
                            }
                        },
                        "description_kind": "plain"
                    }
                }
            },
            "list_resource_schemas": {
                "simple_resource": {
                    "version": 0,
                    "block": {
                        "attributes": {
                            "value": {
                                "type": "string",
                                "description_kind": "plain",
                                "optional": true
                            }
                        },
                        "description_kind": "plain"
                    }
                }
            },
            "functions": {
                "noop": {
                    "description": "noop takes any single argument and returns the same value",
                    "return_type": "dynamic",
                    "parameters": [
                        {
                            "name": "noop",
                            "description": "any value",
                            "is_nullable": true,
                            "type": "dynamic"
                        }
                    ]
                }
            },
            "resource_identity_schemas": {
                "simple_resource": {
                    "version": 0,
                    "attributes": {
                        "id": {
                            "type": "string",
                            "required_for_import": true
                        }
                    }
                }
            },
            "state_store_schemas" : {
                "simple6_fs": {
                    "version":0,
                    "block": {
                        "attributes": {
                            "workspace_dir": {
                                "type":"string",
                                "description":"The directory where state files will be created. When unset the value will default to terraform.tfstate.d","description_kind":"plain","optional":true}
                            },
                        "description_kind":"plain"
                    }
                },
                "simple6_inmem": {
                    "version": 0,
                    "block": {
                        "attributes": {
                            "lock_id": {
                                "type": "string",
                                "description": "initializes the state in a locked configuration",
                                "description_kind": "plain",
                                "optional": true
                            }
                        },
                        "description_kind":"plain"
                    }
                }
            }
        }
    }
}
`
	var expectedOutput bytes.Buffer
	err = json.Compact(&expectedOutput, []byte(expectedRawOutput))
	if err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := tf.Run("providers", "schema", "-json")
	if err != nil {
		t.Fatalf("unexpected error: %s\n%s", err, stderr)
	}

	var output bytes.Buffer
	err = json.Compact(&output, []byte(stdout))
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(expectedOutput.String(), output.String()); diff != "" {
		t.Fatalf("unexpected schema: %s\n", diff)
	}
}
