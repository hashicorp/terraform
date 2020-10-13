package chef

import (
	"fmt"
	"path"
	"testing"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func TestResourceProvider_windowsInstallChefClient(t *testing.T) {
	cases := map[string]struct {
		Config        map[string]interface{}
		Commands      map[string]bool
		UploadScripts map[string]string
	}{
		"Default": {
			Config: map[string]interface{}{
				"node_name":  "nodename1",
				"run_list":   []interface{}{"cookbook::recipe"},
				"server_url": "https://chef.local",
				"user_name":  "bob",
				"user_key":   "USER-KEY",
			},

			Commands: map[string]bool{
				"powershell -NoProfile -ExecutionPolicy Bypass -File ChefClient.ps1": true,
			},

			UploadScripts: map[string]string{
				"ChefClient.ps1": defaultWindowsInstallScript,
			},
		},

		"Proxy": {
			Config: map[string]interface{}{
				"http_proxy": "http://proxy.local",
				"no_proxy":   []interface{}{"http://local.local", "http://local.org"},
				"node_name":  "nodename1",
				"run_list":   []interface{}{"cookbook::recipe"},
				"server_url": "https://chef.local",
				"user_name":  "bob",
				"user_key":   "USER-KEY",
			},

			Commands: map[string]bool{
				"powershell -NoProfile -ExecutionPolicy Bypass -File ChefClient.ps1": true,
			},

			UploadScripts: map[string]string{
				"ChefClient.ps1": proxyWindowsInstallScript,
			},
		},

		"Version": {
			Config: map[string]interface{}{
				"node_name":  "nodename1",
				"run_list":   []interface{}{"cookbook::recipe"},
				"server_url": "https://chef.local",
				"user_name":  "bob",
				"user_key":   "USER-KEY",
				"version":    "11.18.6",
			},

			Commands: map[string]bool{
				"powershell -NoProfile -ExecutionPolicy Bypass -File ChefClient.ps1": true,
			},

			UploadScripts: map[string]string{
				"ChefClient.ps1": versionWindowsInstallScript,
			},
		},

		"Channel": {
			Config: map[string]interface{}{
				"channel":    "current",
				"node_name":  "nodename1",
				"run_list":   []interface{}{"cookbook::recipe"},
				"server_url": "https://chef.local",
				"user_name":  "bob",
				"user_key":   "USER-KEY",
				"version":    "11.18.6",
			},

			Commands: map[string]bool{
				"powershell -NoProfile -ExecutionPolicy Bypass -File ChefClient.ps1": true,
			},

			UploadScripts: map[string]string{
				"ChefClient.ps1": channelWindowsInstallScript,
			},
		},
	}

	o := new(terraform.MockUIOutput)
	c := new(communicator.MockCommunicator)

	for k, tc := range cases {
		c.Commands = tc.Commands
		c.UploadScripts = tc.UploadScripts

		p, err := decodeConfig(
			schema.TestResourceDataRaw(t, Provisioner().(*schema.Provisioner).Schema, tc.Config),
		)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}

		p.useSudo = false

		err = p.windowsInstallChefClient(o, c)
		if err != nil {
			t.Fatalf("Test %q failed: %v", k, err)
		}
	}
}

