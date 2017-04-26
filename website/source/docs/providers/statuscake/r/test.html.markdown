---
layout: "statuscake"
page_title: "StatusCake: statuscake_test"
sidebar_current: "docs-statuscake-test"
description: |-
  The statuscake_test resource allows StatusCake tests to be managed by Terraform.
---

# statuscake\_test

The test resource allows StatusCake tests to be managed by Terraform.

## Example Usage

```hcl
resource "statuscake_test" "google" {
  website_name = "google.com"
  website_url  = "www.google.com"
  test_type    = "HTTP"
  check_rate   = 300
  contact_id   = 12345
}
```

## Argument Reference

The following arguments are supported:

* `website_name` - (Required) This is the name of the test and the website to be monitored.
* `website_url` - (Required) The URL of the website to be monitored
* `check_rate` - (Optional) Test check rate in seconds. Defaults to 300
* `contact_id` - (Optional) The id of the contact group to be add to the test.  Each test can have only one.
* `test_type` - (Required) The type of Test. Either HTTP or TCP
* `paused` - (Optional) Whether or not the test is paused. Defaults to false.
* `timeout` - (Optional) The timeout of the test in seconds.
* `confirmations` - (Optional) The number of confirmation servers to use in order to detect downtime. Defaults to 0.
* `port` - (Optional) The port to use when specifying a TCP test.
* `trigger_rate` - (Optional) The number of minutes to wait before sending an alert. Default is `5`.


## Attributes Reference

The following attribute is exported:

* `test_id` - A unique identifier for the test.
