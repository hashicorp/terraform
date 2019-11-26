package habitat

import (
	"testing"

	"fmt"
	"os"
	"path"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

const winInstallScriptContents = `
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
iwr https://api.bintray.com/content/habitat/stable/windows/x86_64/hab-%24latest-x86_64-windows.zip?bt_package=hab-x86_64-windows -Outfile c:\habitat.zip
Expand-Archive c:/habitat.zip c:/
mv c:/hab-* c:/habitat
$env:Path = $env:Path,"C:\habitat" -join ";"
[System.Environment]::SetEnvironmentVariable('Path', $env:Path, [System.EnvironmentVariableTarget]::Machine)
# Install hab as a Windows service
hab pkg install core/windows-service
New-NetFirewallRule -DisplayName "Habitat TCP" -Direction Inbound -Action Allow -Protocol TCP -LocalPort 9631,9638
New-NetFirewallRule -DisplayName "Habitat UDP" -Direction Inbound -Action Allow -Protocol UDP -LocalPort 9638
`
const winHabLicAcceptContents = `
[System.Environment]::SetEnvironmentVariable('HAB_LICENSE', "accept", [System.EnvironmentVariableTarget]::Machine)
[System.Environment]::SetEnvironmentVariable('HAB_LICENSE', "accept", [System.EnvironmentVariableTarget]::Process)
[System.Environment]::SetEnvironmentVariable('HAB_LICENSE', "accept", [System.EnvironmentVariableTarget]::User)
`

func TestWinProvisioner_winInstallHabitat(t *testing.T) {
	var uploadPath, scriptName string
	uploadPath = os.TempDir()
	scriptName = "win_hab_install.ps1"

	cases := map[string]struct {
		Config     map[string]interface{}
		Commands   map[string]bool
		Uploads    map[string]string
		ScriptPath string
	}{
		"Installation of version before license acceptance requirement": {
			Config: map[string]interface{}{
				"version": "0.79.1",
			},

			Commands: map[string]bool{
				fmt.Sprintf("powershell -NoProfile -ExecutionPolicy Bypass -File %s", path.Join(path.Dir(uploadPath), scriptName)): true,
			},

			Uploads: map[string]string{
				path.Join(path.Dir(uploadPath), scriptName): fmt.Sprintf("%s", winInstallScript),
			},
		},
		"Installation of version after license acceptance requirement": {
			Config: map[string]interface{}{
				"version":        "0.81.1",
				"accept_license": true,
			},

			Commands: map[string]bool{
				fmt.Sprintf("powershell -NoProfile -ExecutionPolicy Bypass -File %s", path.Join(path.Dir(uploadPath), scriptName)): true,
			},

			Uploads: map[string]string{
				path.Join(path.Dir(uploadPath), scriptName): fmt.Sprintf("%s\n%s", winHabLicAccept, winInstallScript),
			},
		},
	}

	o := new(terraform.MockUIOutput)
	c := new(communicator.MockCommunicator)

	for k, tc := range cases {
		c.Commands = tc.Commands
		c.UploadScripts = tc.Uploads
		c.RemoteScriptPath = uploadPath

		p, err := decodeConfig(
			schema.TestResourceDataRaw(t, Provisioner().(*schema.Provisioner).Schema, tc.Config),
		)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}

		err = p.winInstallHabitat(o, c)
		if err != nil {
			t.Fatalf("Test %q failed: %v", k, err)
		}
	}
}

