---
layout: "ignition"
page_title: "Ignition: ignition_file"
sidebar_current: "docs-ignition-datasource-file"
description: |-
  Describes a file to be written in a particular filesystem.
---

# ignition\_file

Describes a file to be written in a particular filesystem.

## Example Usage

File with inline content:

```hcl
data "ignition_file" "hello" {
	filesystem = "foo"
	path = "/hello.txt"
	content {
		content = "Hello World!"
	}
}
```

File with remote content:

```hcl
data "ignition_file" "hello" {
	filesystem = "qux"
	path = "/hello.txt"
	source {
		source = "http://example.com/hello.txt.gz"
		compression = "gzip"
		verification = "sha512-0123456789abcdef0123456789...456789abcdef"
	}
}
```

## Argument Reference

The following arguments are supported:

* `filesystem` - (Optional) The internal identifier of the filesystem. This matches the last filesystem with the given identifier. This should be a valid name from a _ignition\_filesystem_ resource.

* `path` - (Optional) The absolute path to the file.

* `content` - (Optional) Block to provide the file content inline.

* `source` - (Optional) Block to retrieve the file content from a remote location.

	__Note__: `content` and `source` are mutually exclusive

* `mode` - (Optional) The list of partitions and their configuration for 
this particular disk..

* `uid` - (Optional) The user ID of the owner.

* `gid` - (Optional) The group ID of the owner.

The `content` block supports:
 
* `mime` - (Required) MIME format of the content (default _text/plain_).

* `content` - (Required) Content of the file.

The `source` block supports:

* `source` - (Required) The URL of the file contents. When using http, it is advisable to use the verification option to ensure the contents haven’t been modified.

* `compression` - (Optional) The type of compression used on the contents (null or gzip).

* `verification` - (Optional) The hash of the config, in the form _\<type\>-\<value\>_ where type is sha512.

## Attributes Reference

The following attributes are exported:

* `id` - ID used to reference this resource in _ignition_config_.