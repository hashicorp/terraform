package chefsolo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

var osDefaults = map[string]provisioner{
	"unix": {
		StagingDirectory: "/tmp/terraform-chef-solo",
		InstallCommand:   "sh -c 'command -v chef-solo || (curl -LO https://omnitruck.chef.io/install.sh && sh install.sh{{if .Version}} -v {{.Version}}{{end}})'",
		ExecuteCommand:   "chef-solo --no-color -c {{.ConfigPath}}",
		createDirCommand: "sh -c 'mkdir -p %q; chmod 777 %q'",
	},
	"windows": {
		StagingDirectory: "C:/Windows/Temp/terraform-chef-solo",
		InstallCommand:   "powershell.exe -Command \". { iwr -useb https://omnitruck.chef.io/install.ps1 } | iex; Install-Project{{if .Version}} -version {{.Version}}{{end}}\"",
		ExecuteCommand:   "C:/opscode/chef/bin/chef-solo.bat --no-color -c {{.ConfigPath}}",
		createDirCommand: "cmd /c if not exist %q mkdir %q",
	},
}

type soloRb struct {
	CookbookPaths              string
	Environment                string
	EnvironmentsPath           string
	DataBagsPath               string
	EncryptedDataBagSecretPath string
	JSON                       map[string]interface{}
	JSONPath                   string
	KeepLog                    bool
	LogPath                    string
	RolesPath                  string
	StagingDirectory           string
}

// defaultConfigTemplate
var defaultSoloRbTemplate = `
         {{- if (not (eq (len .CookbookPaths) 0)) -}}         cookbook_path             [{{.CookbookPaths}}]
{{ end }}{{- if (not (eq .EncryptedDataBagSecretPath "")) -}} encrypted_data_bag_secret "{{.EncryptedDataBagSecretPath}}"
{{ end }}{{- if (not (eq .Environment "")) -}}                environment               "{{.Environment}}"
{{ end }}{{- if (not (eq .EnvironmentsPath "")) -}}           environment_path          "{{.EnvironmentsPath}}"
{{ end }}{{- if (not (eq .DataBagsPath "")) -}}               data_bag_path             "{{.DataBagsPath}}"
{{ end }}{{- if (not (eq (len .JSON) 0)) -}}                  json_attribs              "{{.JSONPath}}"
{{ end }}{{- if .KeepLog -}}                                  log_location              "{{.LogPath}}"
{{ end }}{{- if (not (eq .RolesPath "")) -}}                  role_path                 "{{.RolesPath}}"
{{ end }}`

func applyFn(ctx context.Context) error {
	o := ctx.Value(schema.ProvOutputKey).(terraform.UIOutput)
	s := ctx.Value(schema.ProvRawStateKey).(*terraform.InstanceState)
	data := ctx.Value(schema.ProvConfigDataKey).(*schema.ResourceData)

	comm, err := getCommunicator(ctx, o, s)
	if err != nil {
		return fmt.Errorf("Couldn't get a new communicator: %v", err)
	}

	p, err := decodeConfig(data)
	if err != nil {
		return fmt.Errorf("Couldn't decode provisioner config: %v", err)
	}

	p.GuestOSType, err = getGuestOSType(s)
	if err != nil {
		return fmt.Errorf("Couldn't find out the OS based on the instance state: %v", err)
	}

	// Setup based on OS
	setIfEmpty(&p.StagingDirectory, osDefaults[p.GuestOSType].StagingDirectory)
	setIfEmpty(&p.InstallCommand, osDefaults[p.GuestOSType].InstallCommand)
	setIfEmpty(&p.ExecuteCommand, osDefaults[p.GuestOSType].ExecuteCommand)

	p.InstallCommand = renderTemplate(p.InstallCommand, p)
	p.ExecuteCommand = renderTemplate(p.ExecuteCommand, struct {
		ConfigPath string
	}{
		fmt.Sprintf("%s/solo.rb", p.StagingDirectory),
	})

	o.Output("Creating configuration...")
	if err := p.createAndUploadConfiguration(o, comm); err != nil {
		return fmt.Errorf("Error creating configuration: %v", err)
	}
	o.Output("Installing Chef-Solo...")
	if !p.SkipInstall {
		if err := p.runCommand(o, comm, p.InstallCommand); err != nil {
			return fmt.Errorf("Error installing Chef: %v", err)
		}
	}
	o.Output("Starting Chef-Solo...")
	if err := p.runCommand(o, comm, p.ExecuteCommand); err != nil {
		return fmt.Errorf("Error executing Chef: %v", err)
	}

	return nil
}

