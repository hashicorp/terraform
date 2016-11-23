---
layout: "gpg"
page_title: "GPG: gpg_message"
sidebar_current: "docs-gpg-datasource-gpg-message"
description: |-
  Decrypts GPG data provided.
---

# archive\_file

Generates an archive from content, a file, or directory of files.

## Example Usage

```
data "gpg_message" "init" {
    encrypted_data = <<EOF
-----BEGIN PGP MESSAGE-----
Version: GnuPG v2

hQEMA576X39jwPWtAQf/ewn4dIw2APLjRBLU0f7BrR3jyK+zrXdxpbc5ZtVvheFC
Los3EfRmypAk7cfxp4yXvyegLdckVAlEj4EUBQxHmeP6iaPt9VW58OEnCYsTCPcc
P7prgM9Ms3u40ZNlp0TQzTAaHSC7HNHGZw0G5ra4mm5EaKH+YP/WPQu/bEP4U8X0
6I8HVqkQKHBBODDqtdhTeOQvARsWhOBRHHB2pzG533MkE9Ck4nnb/tA1LhgGFIHO
pPXbEFByuwCu/fjdBSCkVERO/g/l6Ji2anjxckTmjTeQfB+QFGJO6c12SqnJ7zGf
yuwJ3+OF2DFF9I+ri7FvUkdiQsyNNG/+Xcd7oqO2vdJGAXYPK1kxUoFUtphHSh2I
oCYSxZq4V6mAn93nfyhFpI/qb5eeUlZBLYxIYVqWHrQIEKU4HOSsQkV5aS5BzF6U
aKII3VUjpQ==
=porM
-----END PGP MESSAGE-----
EOF
    key_directory = "gpgkeys"
}
```

## Argument Reference

The following arguments are supported:

NOTE: One of `source_content_filename` (with `source_content`), `source_file`, or `source_dir` must be specified.

* `encrypted_data` - (required) The GPG encrypted blob.

* `key_directory` - (optional) The path to the GPG directory, defaults to ${HOME}/.gnupg.

## Attributes Reference

The following attributes are exported:

* `decrypted_data` - The decrypted data.
