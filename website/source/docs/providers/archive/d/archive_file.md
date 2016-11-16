---
layout: "archive"
page_title: "Archive: archive_file"
sidebar_current: "docs-archive-datasource-archive-file"
description: |-
  Generates an archive from content, a file, or directory of files.
---

# archive\_file

Generates an archive from content, a file, or directory of files.

## Example Usage

```
data "archive_file" "init" {
    type        = "zip"
    source_file = "${path.module}/init.tpl"
    output_path = "${path.module}/files/init.zip"
}
```

## Argument Reference

The following arguments are supported:

NOTE: One of `source_content`, `source_file`, or `source_dir` must be specified.

* `type` - (required) The type of archive to generate.
  NOTE: `zip` is supported.

* `output_path` - (required) The output of the archive file.

* `source_content` - (optional) See [Source Content](#source-content) below for details.

* `source_file` - (optional) Package this file into the archive.

* `source_dir` - (optional) Package entire contents of this directory into the archive.

## Source Content

Each `source_content` supports the following:

* `content` - (required) The content of the file

* `filename` - (required) The filename to store the content as


## Attributes Reference

The following attributes are exported:

* `output_size` - The size of the output archive file.

* `output_sha` - The SHA1 checksum of output archive file.

* `output_base64sha256` - The base64-encoded SHA256 checksum of output archive file.