func (p *provisioner) createAndUploadConfiguration(o terraform.UIOutput, comm communicator.Communicator) error {
	if err := p.createDir(o, comm, p.StagingDirectory); err != nil {
		return fmt.Errorf("Error creating staging directory: %v", err)
	}
	if err := p.uploadCookbooks(o, comm); err != nil {
		return fmt.Errorf("Error uploading cookbooks: %v", err)
	}
	if err := p.uploadDir(o, comm, p.getRemotePath(p.RolesPath), p.RolesPath); err != nil {
		return fmt.Errorf("Error uploading roles: %v", err)
	}
	if err := p.uploadDir(o, comm, p.getRemotePath(p.EnvironmentsPath), p.EnvironmentsPath); err != nil {
		return fmt.Errorf("Error uploading roles: %v", err)
	}
	if err := p.uploadDir(o, comm, p.getRemotePath(p.DataBagsPath), p.DataBagsPath); err != nil {
		return fmt.Errorf("Error uploading data bags: %v", err)
	}
	if err := p.uploadFile(o, comm, p.getRemotePath(p.EncryptedDataBagSecretPath), p.EncryptedDataBagSecretPath); err != nil {
		return fmt.Errorf("Error uploading encrypted data bag secret: %v", err)
	}
	if err := p.createAndUploadJSONAttributes(o, comm); err != nil {
		return fmt.Errorf("Error creating and uploading the JSON attributes: %v", err)
	}
	if err := p.createAndUploadSoloRb(o, comm); err != nil {
		return fmt.Errorf("Error creating and uploading the solo.rb config file: %v", err)
	}
	return nil
}

// maps the local cookbook paths to remote cookbook paths
func (p *provisioner) getRemoteCookbookPaths() []string {
	remoteCookbookPaths := make([]string, 0, len(p.CookbookPaths))
	for i := range p.CookbookPaths {
		remotePath := p.getRemotePath(fmt.Sprintf("cookbooks-%d", i))
		remoteCookbookPaths = append(remoteCookbookPaths, remotePath)
	}
	return remoteCookbookPaths
}

func (p *provisioner) getRemotePath(localPath string) string {
	if localPath != "" {
		return fmt.Sprintf("%s/%s", p.StagingDirectory, localPath)
	}
	return ""
}

// uploads the cookbooks from the local cookbook paths to remote cookbook paths
func (p *provisioner) uploadCookbooks(o terraform.UIOutput, comm communicator.Communicator) error {
	for i, remotePath := range p.getRemoteCookbookPaths() {
		localPath := p.CookbookPaths[i]
		if err := p.uploadDir(o, comm, remotePath, localPath); err != nil {
			return fmt.Errorf("Error uploading cookbooks: %v", err)
		}
	}
	return nil
}

// get the node attributes, add the `run_list` if it's specified, and upload it
func (p *provisioner) createAndUploadJSONAttributes(o terraform.UIOutput, comm communicator.Communicator) error {
	o.Output("Creating Chef JSON attributes file...")

	jsonData := make(map[string]interface{})
	for k, v := range p.JSON {
		jsonData[k] = v
	}
	if len(p.RunList) > 0 {
		jsonData["run_list"] = p.RunList
	}
	jsonBytes, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		return err
	}
	remotePath := filepath.ToSlash(filepath.Join(p.StagingDirectory, "attributes.json"))
	if err := comm.Upload(remotePath, bytes.NewReader(jsonBytes)); err != nil {
		return fmt.Errorf("Error creating the Chef JSON attributes file: %v", err)
	}
	return nil
}

// creates and uploads the solo.rb config file for Chef Solo to use
func (p *provisioner) createAndUploadSoloRb(o terraform.UIOutput, comm communicator.Communicator) error {
	o.Output("Creating solo.rb config filed...")

	quotedRemotePaths := make([]string, len(p.CookbookPaths)+len(p.RemoteCookbookPaths))
	for i, remotePath := range p.getRemoteCookbookPaths() {
		quotedRemotePaths[i] = fmt.Sprintf(`"%s"`, remotePath)
	}
	for i, remotePath := range p.RemoteCookbookPaths {
		i = len(p.CookbookPaths) + i
		quotedRemotePaths[i] = fmt.Sprintf(`"%s"`, remotePath)
	}

	soloRbConfig := renderTemplate(p.ConfigTemplate, &soloRb{
		CookbookPaths:              strings.Join(quotedRemotePaths, ","),
		DataBagsPath:               p.getRemotePath(p.DataBagsPath),
		EncryptedDataBagSecretPath: p.getRemotePath(p.EncryptedDataBagSecretPath),
		Environment:                p.Environment,
		EnvironmentsPath:           p.getRemotePath(p.EnvironmentsPath),
		JSON:                       p.JSON,
		JSONPath:                   fmt.Sprintf("%s/attributes.json", p.StagingDirectory),
		KeepLog:                    p.KeepLog,
		LogPath:                    fmt.Sprintf("%s/chef.log", p.StagingDirectory),
		RolesPath:                  p.getRemotePath(p.RolesPath),
		StagingDirectory:           p.StagingDirectory,
	})

	remoteSoloRbPath := filepath.ToSlash(filepath.Join(p.StagingDirectory, "solo.rb"))
	if err := comm.Upload(remoteSoloRbPath, strings.NewReader(soloRbConfig)); err != nil {
		return fmt.Errorf("Error creating the solo.rb file: %v", err)
	}
	return nil
}
