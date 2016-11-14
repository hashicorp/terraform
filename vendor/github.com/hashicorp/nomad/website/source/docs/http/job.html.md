---
layout: "http"
page_title: "HTTP API: /v1/job"
sidebar_current: "docs-http-job-"
description: |-
  The '/1/job' endpoint is used for CRUD on a single job.
---

# /v1/job

The `job` endpoint is used for CRUD on a single job. By default, the agent's local
region is used; another region can be specified using the `?region=` query parameter.

## GET

<dl>
  <dt>Description</dt>
  <dd>
    Query a single job for its specification and status.
  </dd>

  <dt>Method</dt>
  <dd>GET</dd>

  <dt>URL</dt>
  <dd>`/v1/job/<ID>`</dd>

  <dt>Parameters</dt>
  <dd>
    None
  </dd>

  <dt>Blocking Queries</dt>
  <dd>
    [Supported](/docs/http/index.html#blocking-queries)
  </dd>

  <dt>Returns</dt>
  <dd>

    ```javascript
    {
    "Region": "global",
    "ID": "binstore-storagelocker",
    "Name": "binstore-storagelocker",
    "Type": "service",
    "Priority": 50,
    "AllAtOnce": false,
    "Datacenters": [
        "us2",
        "eu1"
    ],
    "Constraints": [
        {
            "LTarget": "${attr.kernel.os}",
            "RTarget": "windows",
            "Operand": "="
        }
    ],
    "TaskGroups": [
        {
            "Name": "binsl",
            "Count": 5,
            "Constraints": [
                {
                    "LTarget": "${attr.kernel.os}",
                    "RTarget": "linux",
                    "Operand": "="
                }
            ],
            "Tasks": [
                {
                    "Name": "binstore",
                    "Driver": "docker",
                    "Config": {
                        "image": "hashicorp/binstore"
                    },
                    "Constraints": null,
                    "Resources": {
                        "CPU": 500,
                        "MemoryMB": 0,
                        "DiskMB": 0,
                        "IOPS": 0,
                        "Networks": [
                            {
                                "Device": "",
                                "CIDR": "",
                                "IP": "",
                                "MBits": 100,
                                "ReservedPorts": null,
                                "DynamicPorts": null
                            }
                        ]
                    },
                    "Meta": null
                },
                {
                    "Name": "storagelocker",
                    "Driver": "java",
                    "Config": {
                        "image": "hashicorp/storagelocker"
                    },
                    "Constraints": [
                        {
                            "LTarget": "${attr.kernel.arch}",
                            "RTarget": "amd64",
                            "Operand": "="
                        }
                    ],
                    "Resources": {
                        "CPU": 500,
                        "MemoryMB": 0,
                        "DiskMB": 0,
                        "IOPS": 0,
                        "Networks": null
                    },
                    "Meta": null
                }
            ],
            "Meta": {
                "elb_checks": "3",
                "elb_interval": "10",
                "elb_mode": "tcp"
            }
        }
    ],
    "Update": {
        "Stagger": 0,
        "MaxParallel": 0
    },
    "Meta": {
        "foo": "bar"
    },
    "Status": "",
    "StatusDescription": "",
    "CreateIndex": 14,
    "ModifyIndex": 14
    }
    ```

  </dd>
</dl>

<dl>
  <dt>Description</dt>
  <dd>
    Query the allocations belonging to a single job.
  </dd>

  <dt>Method</dt>
  <dd>GET</dd>

  <dt>URL</dt>
  <dd>`/v1/job/<ID>/allocations`</dd>

  <dt>Parameters</dt>
  <dd>
    None
  </dd>

  <dt>Blocking Queries</dt>
  <dd>
    [Supported](/docs/http/index.html#blocking-queries)
  </dd>

  <dt>Returns</dt>
  <dd>

    ```javascript
    [
    {
        "ID": "3575ba9d-7a12-0c96-7b28-add168c67984",
        "EvalID": "151accaa-1ac6-90fe-d427-313e70ccbb88",
        "Name": "binstore-storagelocker.binsl[0]",
        "NodeID": "a703c3ca-5ff8-11e5-9213-970ee8879d1b",
        "JobID": "binstore-storagelocker",
        "TaskGroup": "binsl",
        "DesiredStatus": "run",
        "DesiredDescription": "",
        "ClientStatus": "running",
        "ClientDescription": "",
        "CreateIndex": 16,
        "ModifyIndex": 16
    },
    ...
    ]
    ```

  </dd>
</dl>

<dl>
  <dt>Description</dt>
  <dd>
    Query the evaluations belonging to a single job.
  </dd>

  <dt>Method</dt>
  <dd>GET</dd>

  <dt>URL</dt>
  <dd>`/v1/job/<ID>/evaluations`</dd>

  <dt>Parameters</dt>
  <dd>
    None
  </dd>

  <dt>Blocking Queries</dt>
  <dd>
    [Supported](/docs/http/index.html#blocking-queries)
  </dd>

  <dt>Returns</dt>
  <dd>

    ```javascript
    [
    {
        "ID": "151accaa-1ac6-90fe-d427-313e70ccbb88",
        "Priority": 50,
        "Type": "service",
        "TriggeredBy": "job-register",
        "JobID": "binstore-storagelocker",
        "JobModifyIndex": 14,
        "NodeID": "",
        "NodeModifyIndex": 0,
        "Status": "complete",
        "StatusDescription": "",
        "Wait": 0,
        "NextEval": "",
        "PreviousEval": "",
        "CreateIndex": 15,
        "ModifyIndex": 17
    },
    ...
    ]
    ```

  </dd>
</dl>

<dl>
  <dt>Description</dt>
  <dd>
    Query the summary of a job.
  </dd>

  <dt>Method</dt>
  <dd>GET</dd>

  <dt>URL</dt>
  <dd>`/v1/job/<ID>/summary`</dd>

  <dt>Parameters</dt>
  <dd>
    None
  </dd>

  <dt>Blocking Queries</dt>
  <dd>
    [Supported](/docs/http/index.html#blocking-queries)
  </dd>

  <dt>Returns</dt>
  <dd>

    ```javascript
    {
      "JobID": "example",
      "Summary": {
        "cache": {
          "Queued": 0,
          "Complete": 0,
          "Failed": 0,
          "Running": 1,
          "Starting": 0,
          "Lost": 0
        }
      },
      "CreateIndex": 6,
      "ModifyIndex": 10
    }
    ```

  </dd>
</dl>


## PUT / POST

<dl>
  <dt>Description</dt>
  <dd>
    Registers a new job or updates an existing job
  </dd>

  <dt>Method</dt>
  <dd>PUT or POST</dd>

  <dt>URL</dt>
  <dd>`/v1/job/<ID>`</dd>

  <dt>Parameters</dt>
  <dd>
    <ul>
      <li>
        <span class="param">Job</span>
        <span class="param-flags">required</span>
        The JSON definition of the job. The general structure is given
        by the [job specification](/docs/jobspec/index.html), and matches
        the return response of GET.
      </li>
    </ul>
  </dd>

  <dt>Returns</dt>
  <dd>

    ```javascript
    {
    "EvalID": "d092fdc0-e1fd-2536-67d8-43af8ca798ac",
    "EvalCreateIndex": 35,
    "JobModifyIndex": 34,
    }
    ```

  </dd>
</dl>

<dl>
  <dt>Description</dt>
  <dd>
    Creates a new evaluation for the given job. This can be used to force
    run the scheduling logic if necessary.
  </dd>

  <dt>Method</dt>
  <dd>PUT or POST</dd>

  <dt>URL</dt>
  <dd>`/v1/job/<ID>/evaluate`</dd>

  <dt>Parameters</dt>
  <dd>
    None
  </dd>

  <dt>Returns</dt>
  <dd>

    ```javascript
    {
    "EvalID": "d092fdc0-e1fd-2536-67d8-43af8ca798ac",
    "EvalCreateIndex": 35,
    "JobModifyIndex": 34,
    }
    ```

  </dd>
</dl>

<dl>
  <dt>Description</dt>
  <dd>
    Invoke a dry-run of the scheduler for the job.
  </dd>

  <dt>Method</dt>
  <dd>PUT or POST</dd>

  <dt>URL</dt>
  <dd>`/v1/job/<ID>/plan`</dd>

  <dt>Parameters</dt>
  <dd>
    <ul>
      <li>
        <span class="param">Job</span>
        <span class="param-flags">required</span>
        The JSON definition of the job. The general structure is given
        by the [job specification](/docs/jobspec/index.html), and matches
        the return response of GET.
      </li>
      <li>
        <span class="param">Diff</span>
        <span class="param-flags">optional</span>
        Whether the diff structure between the submitted and server side version
        of the job should be included in the response.
      </li>
    </ul>
  </dd>

  <dt>Returns</dt>
  <dd>

    ```javascript
	{
	  "Index": 0,
	  "NextPeriodicLaunch": "0001-01-01T00:00:00Z",
	  "Diff": {
		"Type": "Added",
		"TaskGroups": [
		  {
			"Updates": {
			  "create": 1
			},
			"Type": "Added",
			"Tasks": [
			  {
				"Type": "Added",
				"Objects": [...],
				"Name": "redis",
				"Fields": [
				  {
					"Type": "Added",
					"Old": "",
					"New": "docker",
					"Name": "Driver",
					"Annotations": null
				  },
				  {
					"Type": "Added",
					"Old": "",
					"New": "5000000000",
					"Name": "KillTimeout",
					"Annotations": null
				  }
				],
				"Annotations": [
				  "forces create"
				]
			  }
			],
			"Objects": [...],
			"Name": "cache",
			"Fields": [...]
		  }
		],
		"Objects": [
		  {
			"Type": "Added",
			"Objects": null,
			"Name": "Datacenters",
			"Fields": [...]
		  },
		  {
			"Type": "Added",
			"Objects": null,
			"Name": "Constraint",
			"Fields": [...]
		  },
		  {
			"Type": "Added",
			"Objects": null,
			"Name": "Update",
			"Fields": [...]
		  }
		],
		"ID": "example",
        "Fields": [...],
		  ...
		]
	  },
	"CreatedEvals": [
		{
		  "ModifyIndex": 0,
		  "CreateIndex": 0,
		  "SnapshotIndex": 0,
		  "AnnotatePlan": false,
		  "EscapedComputedClass": false,
		  "NodeModifyIndex": 0,
		  "NodeID": "",
		  "JobModifyIndex": 0,
		  "JobID": "example",
		  "TriggeredBy": "job-register",
		  "Type": "batch",
		  "Priority": 50,
		  "ID": "312e6a6d-8d01-0daf-9105-14919a66dba3",
		  "Status": "blocked",
		  "StatusDescription": "created to place remaining allocations",
		  "Wait": 0,
		  "NextEval": "",
		  "PreviousEval": "80318ae4-7eda-e570-e59d-bc11df134817",
		  "BlockedEval": "",
		  "FailedTGAllocs": null,
		  "ClassEligibility": {
			"v1:7968290453076422024": true
		  }
		}
	  ],
	  "JobModifyIndex": 0,
	  "FailedTGAllocs": {
		"cache": {
		  "CoalescedFailures": 3,
		  "AllocationTime": 46415,
		  "Scores": null,
		  "NodesEvaluated": 1,
		  "NodesFiltered": 0,
		  "NodesAvailable": {
			"dc1": 1
		  },
		  "ClassFiltered": null,
		  "ConstraintFiltered": null,
		  "NodesExhausted": 1,
		  "ClassExhausted": null,
		  "DimensionExhausted": {
			"cpu exhausted": 1
		  }
		}
	  },
	  "Annotations": {
		"DesiredTGUpdates": {
		  "cache": {
			"DestructiveUpdate": 0,
			"InPlaceUpdate": 0,
			"Stop": 0,
			"Migrate": 0,
			"Place": 11,
			"Ignore": 0
		  }
		}
	  }
	}
    ```

  </dd>

  <dt>Field Reference</dt>
  <dd>
    <ul>
      <li>
        <span class="param">Diff</span>
        A diff structure between the submitted job and the server side version.
        The top-level object is a Job Diff which contains, Task Group Diffs
        which in turn contain Task Diffs. Each of these objects then has Object
        and Field Diff structures in-bedded.
      </li>
      <li>
        <span class="param">NextPeriodicLaunch</span>
        If the job being planned is periodic, this field will include the next
        launch time for the job.
      </li>
      <li>
        <span class="param">CreatedEvals</span>
        A set of evaluations that were created as a result of the dry-run. These
        evaluations can signify a follow-up rolling update evaluation or a
        blocked evaluation.
      </li>
      <li>
        <span class="param">JobModifyIndex</span>
        The JobModifyIndex of the server side version of this job.
      </li>
      <li>
        <span class="param">FailedTGAllocs</span>
        A set of metrics to understand any allocation failures that occurred for
        the Task Group.
      </li>
      <li>
        <span class="param">Annotations</span>
        Annotations include the DesiredTGUpdates, which tracks what the
        scheduler would do given enough resources for each Task Group.
      </li>
    </ul>
  </dd>
</dl>


<dl>
  <dt>Description</dt>
  <dd>
    Forces a new instance of the periodic job. A new instance will be created
    even if it violates the job's
    [`prohibit_overlap`](/docs/jobspec/index.html#prohibit_overlap) settings. As
    such, this should be only used to immediately run a periodic job.
  </dd>

  <dt>Method</dt>
  <dd>PUT or POST</dd>

  <dt>URL</dt>
  <dd>`/v1/job/<ID>/periodic/force`</dd>

  <dt>Parameters</dt>
  <dd>
    None
  </dd>

  <dt>Returns</dt>
  <dd>

    ```javascript
    {
    "EvalCreateIndex": 7,
    "EvalID": "57983ddd-7fcf-3e3a-fd24-f699ccfb36f4"
    }
    ```

  </dd>
</dl>

## DELETE

<dl>
  <dt>Description</dt>
  <dd>
    Deregisters a job, and stops all allocations part of it.
  </dd>

  <dt>Method</dt>
  <dd>DELETE</dd>

  <dt>URL</dt>
  <dd>`/v1/job/<ID>`</dd>

  <dt>Parameters</dt>
  <dd>
    None
  </dd>

  <dt>Returns</dt>
  <dd>

    ```javascript
    {
    "EvalID": "d092fdc0-e1fd-2536-67d8-43af8ca798ac",
    "EvalCreateIndex": 35,
    "JobModifyIndex": 34,
    }
    ```

  </dd>
</dl>
