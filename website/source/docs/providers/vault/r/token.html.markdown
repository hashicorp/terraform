---
layout: "vault"
page_title: "Vault: vault_token"
sidebar_current: "docs-vault-resource-token"
description: |-
  Provides a resource for managing Vault tokens.
---

# vault\_token

Provides a resource for managing Vault tokens.

## Example Usage

```
resource "vault_token" "foo" {
  display_name       = "some-service-token"
  ttl                = "20m"
  num_uses           = 10
  policies           = ["redis-admin", "rmq-admin"]
  no_default_profile = true
  meta {
    env = "gamma"
  }
}
```

## Argument Reference

More detail about each of these fields can be found in the [official
Vault API documentation](https://www.vaultproject.io/docs/auth/token.html).

The following arguments are supported:

* `policies` (Unordered List of Strings, Optional) A list of policies for the token. This must be a subset of the policies belonging to the token making the request, unless root. If not specified, defaults to all the policies of the calling token.
* `meta` (Map of Strings, Optional) Metadata to be passed through to the audit backends.
* `no_default_policy` (Boolean, Default: `false`) If true the default policy will not be a part of this token's policy set.
* `ttl` (String, Optional) The TTL period of the token, provided as "1h", where hour is the largest suffix. If not provided, the token is valid for the default lease TTL, or indefinitely if the root policy is used. Must be a string parseable by [`time.ParseDuration`](https://golang.org/pkg/time/#ParseDuration).
* `display_name` (String, Default: `"token"`) The display name of the token.
* `num_uses` (Integer, Optional) The maximum uses for the given token. This can be used to create a one-time-token or limited use token. Defaults to 0, which has no limit to number of uses.
