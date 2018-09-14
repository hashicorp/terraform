---
description: |
    The Chef solo Terraform provisioner installs and configures software on machines
    spun up by Terraform by using [chef-solo](https://docs.chef.io/chef_solo.html).
    It is inspired and based off the Chef Solo Packer provisioner.
layout: docs
page_title: 'Chef Solo - Provisioners'
sidebar_current: 'docs-provisioners-chef-solo'
---

# Chef Solo Provisioner

Type: `chef-solo`

The Chef solo Terraform provisioner installs and configures software on machines
spun up by Terraform by using [chef-solo](https://docs.chef.io/chef_solo.html).
It is inspired and based off the Chef solo Packer provisioner.

Cookbooks can be uploaded from your local machine to the remote machine or remote
paths can be used. The provisioner will also install Chef onto your machine if it
isn't already installed, using the official Chef installers provided by Chef Inc.

## Example Usages

The following example is fully functional and expects cookbooks in the "cookbooks"
directory relative to your working directory. It will install the specified
version of Chef, upload the "cookbooks" directory to the remote machine,
and execute the `book::recipe` recipe using the specified JSON attributes.

```hcl
resource "aws_instance" "web" {
  # ...

  provisioner "chef-solo" {
    version         = "12"
    cookbook_paths  = ["cookbooks"]
    run_list        = ["book::recipe"]
    json            = <<-EOF
      {
        "a": "b",
        "c": "d"
      }
    EOF
  }
}
```

Specifying static JSON attributes only gets us so far though, in which case we can use
[template files](https://www.terraform.io/docs/providers/template/d/file.html)
instead. For example, if we want to pass in the IP of a resource managed by Terraform,
we can create a `attributes.json.tpl` file locally with the following contents:

```json
{
  "node": {
    "web": "${web_node_ip}"
  }
}
```

Now our Terraform script would look like this:

```hcl
resource "aws_instance" "web" {
  # ...

  provisioner "chef-solo" {
    version         = "12"
    cookbook_paths  = ["cookbooks"]
    run_list        = ["book::recipe"]
    json            = "${data.template_file.web_attributes.rendered}"
  }
}

data "template_file" "web_attributes": {
  template = "${file("attributes.json.tpl")}"
  vars {
    web_node_ip = ${aws_instance.web.private_ip}"
  }
}
```

This renders the template file with the provided IP so that it's available when
running Chef solo.

## Configuration Reference

The reference of available configuration options is listed below. No
configuration is actually required, but at least `run_list` is recommended.
Unless otherwise specified, the default values of all the options are empty.

-   `config_template` (string) - The contents of a
    [solo.rb](https://docs.chef.io/config_rb_solo.html) config template.
    By default, Terraform will only set configuration it needs to match the
    settings provided in the provisioner configuration. If you need to set any
    configuration that this provisioner doesn't support, then you should use a
    custom configuration template. See the dedicated "Chef Configuration" section
    below for more details.

-   `cookbook_paths` (array of strings) - This is an array of paths to Chef
    "cookbooks" directories on your local filesystem. These will be uploaded
    to the remote machine in the directory specified by the `staging_directory`.

-   `data_bags_path` (string) - The path to the data bags directory on your
    local filesystem. These will be uploaded to the remote machine in the
    directory specified by the `staging_directory`.

-   `encrypted_data_bag_secret_path` (string) - The path to the file containing
    the secret for encrypted data bags.

-   `environment` (string) - The name of the Chef environment to use when
    uploading different Chef environments via `environments_path`.

-   `environments_path` (string) - The path to the "environments" directory on
    your local filesystem. These will be uploaded to the remote machine in the
    directory specified by the `staging_directory`.

-   `execute_command` (string) - The command used to execute Chef. This has
    various configuration template variables available to use. See below for
    more information.

-   `guest_os_type` (string) - The target guest OS type, either "unix" or
    "windows". Setting this to "windows" will cause the provisioner to use
    Windows friendly paths and commands. By default, this is detected by the
    connection type you're using for provisioning, i.e. "unix" for "ssh",
    and "windows" for "winrm".

-   `install_command` (string) - The command used to install Chef. This has
    various configuration template variables available to use. See below for
    more information.

-   `json` (object) - An arbitrary mapping of JSON that will be available as
    node attributes while running Chef.

-   `keep_log` (boolean) - By default, the log for the Chef run will be logged to
    "`staging_directory`/chef.log". Set this to "false" if you don't want to
    keep the log.

-   `prevent_sudo` (boolean) - By default, the configured commands that are
    executed to install and run Chef are executed with `sudo`. If this is true,
    then the sudo will be omitted. This is ignored when `guest_os_type` is
    "windows".

-   `remote_cookbook_paths` (array of strings) - A list of paths on the remote
    machine where cookbooks will already exist. These may exist from a previous
    provisioner or step. If specified, Chef will be configured to look for
    cookbooks here.

-   `roles_path` (string) - The path to the "roles" directory on your
    local filesystem. These will be uploaded to the remote machine in the
    directory specified by the `staging_directory`.

-   `run_list` (array of strings) - The [run
    list](https://docs.chef.io/run_lists.html) for Chef.

-   `skip_install` (boolean) - If true, Chef will not automatically be installed
    on the machine using the Chef omnibus installers. By default, this is false.

-   `staging_directory` (string) - This is the directory where all the
    configuration of Chef by Terraform will be placed. By default this is
    "/tmp/terraform-chef-solo" when `guest_os_type` is unix and
    "C:/Windows/Temp/terraform-chef-solo" when windows. This directory doesn't
    need to exist but must have proper permissions so that the user that Terraform
    uses is able to create directories and write into this directory. If the permissions
    are not correct, use a shell provisioner prior to this to configure it properly.

-   `version` (string) - The version of Chef to be installed. If left empty,
    this will install the latest version of Chef.

## Chef Configuration

By default, Terraform creates a simple Chef "solo.rb" configuration file in order to
set the options specified by the provisioner. If you'd like to set your own custom
configurations not supported above and don't want to submit a feature request, you
can always specify a different configuration template through the `config_template`
setting. However, if you found it useful, then someone else might too! So please do
submit those requests!

The "solo.rb" file is generated from a [Golang
template](https://golang.org/pkg/text/template/), and has the following variables
available to use. All of them take the value of the correspondingly named configuration settings above. 

-   `CookbookPaths` is the path(s) to the cookbooks that were uploaded to the machine.
     Note that these paths are already quoted. See below example.
-   `DataBagsPath` is the path to the data bags directory.
-   `EncryptedDataBagSecretPath` - The path to the encrypted data bag secret.
-   `Environment` - The current Chef environment.
-   `EnvironmentsPath` - The path to the environments directory..
-   `JSON` - The JSON attributes. You typically would not need to use these directly
     in the template.
-   `KeepLog` - A boolean flag for whether or not you want to keep the log.
-   `RolesPath` - The path to the roles directory.

In addition to the above variables, we have the following variables that do not match
any configuration settings.

-   `JSONPath` - The path to where the JSON attributes file is.
     This has the value of "{{.StagingDirectory}}/attributes.json".
-   `LogPath` - The path to where the log file is stored.
     This has the value of "{{.StagingDirectory}}/chef.log".

### How do I use any of this?

For example, right now this provisioner does not support the Chef "log\_level"
setting available in the "solo.rb" file. If we'd like to set it, we'd have to provide
our own configuration template. It would look something like this.

```hcl
resource "aws_instance" "web" {
  # ...

  provisioner "chef-solo" {
    config_template = <<-EOF
      cookbook_path   [{{.CookbookPaths}}]
      json_attribs    "{{.JSONPath}}"
      log_location    "{{.StagingDirectory}}/my-chef-log"
      log_level       :debug
    EOF
    cookbook_paths  = ["cookbooks"]
    run_list        = ["book::recipe"]
  }
}
```

The above Terraform code will create a "solo.rb" file with a debug "log\_level"
to be used for the Chef run.

## Execute Command

By default, Terraform uses the following command to execute Chef for UNIX systems:

```liquid
chef-solo --no-color -c {{.ConfigPath}}
```

When `prevent_sudo` is false, the above command is prefaced with sudo.

And the following for Windows systems:

```liquid
C:/opscode/chef/bin/chef-solo.bat --no-color -c {{.ConfigPath}}
```

While you can change this command through the `execute_command` setting, the
only template variable available to you here is "{{.ConfigPath}}" which will
evaluate to the path of the "solo.rb" config file.

This might be useful if you'd like to use `chef-client --local-mode` instead
of `chef-solo` if some configuration setting is not supported by Chef Solo.

```hcl
resource "aws_instance" "web" {
  # ...

  provisioner "chef-solo" {
    config_template = <<- EOF
      # ...
    EOF
    execute_command = "chef-client -z -c {{.ConfigPath}}"
  }
}
```

## Install Command

By default, Terraform uses the following command to execute Chef for UNIX systems:

```liquid
sh -c 'command -v chef-solo || (curl -LO https://omnitruck.chef.io/install.sh && sh install.sh{{if .Version}} -v {{.Version}}{{end}})'
```

When `prevent_sudo` is false, the above command is prefaced with sudo.

And the following for Windows systems:

```liquid
powershell.exe -Command \". { iwr -useb https://omnitruck.chef.io/install.ps1 } | iex; Install-Project{{if .Version}} -version {{.Version}}{{end}}\"
```

Similarly, you can change this through the `install_command` setting. The only
template variable available to you here is "{{.Version}}" which evaluates to whatever
the `version` setting is.
