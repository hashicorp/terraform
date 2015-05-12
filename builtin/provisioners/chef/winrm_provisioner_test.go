package chef

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/terraform"
)

func TestResourceProvider_winrmInstallChefClient(t *testing.T) {
	cases := map[string]struct {
		Config        *terraform.ResourceConfig
		Commands      map[string]bool
		UploadScripts map[string]string
	}{
		"Default": {
			Config: testConfig(t, map[string]interface{}{
				"node_name":              "nodename1",
				"run_list":               []interface{}{"cookbook::recipe"},
				"server_url":             "https://chef.local",
				"validation_client_name": "validator",
				"validation_key_path":    "validator.pem",
			}),

			Commands: map[string]bool{
				"powershell -NoProfile -ExecutionPolicy Bypass -File ChefClient.ps1": true,
			},

			UploadScripts: map[string]string{
				"ChefClient.ps1": defaultWinRMInstallScript,
			},
		},

		"Proxy": {
			Config: testConfig(t, map[string]interface{}{
				"http_proxy":             "http://proxy.local",
				"no_proxy":               []interface{}{"http://local.local", "http://local.org"},
				"node_name":              "nodename1",
				"run_list":               []interface{}{"cookbook::recipe"},
				"server_url":             "https://chef.local",
				"validation_client_name": "validator",
				"validation_key_path":    "validator.pem",
			}),

			Commands: map[string]bool{
				"powershell -NoProfile -ExecutionPolicy Bypass -File ChefClient.ps1": true,
			},

			UploadScripts: map[string]string{
				"ChefClient.ps1": proxyWinRMInstallScript,
			},
		},

		"Version": {
			Config: testConfig(t, map[string]interface{}{
				"node_name":              "nodename1",
				"run_list":               []interface{}{"cookbook::recipe"},
				"server_url":             "https://chef.local",
				"validation_client_name": "validator",
				"validation_key_path":    "validator.pem",
				"version":                "11.18.6",
			}),

			Commands: map[string]bool{
				"powershell -NoProfile -ExecutionPolicy Bypass -File ChefClient.ps1": true,
			},

			UploadScripts: map[string]string{
				"ChefClient.ps1": versionWinRMInstallScript,
			},
		},
	}

	r := new(ResourceProvisioner)
	o := new(terraform.MockUIOutput)
	c := new(communicator.MockCommunicator)

	for k, tc := range cases {
		c.Commands = tc.Commands
		c.UploadScripts = tc.UploadScripts

		p, err := r.decodeConfig(tc.Config)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}

		p.useSudo = false

		err = p.winrmInstallChefClient(o, c)
		if err != nil {
			t.Fatalf("Test %q failed: %v", k, err)
		}
	}
}

func TestResourceProvider_winrmCreateConfigFiles(t *testing.T) {
	cases := map[string]struct {
		Config   *terraform.ResourceConfig
		Commands map[string]bool
		Uploads  map[string]string
	}{
		"Default": {
			Config: testConfig(t, map[string]interface{}{
				"node_name":              "nodename1",
				"run_list":               []interface{}{"cookbook::recipe"},
				"server_url":             "https://chef.local",
				"validation_client_name": "validator",
				"validation_key_path":    "test-fixtures/validator.pem",
			}),

			Commands: map[string]bool{
				fmt.Sprintf("if not exist %q mkdir %q", windowsConfDir, windowsConfDir): true,
			},

			Uploads: map[string]string{
				"C:/chef/validation.pem":  "VALIDATOR-PEM-FILE",
				"C:/chef/client.rb":       defaultWinRMClientConf,
				"C:/chef/first-boot.json": `{"run_list":["cookbook::recipe"]}`,
			},
		},

		"Proxy": {
			Config: testConfig(t, map[string]interface{}{
				"http_proxy":             "http://proxy.local",
				"https_proxy":            "https://proxy.local",
				"no_proxy":               []interface{}{"http://local.local", "https://local.local"},
				"node_name":              "nodename1",
				"run_list":               []interface{}{"cookbook::recipe"},
				"server_url":             "https://chef.local",
				"validation_client_name": "validator",
				"validation_key_path":    "test-fixtures/validator.pem",
			}),

			Commands: map[string]bool{
				fmt.Sprintf("if not exist %q mkdir %q", windowsConfDir, windowsConfDir): true,
			},

			Uploads: map[string]string{
				"C:/chef/validation.pem":  "VALIDATOR-PEM-FILE",
				"C:/chef/client.rb":       proxyWinRMClientConf,
				"C:/chef/first-boot.json": `{"run_list":["cookbook::recipe"]}`,
			},
		},

		"Attributes": {
			Config: testConfig(t, map[string]interface{}{
				"attributes": []map[string]interface{}{
					map[string]interface{}{
						"key1": []map[string]interface{}{
							map[string]interface{}{
								"subkey1": []map[string]interface{}{
									map[string]interface{}{
										"subkey2a": []interface{}{
											"val1", "val2", "val3",
										},
										"subkey2b": []map[string]interface{}{
											map[string]interface{}{
												"subkey3": "value3",
											},
										},
									},
								},
							},
						},
						"key2": "value2",
					},
				},
				"node_name":              "nodename1",
				"run_list":               []interface{}{"cookbook::recipe"},
				"server_url":             "https://chef.local",
				"validation_client_name": "validator",
				"validation_key_path":    "test-fixtures/validator.pem",
			}),

			Commands: map[string]bool{
				fmt.Sprintf("if not exist %q mkdir %q", windowsConfDir, windowsConfDir): true,
			},

			Uploads: map[string]string{
				"C:/chef/validation.pem": "VALIDATOR-PEM-FILE",
				"C:/chef/client.rb":      defaultWinRMClientConf,
				"C:/chef/first-boot.json": `{"key1":{"subkey1":{"subkey2a":["val1","val2","val3"],` +
					`"subkey2b":{"subkey3":"value3"}}},"key2":"value2","run_list":["cookbook::recipe"]}`,
			},
		},
	}

	r := new(ResourceProvisioner)
	o := new(terraform.MockUIOutput)
	c := new(communicator.MockCommunicator)

	for k, tc := range cases {
		c.Commands = tc.Commands
		c.Uploads = tc.Uploads

		p, err := r.decodeConfig(tc.Config)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}

		p.useSudo = false

		err = p.winrmCreateConfigFiles(o, c)
		if err != nil {
			t.Fatalf("Test %q failed: %v", k, err)
		}
	}
}

