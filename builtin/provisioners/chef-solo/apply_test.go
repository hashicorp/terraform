package chefsolo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func TestApply_UploadCookbooks(t *testing.T) {
	// setup
	happyCases := map[string]provisioner{
		"unix": {
			GuestOSType:      "unix",
			StagingDirectory: osDefaults["unix"].StagingDirectory,
			CookbookPaths:    []string{"chef/cookbooks/", "chef/bookcooks/"},
			PreventSudo:      true,
		},
		"windows": {
			GuestOSType:      "windows",
			StagingDirectory: osDefaults["windows"].StagingDirectory,
			CookbookPaths:    []string{"chef/cookbooks/", "chef/bookcooks/"},
		},
	}

	o := new(terraform.MockUIOutput)
	comm := new(communicator.MockCommunicator)
	comm.UploadDirs = map[string]string{}
	comm.Commands = map[string]bool{}

	for k, p := range happyCases {
		// setup result
		for i, path := range p.CookbookPaths {
			remotePath := fmt.Sprintf("%s/cookbooks-%d", p.StagingDirectory, i)
			comm.UploadDirs[path] = remotePath
			comm.Commands[fmt.Sprintf(osDefaults[p.GuestOSType].createDirCommand, remotePath, remotePath)] = true
		}
		// do it
		if err := p.uploadCookbooks(o, comm); err != nil {
			t.Fatalf("Test %q failed: %v", k, err)
		}
	}
}

func TestApply_CreateAndUploadJSONAttributes(t *testing.T) {
	// setup
	jsonAttributes := map[string]interface{}{
		"run_list": []string{"x", "y"},
		"a": map[string]interface{}{
			"b": 1,
			"c": true,
		},
	}
	happyCases := map[string]provisioner{
		"unix": {
			StagingDirectory: osDefaults["unix"].StagingDirectory,
			JSON:             jsonAttributes,
		},
		"windows": {
			StagingDirectory: osDefaults["windows"].StagingDirectory,
			JSON:             jsonAttributes,
		},
	}
	sadCases := map[string]provisioner{
		"run-list-overwritten": {
			StagingDirectory: osDefaults["unix"].StagingDirectory,
			JSON:             jsonAttributes,
			RunList:          []string{"x", "z"},
		},
	}

	o := new(terraform.MockUIOutput)
	comm := new(communicator.MockCommunicator)

	// setup result
	marshalled := `{"a":{"b":1,"c":true},"run_list":["x","y"]}`
	var expected bytes.Buffer
	json.Indent(&expected, []byte(marshalled), "", "  ")

	// continue setup result by case and then do it
	for k, p := range happyCases {
		comm.Uploads = map[string]string{
			fmt.Sprintf("%s/%s", p.StagingDirectory, "attributes.json"): expected.String(),
		}
		if err := p.createAndUploadJSONAttributes(o, comm); err != nil {
			t.Fatalf("Test %q failed: %v", k, err)
		}
	}
	for k, p := range sadCases {
		comm.Uploads = map[string]string{
			fmt.Sprintf("%s/%s", p.StagingDirectory, "attributes.json"): expected.String(),
		}
		if err := p.createAndUploadJSONAttributes(o, comm); err == nil {
			t.Fatalf("Test %q failed. Expected failure but got nil.", k)
		}
	}
}

func TestApply_CreateAndUploadSoloRb_Happy(t *testing.T) {
	// setup
	happyCases := map[string]struct {
		config   map[string]interface{}
		expected string
	}{
		"unix empty config defaults": {
			config: map[string]interface{}{
				"staging_directory": osDefaults["unix"].StagingDirectory,
			},
			expected: `log_location              "/tmp/terraform-chef-solo/chef.log"`,
		},
		"unix role_path": {
			config: map[string]interface{}{
				"keep_log":          false,
				"roles_path":        "a/b/c",
				"staging_directory": osDefaults["unix"].StagingDirectory,
			},
			expected: fmt.Sprintf(
				`role_path                 "%s/a/b/c"`,
				osDefaults["unix"].StagingDirectory,
			),
		},
		"windows cookbook_path": {
			config: map[string]interface{}{
				"cookbook_paths":    []string{"a/b/", "c/d/"},
				"keep_log":          false,
				"staging_directory": osDefaults["windows"].StagingDirectory,
			},
			expected: fmt.Sprintf(
				`cookbook_path             ["%[1]s/cookbooks-0","%[1]s/cookbooks-1"]`,
				osDefaults["windows"].StagingDirectory,
			),
		},
		"unix json_attribs": {
			config: map[string]interface{}{
				"json":              `{ "a": "b" }`,
				"keep_log":          false,
				"staging_directory": osDefaults["unix"].StagingDirectory,
			},
			expected: fmt.Sprintf(
				`json_attribs              "%s/attributes.json"`,
				osDefaults["unix"].StagingDirectory,
			),
		},
		"unix several": {
			config: map[string]interface{}{
				"cookbook_paths":    []string{"a/b/", "c/d/"},
				"environments_path": "x/y/z",
				"json":              `{ "a": "b" }`,
				"keep_log":          false,
				"roles_path":        "a/b/c",
				"staging_directory": osDefaults["unix"].StagingDirectory,
			},
			expected: fmt.Sprintf(
				`cookbook_path             ["%[1]s/cookbooks-0","%[1]s/cookbooks-1"]`+"\n"+
					`environment_path          "%[1]s/x/y/z"`+"\n"+
					`json_attribs              "%[1]s/attributes.json"`+"\n"+
					`role_path                 "%[1]s/a/b/c"`+"\n",
				osDefaults["unix"].StagingDirectory,
			),
		},
	}

	o := new(terraform.MockUIOutput)
	comm := new(communicator.MockCommunicator)

	// continue setup result by case and then do it
	for k, tc := range happyCases {
		comm.Uploads = map[string]string{
			fmt.Sprintf("%s/%s", tc.config["staging_directory"], "solo.rb"): tc.expected,
		}
		p, _ := decodeConfig(
			schema.TestResourceDataRaw(t, Provisioner().(*schema.Provisioner).Schema, tc.config),
		)
		if err := p.createAndUploadSoloRb(o, comm); err != nil {
			t.Fatalf("Test %q failed: %v", k, err)
		}
	}
}

func TestApply_CreateAndUploadSoloRb_Unhappy(t *testing.T) {
	// setup
	unhappyCases := map[string]struct {
		config   map[string]interface{}
		expected string
	}{
		"forgot staging directory": {
			config: map[string]interface{}{
				"cookbook_paths": []string{"doesn't matter"},
			},
			expected: "doesn't matter either",
		},
	}

	o := new(terraform.MockUIOutput)
	comm := new(communicator.MockCommunicator)

	// continue setup result by case and then do it
	for k, tc := range unhappyCases {
		comm.Uploads = map[string]string{
			fmt.Sprintf("%s/%s", tc.config["staging_directory"], "solo.rb"): tc.expected,
		}
		p, _ := decodeConfig(
			schema.TestResourceDataRaw(t, Provisioner().(*schema.Provisioner).Schema, tc.config),
		)
		if err := p.createAndUploadSoloRb(o, comm); err == nil {
			t.Fatalf("Test %q failed. Expected failure but got nil.", k)
		}
	}
}
