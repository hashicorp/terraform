---
layout: "http"
page_title: "HTTP API: /v1/allocations"
sidebar_current: "docs-http-allocs"
description: |-
  The '/1/allocations' endpoint is used to list the allocations.
---

# /v1/allocations

The `allocations` endpoint is used to query the status of allocations.
By default, the agent's local region is used; another region can
be specified using the `?region=` query parameter.

## GET

<dl>
  <dt>Description</dt>
  <dd>
    Lists all the allocations.
  </dd>

  <dt>Method</dt>
  <dd>GET</dd>

  <dt>URL</dt>
  <dd>`/v1/allocations`</dd>

  <dt>Parameters</dt>
  <dd>
    <ul>
      <li>
        <span class="param">prefix</span>
        <span class="param-flags">optional</span>
        <span class="param-flags">even-length</span>
        Filter allocations based on an identifier prefix.
      </li>
    </ul>
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
      "TaskGroup": "cache",
      "DesiredStatus": "run",
      "DesiredDescription": ""
      "ClientDescription": "",
      "ClientStatus": "running",
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
      "CreateIndex": 7,
      "ModifyIndex": 9,
    }
    ...
    ]
    ```

  </dd>
</dl>
