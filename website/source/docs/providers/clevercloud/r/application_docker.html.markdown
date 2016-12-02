---
layout: "clevercloud"
page_title: "Clever Cloud: clevercloud_application_docker"
sidebar_current: "docs-clevercloud-resource-application-docker"
description: |-
  Allows Terraform to manage a Docker application on Clever Cloud.
---

# clevercloud\_application\_docker

Allows Terraform to manage a Docker application on Clever Cloud.

## Example Usage

```
resource "clevercloud_application_docker" "myapp" {
    name = "Hello World"
    description = "This is my fresh app"
    region = "par"

    # Instance size
    min_size = "nano"
    max_size = "M"

    # Instance number
    min_count = 1
    max_count = 3

    environment = {
        "POSTGRESQL_HOST" = "${var.pg_host}"
        "POSTGRESQL_PORT" = "${var.pg_port}"
        "POSTGRESQL_USER" = "${var.pg_user}"
        "POSTGRESQL_PASS" = "${var.pg_pass}"
    }
    fqnds = [
        "my-funny-hello-world-app.cleverapps.io",
        "stack.grep.news"
    ]
}
```

## Argument Reference

The following arguments are supported:

* `Name` - (Required, string) The name of the application.

* `Description` - (Optional, string) The description of the application.

* `Region` - (Optional, string, default: `par` - Paris) The region of the application.

* `fqdns` - (Optional, list) Specifies the DNS of the application. For pointing your DNS on Clever Cloud, 
  please refer to the [documentation](https://www.clever-cloud.com/doc/admin-console/custom-domain-names/). 
  Clever Cloud provide custom domain names suffixed by `.cleverapps.io`, supporting SSL.

* `environment` - (Optional, map) A key/value map of environment variables to set to the application. Please
  set `restart_on_change` to `true`, to automatically restart the application after update.

* `cancel_on_push` - (Optional, boolean, default: `false`) A "git push" will cancel any ongoing deployment
  and start a new one with the last available commit.

* `separate_build` - (Optional, boolean, default: `false`) Your application will build on a dedicated machine 
  allowing you to use a small scaler to run your application. But, using this option will make your 
  deployment slower (by ~10 seconds)

* `sticky_sessions` - (Optional, boolean, default: `false`) When horizontal scalability is enabled, a user 
  is always served by the same scaler. Some frameworks or technologies require this option.

* `homogeneous` - (Optional, boolean, default: `true`) During a deployment, old scalers are kept up until
  the new instances work. Updates are thus transparent to the user. Your application has to work correctly
  with several scalers in parallel (e.g. for connections to databases).

* `restart_on_change` - (Optional, boolean, default: `false`) Restart instances after a change in Terraform
  configuration. Only supports environment updates.

* `min_size` / `max_size` - (Optional, string, default: `nano`) Define a range of instance size. Clever Cloud will
  choose the best configuration according to your application load. Available values: `pico`, `nano`, `xs`, `s`, `m`, `l`, `xl`

* `min_count` / `max_count` - (Optional, int, default: `1`) Define a range of deployed instance. Clever Cloud will
  choose the best configuration according to your application load.


## Attributes Reference

The following attributes are exported:

* `id` - The instance ID.

* `all_fqdns` - Similar to `fqdns`, but also include DNS set externally (from the console).

* `all_environment` - Similar to `environment`, but also include environment variables set externally (from the console).

* `runtime_name` - Runtime label.

* `git_ssh` - Remote repository to push the application code.

* `git_http` - Remote repository to push the application code.

* `branch` - Expose the branch of the running application.

* `commit_id` - Expose the commit of the running application (or null if not deployed).