const defaultWinRMInstallScript = `
$winver = [System.Environment]::OSVersion.Version | % {"{0}.{1}" -f $_.Major,$_.Minor}

switch ($winver)
{
  "6.0" {$machine_os = "2008"}
  "6.1" {$machine_os = "2008r2"}
  "6.2" {$machine_os = "2012"}
  "6.3" {$machine_os = "2012"}
  default {$machine_os = "2008r2"}
}

if ([System.IntPtr]::Size -eq 4) {$machine_arch = "i686"} else {$machine_arch = "x86_64"}

$url = "http://www.chef.io/chef/download?p=windows&pv=$machine_os&m=$machine_arch&v="
$dest = [System.IO.Path]::GetTempFileName()
$dest = [System.IO.Path]::ChangeExtension($dest, ".msi")
$downloader = New-Object System.Net.WebClient

$http_proxy = ''
if ($http_proxy -ne '') {
	$no_proxy = ''
  if ($no_proxy -eq ''){
    $no_proxy = "127.0.0.1"
  }

  $proxy = New-Object System.Net.WebProxy($http_proxy, $true, ,$no_proxy.Split(','))
  $downloader.proxy = $proxy
}

Write-Host 'Downloading Chef Client...'
$downloader.DownloadFile($url, $dest)

Write-Host 'Installing Chef Client...'
Start-Process -FilePath msiexec -ArgumentList /qn, /i, $dest -Wait
`

const proxyWinRMInstallScript = `
$winver = [System.Environment]::OSVersion.Version | % {"{0}.{1}" -f $_.Major,$_.Minor}

switch ($winver)
{
  "6.0" {$machine_os = "2008"}
  "6.1" {$machine_os = "2008r2"}
  "6.2" {$machine_os = "2012"}
  "6.3" {$machine_os = "2012"}
  default {$machine_os = "2008r2"}
}

if ([System.IntPtr]::Size -eq 4) {$machine_arch = "i686"} else {$machine_arch = "x86_64"}

$url = "http://www.chef.io/chef/download?p=windows&pv=$machine_os&m=$machine_arch&v="
$dest = [System.IO.Path]::GetTempFileName()
$dest = [System.IO.Path]::ChangeExtension($dest, ".msi")
$downloader = New-Object System.Net.WebClient

$http_proxy = 'http://proxy.local'
if ($http_proxy -ne '') {
	$no_proxy = 'http://local.local,http://local.org'
  if ($no_proxy -eq ''){
    $no_proxy = "127.0.0.1"
  }

  $proxy = New-Object System.Net.WebProxy($http_proxy, $true, ,$no_proxy.Split(','))
  $downloader.proxy = $proxy
}

Write-Host 'Downloading Chef Client...'
$downloader.DownloadFile($url, $dest)

Write-Host 'Installing Chef Client...'
Start-Process -FilePath msiexec -ArgumentList /qn, /i, $dest -Wait
`

const versionWinRMInstallScript = `
$winver = [System.Environment]::OSVersion.Version | % {"{0}.{1}" -f $_.Major,$_.Minor}

switch ($winver)
{
  "6.0" {$machine_os = "2008"}
  "6.1" {$machine_os = "2008r2"}
  "6.2" {$machine_os = "2012"}
  "6.3" {$machine_os = "2012"}
  default {$machine_os = "2008r2"}
}

if ([System.IntPtr]::Size -eq 4) {$machine_arch = "i686"} else {$machine_arch = "x86_64"}

$url = "http://www.chef.io/chef/download?p=windows&pv=$machine_os&m=$machine_arch&v=11.18.6"
$dest = [System.IO.Path]::GetTempFileName()
$dest = [System.IO.Path]::ChangeExtension($dest, ".msi")
$downloader = New-Object System.Net.WebClient

$http_proxy = ''
if ($http_proxy -ne '') {
	$no_proxy = ''
  if ($no_proxy -eq ''){
    $no_proxy = "127.0.0.1"
  }

  $proxy = New-Object System.Net.WebProxy($http_proxy, $true, ,$no_proxy.Split(','))
  $downloader.proxy = $proxy
}

Write-Host 'Downloading Chef Client...'
$downloader.DownloadFile($url, $dest)

Write-Host 'Installing Chef Client...'
Start-Process -FilePath msiexec -ArgumentList /qn, /i, $dest -Wait
`

const defaultWinRMClientConf = `log_location            STDOUT
chef_server_url         "https://chef.local"
validation_client_name  "validator"
node_name               "nodename1"`

const proxyWinRMClientConf = `log_location            STDOUT
chef_server_url         "https://chef.local"
validation_client_name  "validator"
node_name               "nodename1"


http_proxy          "http://proxy.local"
ENV['http_proxy'] = "http://proxy.local"
ENV['HTTP_PROXY'] = "http://proxy.local"



https_proxy          "https://proxy.local"
ENV['https_proxy'] = "https://proxy.local"
ENV['HTTPS_PROXY'] = "https://proxy.local"


no_proxy "http://local.local,https://local.local"`
