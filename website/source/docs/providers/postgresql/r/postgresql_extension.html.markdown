---
layout: "postgresql"
page_title: "PostgreSQL: postgresql_extension"
sidebar_current: "docs-postgresql-resource-postgresql_extension"
description: |-
  Creates and manages an extension on a PostgreSQL server.
---

# postgresql\_extension

The ``postgresql_extension`` resource creates and manages an extension on a PostgreSQL
server.


## Usage

```hcl
resource "postgresql_extension" "my_extension" {
  name = "pg_trgm"
}
```

## Argument Reference

* `name` - (Required) The name of the extension.
