---
layout: "swift"
page_title: "Swift: object"
sidebar_current: "docs-swift-resource-object"
description: |-
  For managing objects in a Swift object store.
---

# swift_object

Provides an object resource. Allows the creation of an object in a Swift object store.

## Example Usage

```
# Create a new object in Swift with contents extracted from a local file
resource "swift_object" "test_object" {
    name = "foo.txt" # Object name
    container_name = "${swift_container.test_container_1.name}"
    contents = "${file("foo.txt")}" # Contents of the new object
}
```

```
# Create a new object in Swift with contents specified as a variable.
# NOTE: the path specified will automatically be created
variable "secrets" {
    type = "string"
}

resource "swift_object" "test_object2" {
    name = "path/bar.txt" # Path name/Object name
    container_name = "${swift_container.test_container_1.name}"
    contents = "${var.secrets}"
}
```

## Argument Reference

The following arguments are supported:

* `name` | *string*
	* Name of the object. This name can also have forward slashes, which will act as a pseudo file path identifying the object location within the container (e.g. _path/to/foo.txt_).
	* **Required**
* `container_name` | *string*
	* Name of the container to put this object in.
	* **Required**
* `contents` | *string*
	* The desired contents of the object.
	* **Optional**

If `contents` is not specified, an empty object will be created.
