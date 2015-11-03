---
layout: "google"
page_title: "Google: google_dataflow"
sidebar_current: "docs-google-dataflow"
description: |-
  Creates a dataflow processing job using a direct execution of java and reads/deletes
  the job using the gcloud cli tool
---

# google\_dataflow

Creates a dataflow processing job using a direct execution of java and reads/deletes
the job using the gcloud cli tool
[the official documentation](https://cloud.google.com/dataflow/what-is-google-cloud-dataflow) and
[API](https://cloud.google.com/dataflow/java-sdk/JavaDoc/index).


## Example Usage

```
resource "google_dataflow" "demo" {
    name = "example-dataflow"
    jarfile = "/home/code/go/src/github.com/hashicorp/terraform/builtin/providers/google/test-fixtures/google-cloud-dataflow-java-examples-all-bundled-1.1.1-SNAPSHOT.jar"
    class = "com.google.cloud.dataflow.examples.WordCount"
    optional_args = {
        stagingLocation = "gs://demo-dataflow-bucket"
        runner = "DataflowPipelineRunner"
    }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A unique name for the resource, required by dataflow.
    Changing this forces a new resource to be created.
* `jarfile` - (Required) A java jar file that contains the dataflow code to be
   executed.  Changing this forces a new resource to be created.
* `class` - (required) Name of class in provided jar file to execute.  Changing this
   forces a new resource to be created.
* `optional_args` - Any other command line switch (key) value pairs that need to be
   passed to the java creation call.  This will only be used during job creationg.
   Changing this forces a new resource to be created.

## Attributes Reference

The following attributes are exported:

* `name` - The name of the resource.
* `jarfile` - The jarfile that contains the dataflow job to execute
* `class` - Name of the class (com.google....) that should be executed to start the job.
* `optional_args` - Additional command line swtiches to pass to java creation command.
* `jobids` - A single dataflow jar file can create multiple dataflow jobs.  This is a list
             of all jobids created by this resource
* `job_states` - A list of all job states for all jobids created by this resource
