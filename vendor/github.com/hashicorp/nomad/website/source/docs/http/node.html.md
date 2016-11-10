---
layout: "http"
page_title: "HTTP API: /v1/node"
sidebar_current: "docs-http-node-"
description: |-
  The '/1/node-' endpoint is used to query a specific client node.
---

# /v1/node

The `node` endpoint is used to query the a specific client node.
By default, the agent's local region is used; another region can
be specified using the `?region=` query parameter.

## GET

<dl>
  <dt>Description</dt>
  <dd>
    Query the status of a client node registered with Nomad.
  </dd>

  <dt>Method</dt>
  <dd>GET</dd>

  <dt>URL</dt>
  <dd>`/v1/node/<ID>`</dd>

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
    "ID": "c9972143-861d-46e6-df73-1d8287bc3e66",
    "Datacenter": "dc1",
    "Name": "Armons-MacBook-Air.local",
    "Attributes": {
        "arch": "amd64",
        "cpu.frequency": "1300.000000",
        "cpu.modelname": "Intel(R) Core(TM) i5-4250U CPU @ 1.30GHz",
        "cpu.numcores": "2",
        "cpu.totalcompute": "2600.000000",
        "driver.exec": "1",
        "driver.java": "1",
        "driver.java.runtime": "Java(TM) SE Runtime Environment (build 1.8.0_05-b13)",
        "driver.java.version": "1.8.0_05",
        "driver.java.vm": "Java HotSpot(TM) 64-Bit Server VM (build 25.5-b02, mixed mode)",
        "hostname": "Armons-MacBook-Air.local",
        "kernel.name": "darwin",
        "kernel.version": "14.4.0",
        "memory.totalbytes": "8589934592",
        "network.ip-address": "127.0.0.1",
        "os.name": "darwin",
        "os.version": "14.4.0",
        "storage.bytesfree": "35888713728",
        "storage.bytestotal": "249821659136",
        "storage.volume": "/dev/disk1"
    },
    "Resources": {
        "CPU": 2600,
        "MemoryMB": 8192,
        "DiskMB": 34226,
        "IOPS": 0,
        "Networks": null
    },
    "Reserved": null,
    "Links": {},
    "Meta": {},
    "NodeClass": "",
    "Drain": false,
    "Status": "ready",
    "StatusDescription": "",
    "CreateIndex": 3,
    "ModifyIndex": 4
    }
    ```

  </dd>
</dl>

<dl>
  <dt>Description</dt>
  <dd>
    Query the allocations belonging to a single node.
  </dd>

  <dt>Method</dt>
  <dd>GET</dd>

  <dt>URL</dt>
  <dd>`/v1/node/<ID>/allocations`</dd>

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
    },
    ...
    ]
    ```

  </dd>
</dl>

## PUT / POST

<dl>
  <dt>Description</dt>
  <dd>
    Creates a new evaluation for the given node. This can be used to force
    run the scheduling logic if necessary.
  </dd>

  <dt>Method</dt>
  <dd>PUT or POST</dd>

  <dt>URL</dt>
  <dd>`/v1/node/<ID>/evaluate`</dd>

  <dt>Parameters</dt>
  <dd>
    None
  </dd>

  <dt>Returns</dt>
  <dd>

    ```javascript
    {
    "EvalIDs": ["d092fdc0-e1fd-2536-67d8-43af8ca798ac"],
    "EvalCreateIndex": 35,
    "NodeModifyIndex": 34
    }
    ```

  </dd>
</dl>

<dl>
  <dt>Description</dt>
  <dd>
    Toggle the drain mode of the node. When enabled, no further
    allocations will be assigned and existing allocations will be
    migrated.
  </dd>

  <dt>Method</dt>
  <dd>PUT or POST</dd>

  <dt>URL</dt>
  <dd>`/v1/node/<ID>/drain`</dd>

  <dt>Parameters</dt>
  <dd>
    <ul>
      <li>
        <span class="param">enable</span>
        <span class="param-flags">required</span>
        Boolean value provided as a query parameter to either set
        enabled to true or false.
      </li>
    </ul>
  </dd>

  <dt>Returns</dt>
  <dd>

    ```javascript
    {
    "EvalID": "d092fdc0-e1fd-2536-67d8-43af8ca798ac",
    "EvalCreateIndex": 35,
    "NodeModifyIndex": 34
    }
    ```

  </dd>
</dl>
