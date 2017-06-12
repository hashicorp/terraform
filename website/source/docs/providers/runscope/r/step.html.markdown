---
layout: "runscope"
page_title: "Runscope: runscope_step"
sidebar_current: "docs-runscope-resource-step"
description: |-
  Provides a Runscope step resource.
---

# runscope\_step

A [step](https://www.runscope.com/docs/api/steps) resource.
API tests are comprised of a series of steps, most often HTTP requests.
In addition to requests, you can also add additional types of steps to
your tests like pauses and conditions.

### Creating a step
```hcl
resource "runscope_step" "main_page" {
  bucket_id      = "${runscope_bucket.bucket.id}"
  test_id        = "${runscope_test.test.id}"
  step_type      = "request"
  url            = "http://example.com"
  method         = "GET"
  variables      = [
  	{
  	   name     = "httpStatus"
  	   source   = "response_status"
  	},
  	{
  	   name     = "httpContentEncoding"
  	   source   = "response_header"
  	   property = "Content-Encoding"
  	},
  ]
  assertions     = [
  	{
  	   source     = "response_status"
       comparison = "equal_number"
       value      = "200"
  	},
  	{
  	   source     = "response_json"
       comparison = "equal"
       value      = "c5baeb4a-2379-478a-9cda-1b671de77cf9",
       property   = "data.id"
  	},
  ],
  headers        = [
  	{
  		header = "Accept-Encoding",
  		value  = "application/json"
  	},
  	{
  		header = "Accept-Encoding",
  		value  = "application/xml"
  	},
  	{
  		header = "Authorization",
  		value  = "Bearer bb74fe7b-b9f2-48bd-9445-bdc60e1edc6a",
	}
  ]
}

resource "runscope_test" "test" {
  bucket_id   = "${runscope_bucket.bucket.id}"
  name        = "runscope test"
  description = "This is a test test..."
}

resource "runscope_bucket" "bucket" {
  name      = "terraform-provider-test"
  team_uuid = "dfb75aac-eeb3-4451-8675-3a37ab421e4f"
}
```

## Argument Reference

The following arguments are supported:

* `bucket_id` - (Required) The id of the bucket to associate this step with.
* `test_id` - (Required) The id of the test to associate this step with.
* `step_type` - (Required) The type of step.
 * [request](#request-steps)
 * pause
 * condition
 * ghost
 * subtest

### Request steps
When creating a `request` type of step the additional arguments also apply:

* `method` - (Required) The HTTP method for this request step.
* `variables` - (Optional) A list of variables to extract out of the HTTP response from this request. Variables documented below.
* `assertions` - (Optional) A list of assertions to apply to the HTTP response from this request. Assertions documented below.
* `headers` - (Optional) A list of headers to apply to the request. Headers documented below.
* `body` - (Optional) A string to use as the body of the request.

Variables (`variables`) supports the following:

* `name` - (Required) Name of the variable to define.
* `property` - (Required) The name of the source property. i.e. header name or json path
* `source` - (Required) The variable source, for list of allowed values see: https://www.runscope.com/docs/api/steps#assertions

Assertions (`assertions`) supports the following:

* `source` - (Required) The assertion source, for list of allowed values see: https://www.runscope.com/docs/api/steps#assertions
* `property` - (Optional) The name of the source property. i.e. header name or json path
* `comparison` - (Required) The assertion comparison to make i.e. `equals`, for list of allowed values see: https://www.runscope.com/docs/api/steps#assertions
* `value` - (Optional) The value the `comparison` will use

**Example Assertions**

Status Code == 200

```json
"assertions": [
    {
        "source": "response_status",
        "comparison": "equal_number",
        "value": 200
    }
]
```

JSON element 'address' contains the text "avenue"


```json
"assertions": [
    {
        "source": "response_json",
        "property": "address",
        "comparison": "contains",
        "value": "avenue"
    }
]
```

Response Time is faster than 1 second.


```json
"assertions": [
    {
        "source": "response_time",
        "comparison": "is_less_than",
        "value": 1000
    }
]
```

The `headers` list supports the following:

* `header` - (Required) The name of the header
* `value` - (Required) The name header value

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the step.
