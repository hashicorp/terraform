package ansible

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

var vaultPasswordFile string
var alternativeVaultPasswordFile string
var playbookFile string

func TestMain(m *testing.M) {

	tempVaultPasswordFile, _ := ioutil.TempFile("", "vault-password-file")
	tempAlternativeVaultPasswordFile, _ := ioutil.TempFile("", "vault-password-file")
	tempPlaybookFile, _ := ioutil.TempFile("", "playbook-file")

	vaultPasswordFile = tempVaultPasswordFile.Name()
	alternativeVaultPasswordFile = tempAlternativeVaultPasswordFile.Name()
	playbookFile = tempPlaybookFile.Name()

	result := m.Run()

	os.Remove(vaultPasswordFile)
	os.Remove(alternativeVaultPasswordFile)
	os.Remove(playbookFile)

	os.Exit(result)
}

func TestResourceProvisioner_impl(t *testing.T) {
	var _ terraform.ResourceProvisioner = Provisioner()
}

func TestProvisioner(t *testing.T) {
	if err := Provisioner().(*schema.Provisioner).InternalValidate(); err != nil {
		t.Fatalf("error: %s", err)
	}
}

func TestBadConfig(t *testing.T) {
	// play.0.playbook with no file_path
	// play.0.module with no module
	c := testConfig(t, map[string]interface{}{
		"plays": []map[string]interface{}{
			map[string]interface{}{
				"playbook": []map[string]interface{}{
					map[string]interface{}{},
				},
			},
			map[string]interface{}{
				"module": []map[string]interface{}{
					map[string]interface{}{},
				},
			},
		},

		"remote": []map[string]interface{}{
			map[string]interface{}{
				"use_sudo":        false,
				"skip_install":    true,
				"skip_cleanup":    true,
				"install_version": "2.3.0.0",
			},
		},

		"defaults": []map[string]interface{}{
			map[string]interface{}{
				"hosts": []interface{}{},
			},
		},
	})
	warn, errs := Provisioner().Validate(c)
	if len(warn) > 0 {
		t.Fatalf("Warnings: %v", warn)
	}
	if len(errs) != 2 {
		t.Fatalf("Expected 2 errors but got: %v", errs)
	}
}

func TestGoodAndCompleteRemoteConfig(t *testing.T) {
	// warnings:
	// = plays.0.playbook.roles_path
	c := testConfig(t, map[string]interface{}{
		"plays": []map[string]interface{}{
			map[string]interface{}{
				"playbook": []map[string]interface{}{
					map[string]interface{}{
						"file_path":      playbookFile,
						"roles_path":     []interface{}{"${path.module}/path/to/a/role/directory"},
						"force_handlers": false,
						"skip_tags":      []string{"tag2"},
						"start_at_task":  "test task",
						"tags":           []string{"tag1", "tag2"},
					},
				},
			},
			map[string]interface{}{
				"module": []map[string]interface{}{
					map[string]interface{}{
						"module":       "some_module",
						"args":         map[string]interface{}{"ARG1": "value 1", "ARG2": "value 2"},
						"background":   10,
						"host_pattern": "all-tests",
						"one_line":     false,
						"poll":         15,
					},
				},
			},
		},

		"remote": []map[string]interface{}{
			map[string]interface{}{
				"use_sudo":        false,
				"skip_install":    true,
				"skip_cleanup":    true,
				"install_version": "2.3.0.0",
			},
		},

		"defaults": []map[string]interface{}{
			map[string]interface{}{
				"hosts":               []interface{}{"localhost"},
				"groups":              []interface{}{"group1", "group2"},
				"become_method":       "sudo",
				"become_user":         "test",
				"extra_vars":          map[string]interface{}{"VAR1": "value 1", "VAR2": "value 2"},
				"forks":               10,
				"limit":               "a=b",
				"vault_password_file": vaultPasswordFile,
			},
		},

		"ansible_ssh_settings": []map[string]interface{}{
			map[string]interface{}{
				"connect_timeout_seconds": 5,
				"connection_attempts":     5,
				"ssh_keyscan_timeout":     30,
			},
		},
	})

	warn, errs := Provisioner().Validate(c)
	if len(warn) != 1 {
		t.Fatalf("Expected one warning.")
	}
	if len(errs) > 0 {
		t.Fatalf("Errors: %v", errs)
	}
}

func TestGoodLocalConfigWithoutPlaybookWarnings(t *testing.T) {
	c := testConfig(t, map[string]interface{}{
		"plays": []map[string]interface{}{
			map[string]interface{}{
				"playbook": []map[string]interface{}{
					map[string]interface{}{
						"file_path": playbookFile,
					},
				},
			},
		},
	})
	warn, errs := Provisioner().Validate(c)
	if len(warn) > 0 {
		t.Fatalf("Warnings: %v", errs)
	}
	if len(errs) > 0 {
		t.Fatalf("Errors: %v", errs)
	}
}

