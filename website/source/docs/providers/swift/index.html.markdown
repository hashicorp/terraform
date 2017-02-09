---
layout: "swift"
page_title: "Provider: Swift"
sidebar_current: "docs-swift-index"
description: |-
  The Swift provider is used to interact directly with a Swift object store.
---

# Swift Provider

The Swift provider is used to interact directly with a Swift-compatible object store.

Use the navigation to the left to read about the available resources.

<div class="alert alert-block alert-info">
<strong>Note:</strong> The Swift provider is brand new.
It is ready to be used but many features are still being added. If there
is a Swift feature missing, please report it in the GitHub repo.
</div>

## Example Usage

Here is an example that will setup the following:

+ A container.
+ An object within that container.

(create this as myswift.tf and run terraform commands from this directory):

```hcl
provider "swift" {
    username = "" # The user name to use for Swift API operations.
    api_key = ""  # The API key to use for Swift API operations.
    auth_url = "" # The swifth object storage url to use for authentication.
    # Optional. Alternate object storage url to access containers in
    # (defaults to storage url returned by authentication api).
    storage_url = ""
}

# This will create a new container for object in the Swift object store.
resource "swift_container" "test_container_1" {
    name = "test_container_1"
}

# This will create an object under the specified container in
# the Swift object store.
resource "swift_object" "test_object_1" {
    name = "foo.txt" # Object name
    container_name = "${swift_container.test_container_1.name}"
    contents = "${file("foo.txt")}" # Contents of the new object
}
```

You'll need to provide your Swift username, API key, and authentication endpoint,
so that Terraform can create a connection. If you don't want to put
credentials in your configuration file, you can leave them
out:

```
provider "swift" {}
```

...and instead set these environment variables:

- **SWIFT_USERNAME**: Your Swift username
- **SWIFT_API_KEY**: Your Swift API key
- **SWIFT_AUTH_URL**: Your Swift authentication endpoint URL
