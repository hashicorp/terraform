---
layout: "http"
page_title: "HTTP API: /v1/evaluation"
sidebar_current: "docs-http-eval-"
description: |-
  The '/v1/evaluation' endpoint is used to query a specific evaluation.
---

# /v1/evaluation

The `evaluation` endpoint is used to query a specific evaluations.
By default, the agent's local region is used; another region can
be specified using the `?region=` query parameter.

## GET

<dl>
  <dt>Description</dt>
  <dd>
    Query a specific evaluation.
  </dd>

  <dt>Method</dt>
  <dd>GET</dd>

  <dt>URL</dt>
  <dd>`/v1/evaluation/<ID>`</dd>

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
    }
    ```

  </dd>
</dl>

<dl>
  <dt>Description</dt>
  <dd>
    Query the allocations created or modified by an evaluation.
  </dd>

  <dt>Method</dt>
  <dd>GET</dd>

  <dt>URL</dt>
  <dd>`/v1/evaluation/<ID>/allocations`</dd>

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