func TestRequirePlaybookFilePath(t *testing.T) {
	c := testConfig(t, map[string]interface{}{
		"plays": []map[string]interface{}{
			map[string]interface{}{
				"playbook": []map[string]interface{}{
					map[string]interface{}{},
				},
			},
		},
	})
	warn, errs := Provisioner().Validate(c)
	if len(warn) > 0 {
		t.Fatalf("Warnings: %v", errs)
	}
	if len(errs) != 1 {
		t.Fatalf("Expected 1 error.")
	}
}

func TestRequireModuleName(t *testing.T) {
	c := testConfig(t, map[string]interface{}{
		"plays": []map[string]interface{}{
			map[string]interface{}{
				"module": []map[string]interface{}{
					map[string]interface{}{},
				},
			},
		},
	})
	warn, errs := Provisioner().Validate(c)
	if len(warn) > 0 {
		t.Fatalf("Warnings: %v", errs)
	}
	if len(errs) != 1 {
		t.Fatalf("Expected 1 error.")
	}
}

func TestConfigWithoutPlaysFails(t *testing.T) {
	// no plays gives a warning:
	c := testConfig(t, map[string]interface{}{})

	warn, errs := Provisioner().Validate(c)
	if len(warn) != 1 {
		t.Fatalf("Should have 1 warning.")
	}
	if len(errs) > 0 {
		t.Fatalf("Errors: %v", errs)
	}
}

func TestConfigWithPlaysbookAndModuleFails(t *testing.T) {
	// no plays gives a warning:
	c := testConfig(t, map[string]interface{}{
		"plays": []map[string]interface{}{
			map[string]interface{}{
				"playbook": []map[string]interface{}{
					map[string]interface{}{
						"file_path": playbookFile,
					},
				},
				"module": []map[string]interface{}{
					map[string]interface{}{
						"module": "module-name",
					},
				},
			},
		},
	})

	warn, errs := Provisioner().Validate(c)
	if len(warn) != 1 {
		t.Fatalf("Should have 1 warning.")
	}
	if len(errs) != 1 {
		t.Fatalf("Should have 1 error.")
	}
}

func TestConfigWithInvalidValueTypeFailes(t *testing.T) {
	// file_path is set to a boolean instead of a string
	c := testConfig(t, map[string]interface{}{
		"plays": []map[string]interface{}{
			map[string]interface{}{
				"playbook": []map[string]interface{}{
					map[string]interface{}{
						"file_path": true,
					},
				},
			},
		},
	})

	warn, errs := Provisioner().Validate(c)
	if len(warn) > 0 {
		t.Fatalf("Warnings: %v", warn)
	}
	if len(errs) != 1 {
		t.Fatalf("Expected one error but received: %v", errs)
	}
}

func TestConfigProvisionerParserDecoder(t *testing.T) {
	c := map[string]interface{}{
		"plays": []map[string]interface{}{
			map[string]interface{}{
				"playbook": []map[string]interface{}{
					map[string]interface{}{
						"file_path": playbookFile,
					},
				},
				"hosts": []interface{}{"host.to.play"},
			},
			map[string]interface{}{
				"module": []map[string]interface{}{
					map[string]interface{}{
						"module": "some-module",
					},
				},
			},
		},

		"remote": []map[string]interface{}{
			map[string]interface{}{
				"use_sudo":        false,
				"skip_install":    true,
				"skip_cleanup":    true,
				"install_version": "2.3.0.0",
			},
		},

		"defaults": []map[string]interface{}{
			map[string]interface{}{
				"hosts":               []interface{}{"localhost"},
				"groups":              []interface{}{"group1", "group2"},
				"become_method":       "sudo",
				"become_user":         "test",
				"extra_vars":          map[string]interface{}{"VAR1": "value 1", "VAR2": "value 2"},
				"forks":               10,
				"limit":               "a=b",
				"vault_password_file": vaultPasswordFile,
			},
		},

		"ansible_ssh_settings": []map[string]interface{}{
			map[string]interface{}{
				"connect_timeout_seconds": 5,
				"connection_attempts":     5,
				"ssh_keyscan_timeout":     30,
			},
		},
	}

	warn, errs := Provisioner().Validate(testConfig(t, c))
	if len(warn) > 0 {
		t.Fatalf("Warnings: %+v", warn)
	}
	if len(errs) > 0 {
		t.Fatalf("Errors: %+v", errs)
	}

	_, err := decodeConfig(
		schema.TestResourceDataRaw(t, Provisioner().(*schema.Provisioner).Schema, c),
	)

	if err != nil {
		t.Fatalf("Unexpected error while decoding the configuration: %+v", err)
	}
}

func testConfig(t *testing.T, c map[string]interface{}) *terraform.ResourceConfig {
	r, err := config.NewRawConfig(c)
	if err != nil {
		t.Fatalf("config error: %s", err)
	}
	return terraform.NewResourceConfig(r)
}