func TestResourceProvider_windowsCreateConfigFiles(t *testing.T) {
	cases := map[string]struct {
		Config   map[string]interface{}
		Commands map[string]bool
		Uploads  map[string]string
	}{
		"Default": {
			Config: map[string]interface{}{
				"ohai_hints": []interface{}{"testdata/ohaihint.json"},
				"node_name":  "nodename1",
				"run_list":   []interface{}{"cookbook::recipe"},
				"secret_key": "SECRET-KEY",
				"server_url": "https://chef.local",
				"user_name":  "bob",
				"user_key":   "USER-KEY",
			},

			Commands: map[string]bool{
				fmt.Sprintf("cmd /c if not exist %q mkdir %q", windowsConfDir, windowsConfDir): true,
				fmt.Sprintf("cmd /c if not exist %q mkdir %q",
					path.Join(windowsConfDir, "ohai/hints"),
					path.Join(windowsConfDir, "ohai/hints")): true,
			},

			Uploads: map[string]string{
				windowsConfDir + "/client.rb":                 defaultWindowsClientConf,
				windowsConfDir + "/encrypted_data_bag_secret": "SECRET-KEY",
				windowsConfDir + "/first-boot.json":           `{"run_list":["cookbook::recipe"]}`,
				windowsConfDir + "/ohai/hints/ohaihint.json":  "OHAI-HINT-FILE",
				windowsConfDir + "/bob.pem":                   "USER-KEY",
			},
		},

		"Proxy": {
			Config: map[string]interface{}{
				"http_proxy":      "http://proxy.local",
				"https_proxy":     "https://proxy.local",
				"no_proxy":        []interface{}{"http://local.local", "https://local.local"},
				"node_name":       "nodename1",
				"run_list":        []interface{}{"cookbook::recipe"},
				"secret_key":      "SECRET-KEY",
				"server_url":      "https://chef.local",
				"ssl_verify_mode": "verify_none",
				"user_name":       "bob",
				"user_key":        "USER-KEY",
			},

			Commands: map[string]bool{
				fmt.Sprintf("cmd /c if not exist %q mkdir %q", windowsConfDir, windowsConfDir): true,
			},

			Uploads: map[string]string{
				windowsConfDir + "/client.rb":                 proxyWindowsClientConf,
				windowsConfDir + "/first-boot.json":           `{"run_list":["cookbook::recipe"]}`,
				windowsConfDir + "/encrypted_data_bag_secret": "SECRET-KEY",
				windowsConfDir + "/bob.pem":                   "USER-KEY",
			},
		},

		"Attributes JSON": {
			Config: map[string]interface{}{
				"attributes_json": `{"key1":{"subkey1":{"subkey2a":["val1","val2","val3"],` +
					`"subkey2b":{"subkey3":"value3"}}},"key2":"value2"}`,
				"node_name":  "nodename1",
				"run_list":   []interface{}{"cookbook::recipe"},
				"secret_key": "SECRET-KEY",
				"server_url": "https://chef.local",
				"user_name":  "bob",
				"user_key":   "USER-KEY",
			},

			Commands: map[string]bool{
				fmt.Sprintf("cmd /c if not exist %q mkdir %q", windowsConfDir, windowsConfDir): true,
			},

			Uploads: map[string]string{
				windowsConfDir + "/client.rb":                 defaultWindowsClientConf,
				windowsConfDir + "/encrypted_data_bag_secret": "SECRET-KEY",
				windowsConfDir + "/bob.pem":                   "USER-KEY",
				windowsConfDir + "/first-boot.json": `{"key1":{"subkey1":{"subkey2a":["val1","val2","val3"],` +
					`"subkey2b":{"subkey3":"value3"}}},"key2":"value2","run_list":["cookbook::recipe"]}`,
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

		p.useSudo = false

		err = p.windowsCreateConfigFiles(o, c)
		if err != nil {
			t.Fatalf("Test %q failed: %v", k, err)
		}
	}
}

const defaultWindowsInstallScript = `
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

$url = "http://omnitruck.chef.io/stable/chef/download?p=windows&pv=$machine_os&m=$machine_arch&v="
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

const proxyWindowsInstallScript = `
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

$url = "http://omnitruck.chef.io/stable/chef/download?p=windows&pv=$machine_os&m=$machine_arch&v="
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

const versionWindowsInstallScript = `
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

$url = "http://omnitruck.chef.io/stable/chef/download?p=windows&pv=$machine_os&m=$machine_arch&v=11.18.6"
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
const channelWindowsInstallScript = `
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

$url = "http://omnitruck.chef.io/current/chef/download?p=windows&pv=$machine_os&m=$machine_arch&v=11.18.6"
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

const defaultWindowsClientConf = `log_location            STDOUT
chef_server_url         "https://chef.local/"
node_name               "nodename1"`

const proxyWindowsClientConf = `log_location            STDOUT
chef_server_url         "https://chef.local/"
node_name               "nodename1"

http_proxy          "http://proxy.local"
ENV['http_proxy'] = "http://proxy.local"
ENV['HTTP_PROXY'] = "http://proxy.local"

https_proxy          "https://proxy.local"
ENV['https_proxy'] = "https://proxy.local"
ENV['HTTPS_PROXY'] = "https://proxy.local"

no_proxy          "http://local.local,https://local.local"
ENV['no_proxy'] = "http://local.local,https://local.local"

ssl_verify_mode  :verify_none`
