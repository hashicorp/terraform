---
layout: "docs"
page_title: "Provisioner: file"
sidebar_current: "docs-provisioners-file"
description: |-
  The `file` provisioner is used to copy files or directories from the machine executing Terraform to the newly created resource. The `file` provisioner supports both `ssh` and `winrm` type connections.
---

# File Provisioner

The `file` provisioner is used to copy files or directories from the machine
executing Terraform to the newly created resource. The `file` provisioner
supports both `ssh` and `winrm` type [connections](/docs/provisioners/connection.html).

-> **Note:** Provisioners should only be used as a last resort. For most
common situations there are better alternatives. For more information, see
[the main Provisioners page](./).

## Example usage

```hcl
resource "aws_instance" "web" {
  # ...

  # Copies the myapp.conf file to /etc/myapp.conf
  provisioner "file" {
    source      = "conf/myapp.conf"
    destination = "/etc/myapp.conf"
  }

  # Copies the string in content into /tmp/file.log
  provisioner "file" {
    content     = "ami used: ${self.ami}"
    destination = "/tmp/file.log"
  }

  # Copies the configs.d folder to /etc/configs.d
  provisioner "file" {
    source      = "conf/configs.d"
    destination = "/etc"
  }

  # Copies all files and folders in apps/app1 to D:/IIS/webapp1
  provisioner "file" {
    source      = "apps/app1/"
    destination = "D:/IIS/webapp1"
  }
}
```

## Argument Reference

The following arguments are supported:

* `source` - This is the source file or folder. It can be specified as
  relative to the current working directory or as an absolute path. This
  attribute cannot be specified with `content`.

* `content` - This is the content to copy on the destination. If destination is a file,
  the content will be written on that file, in case of a directory a file named
  `tf-file-content` is created. It's recommended to use a file as the destination. A
  [`template_file`](/docs/providers/template/d/file.html) might be referenced in here, or
  any interpolation syntax. This attribute cannot be specified with `source`.

* `destination` - (Required) This is the destination path. It must be specified as an
  absolute path.

## Directory Uploads

The file provisioner is also able to upload a complete directory to the remote machine.
When uploading a directory, there are a few important things you should know.

First, when using the `ssh` connection type the destination directory must already exist.
If you need to create it, use a remote-exec provisioner just prior to the file provisioner
in order to create the directory. When using the `winrm` connection type the destination
directory will be created for you if it doesn't already exist.

Next, the existence of a trailing slash on the source path will determine whether the
directory name will be embedded within the destination, or whether the destination will
be created. An example explains this best:

If the source is `/foo` (no trailing slash), and the destination is `/tmp`, then the contents
of `/foo` on the local machine will be uploaded to `/tmp/foo` on the remote machine. The
`foo` directory on the remote machine will be created by Terraform.

If the source, however, is `/foo/` (a trailing slash is present), and the destination is
`/tmp`, then the contents of `/foo` will be uploaded directly into `/tmp`.

This behavior was adopted from the standard behavior of
[rsync](https://linux.die.net/man/1/rsync).

-> **Note:** Under the covers, rsync may or may not be used.
