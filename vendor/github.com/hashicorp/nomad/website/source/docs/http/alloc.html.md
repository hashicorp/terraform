---
layout: "http"
page_title: "HTTP API: /v1/allocation"
sidebar_current: "docs-http-alloc-"
description: |-
  The '/1/allocation' endpoint is used to query a specific allocation.
---

# /v1/allocation

The `allocation` endpoint is used to query the a specific allocation.
By default, the agent's local region is used; another region can
be specified using the `?region=` query parameter.

## GET

<dl>
  <dt>Description</dt>
  <dd>
    Query a specific allocation.
  </dd>

  <dt>Method</dt>
  <dd>GET</dd>

  <dt>URL</dt>
  <dd>`/v1/allocation/<ID>`</dd>

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
      "ID": "203266e5-e0d6-9486-5e05-397ed2b184af",
      "EvalID": "e68125ed-3fba-fb46-46cc-291addbc4455",
      "Name": "example.cache[0]",
      "NodeID": "e02b6169-83bd-9df6-69bd-832765f333eb",
      "JobID": "example",
      "ModifyIndex": 9,
      "Resources": {
        "Networks": [
          {
            "DynamicPorts": [
              {
                "Value": 20802,
                "Label": "db"
              }
            ],
            "ReservedPorts": null,
            "MBits": 10,
            "IP": "",
            "CIDR": "",
            "Device": ""
          }
        ],
        "IOPS": 0,
        "DiskMB": 0,
        "MemoryMB": 256,
        "CPU": 500
      },
      "TaskGroup": "cache",
      "Job": {
        "ModifyIndex": 5,
        "CreateIndex": 5,
        "StatusDescription": "",
        "Status": "",
        "Meta": null,
        "Update": {
          "MaxParallel": 1,
          "Stagger": 1e+10
        },
        "TaskGroups": [
          {
            "Meta": null,
            "Tasks": [
              {
                "Meta": null,
                "Resources": {
                  "Networks": [
                    {
                      "DynamicPorts": [
                        {
                          "Value": 20802,
                          "Label": "db"
                        }
                      ],
                      "ReservedPorts": null,
                      "MBits": 0,
                      "IP": "127.0.0.1",
                      "CIDR": "",
                      "Device": "lo"
                    }
                  ],
                  "IOPS": 0,
                  "DiskMB": 0,
                  "MemoryMB": 256,
                  "CPU": 500
                },
                "Constraints": null,
                "Services": [
                  {
                    "Checks": [
                      {
                        "Timeout": 2e+09,
                        "Interval": 1e+10,
                        "Protocol": "",
                        "Http": "",
                        "Script": "",
                        "Type": "tcp",
                        "Name": "alive",
                        "Id": ""
                      }
                    ],
                    "PortLabel": "db",
                    "Tags": [
                      "global",
                      "cache"
                    ],
                    "Name": "example-cache-redis",
                    "Id": ""
                  }
                ],
                "Env": null,
                "Config": {
                  "port_map": [
                    {
                      "db": 6379
                    }
                  ],
                  "image": "redis:latest"
                },
                "Driver": "docker",
                "Name": "redis"
              }
            ],
            "RestartPolicy": {
              "Delay": 2.5e+10,
              "Interval": 3e+11,
              "Attempts": 10
            },
            "Constraints": null,
            "Count": 1,
            "Name": "cache"
          }
        ],
        "Region": "global",
        "ID": "example",
        "Name": "example",
        "Type": "service",
        "Priority": 50,
        "AllAtOnce": false,
        "Datacenters": [
          "dc1"
        ],
        "Constraints": [
          {
            "Operand": "=",
            "RTarget": "linux",
            "LTarget": "${attr.kernel.name}"
          }
        ]
      },
      "TaskResources": {
        "redis": {
          "Networks": [
            {
              "DynamicPorts": [
                {
                  "Value": 20802,
                  "Label": "db"
                }
              ],
              "ReservedPorts": null,
              "MBits": 0,
              "IP": "127.0.0.1",
              "CIDR": "",
              "Device": "lo"
            }
          ],
          "IOPS": 0,
          "DiskMB": 0,
          "MemoryMB": 256,
          "CPU": 500
        }
      },
      "Metrics": {
        "CoalescedFailures": 0,
        "AllocationTime": 1590406,
        "NodesEvaluated": 1,
        "NodesFiltered": 0,
        "ClassFiltered": null,
        "ConstraintFiltered": null,
        "NodesExhausted": 0,
        "ClassExhausted": null,
        "DimensionExhausted": null,
        "Scores": {
          "e02b6169-83bd-9df6-69bd-832765f333eb.binpack": 6.133651487695705
        }
      },
      "DesiredStatus": "run",
      "DesiredDescription": "",
      "ClientStatus": "running",
      "ClientDescription": "",
      "TaskStates": {
        "redis": {
          "Events": [
            {
              "KillError": "",
              "Message": "",
              "Signal": 0,
              "ExitCode": 0,
              "DriverError": "",
              "Time": 1447806038427841000,
              "Type": "Started"
            }
          ],
          "State": "running"
        }
      },
      "CreateIndex": 7
    }
    ```

  </dd>
</dl>

### Field Reference

*   `TaskStates` - `TaskStates` is a map of tasks to their current state and the
    latest events that have effected the state.

    A task can be in the following states:

    * `TaskStatePending` - The task is waiting to be run, either for the first
      time or due to a restart.
    * `TaskStateRunning` - The task is currently running.
    * `TaskStateDead` - The task is dead and will not run again.

    <p>The latest 10 events are stored per task. Each event is timestamped (unix nano-seconds)
    and has one of the following types:</p>

    * `Driver Failure` - The task could not be started due to a failure in the
      driver.
    * `Started` - The task was started; either for the first time or due to a
      restart.
    * `Terminated` - The task was started and exited.
    * `Killing` - The task has been sent the kill signal.
    * `Killed` - The task was killed by an user.
    * `Received` - The task has been pulled by the client at the given timestamp.
    * `Failed Validation` - The task was invalid and as such it didn't run.
    * `Restarting` - The task terminated and is being restarted.
    * `Not Restarting` - the task has failed and is not being restarted because it has exceeded its restart policy.
    * `Downloading Artifacts` - The task is downloading the artifact(s) specified in the task. 
    * `Failed Artifact Download` - Artifact(s) specified in the task failed to download.

    Depending on the type the event will have applicable annotations.