func TestWinProvisioner_winStartHabitat(t *testing.T) {
	var uploadPath, scriptName, scriptContent, habOptions string
	uploadPath = os.TempDir()
	scriptName = "win_hab_start.ps1"
	habOptions = " --peer 111.222.333.444 --no-color"
	scriptContent += fmt.Sprintf("$svcPath = Join-Path $env:SystemDrive \"hab\\svc\\windows-service\"\n")
	scriptContent += fmt.Sprintf("[xml]$configXml = Get-Content (Join-Path $svcPath HabService.dll.config)\n")
	scriptContent += fmt.Sprintf("$configXml.configuration.appSettings.ChildNodes[\"2\"].value = \"%s\"\n", habOptions)
	scriptContent += fmt.Sprintf("$configXml.Save((Join-Path $svcPath HabService.dll.config))\n")
	scriptContent += fmt.Sprintf("Start-Service Habitat\n")

	cases := map[string]struct {
		Config   map[string]interface{}
		Commands map[string]bool
		Uploads  map[string]string
	}{
		"Start Habitat with correct options": {
			Config: map[string]interface{}{
				"version":        "0.81.1",
				"peer":           "111.222.333.444",
				"accept_license": true,
			},

			Commands: map[string]bool{
				fmt.Sprintf("powershell -NoProfile -ExecutionPolicy Bypass -File %s", path.Join(path.Dir(uploadPath), scriptName)): true,
			},

			Uploads: map[string]string{
				path.Join(path.Dir(uploadPath), scriptName): fmt.Sprintf("%s", scriptContent),
			},
		},
	}

	o := new(terraform.MockUIOutput)
	c := new(communicator.MockCommunicator)

	for k, tc := range cases {
		c.Commands = tc.Commands
		c.UploadScripts = tc.Uploads
		c.RemoteScriptPath = uploadPath

		p, err := decodeConfig(
			schema.TestResourceDataRaw(t, Provisioner().(*schema.Provisioner).Schema, tc.Config),
		)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}

		err = p.winStartHabitat(o, c)
		if err != nil {
			t.Fatalf("Test %q failed: %v", k, err)
		}
	}
}

func TestWinProvisioner_winStartHabService(t *testing.T) {
	var uploadPath, scriptName, scriptContent, buildAuthToken string
	uploadPath = os.TempDir()
	scriptName = "win_hab_start.ps1"

	//svcOptions = " --name haborigin/service --topology standalone --binds [ \"database:sqlserver.default\"]"
	// svcBind := Bind{Alias: "database", Service: "sqlserver", Group: "default"}
	buildAuthToken = "1234567890abcdefghijklmnopqrstuvwxyz"
	// service := Service{
	// 	Name:     "haborigin/service",
	// 	Topology: "standalon",
	// 	Binds:    []Bind{svcBind},
	// }

	cases := map[string]struct {
		Config   map[string]interface{}
		Commands map[string]bool
		Uploads  map[string]string
	}{
		"Start Habitat Service with correct options": {
			Config: map[string]interface{}{
				"version":            "0.81.1",
				"peer":               "111.222.333.444",
				"accept_license":     true,
				"builder_auth_token": buildAuthToken,
				"service": []interface{}{
					map[string]interface{}{
						"name":     "core/foo",
						"topology": "standalone",
						"strategy": "none",
						"channel":  "stable",
						"bind": []interface{}{
							map[string]interface{}{
								"alias":   "backend",
								"service": "bar",
								"group":   "default",
							},
						},
					},
					map[string]interface{}{
						"name":      "core/bar",
						"topology":  "standalone",
						"strategy":  "rolling",
						"channel":   "staging",
						"user_toml": "[config]\n port = 8095",
					},
				},
			},

			Commands: map[string]bool{
				fmt.Sprintf("set HAB_AUTH_TOKEN=%s hab svc load core/foo  --topology standalone --strategy none --channel stable --bind backend:bar.default", buildAuthToken): true,
				fmt.Sprintf("set HAB_AUTH_TOKEN=%s hab svc load core/bar  --topology standalone --strategy rolling --channel staging", buildAuthToken):                        true,
				fmt.Sprintf("mkdir C:\\hab\\user\\bar\\config"): true,
			},

			Uploads: map[string]string{
				path.Join(path.Dir(uploadPath), scriptName):          fmt.Sprintf("%s", scriptContent),
				path.Join("C:\\hab\\user\\bar\\config", "user.toml"): "[config]\n port = 8095",
			},
		},
	}

	o := new(terraform.MockUIOutput)
	c := new(communicator.MockCommunicator)

	for k, tc := range cases {
		c.Commands = tc.Commands
		c.Uploads = tc.Uploads

		p, err := decodeConfig(
			schema.TestResourceDataRaw(t, Provisioner().(*schema.Provisioner).Schema, tc.Config),
		)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}

		var errs []error
		for _, s := range p.Services {
			err = p.winStartHabService(o, c, s)
			if err != nil {
				errs = append(errs, err)
			}
		}

		if len(errs) > 0 {
			for _, e := range errs {
				t.Logf("Test %q failed: %v", k, e)
				t.Fail()
			}
		}
	}
}
