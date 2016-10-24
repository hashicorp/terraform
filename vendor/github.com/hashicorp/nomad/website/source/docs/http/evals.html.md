---
layout: "http"
page_title: "HTTP API: /v1/evaluations"
sidebar_current: "docs-http-evals"
description: |-
  The '/1/evaluations' endpoint is used to list the evaluations.
---

# /v1/evaluations

The `evaluations` endpoint is used to query the status of evaluations.
By default, the agent's local region is used; another region can
be specified using the `?region=` query parameter.

## GET

<dl>
  <dt>Description</dt>
  <dd>
    Lists all the evaluations.
  </dd>

  <dt>Method</dt>
  <dd>GET</dd>

  <dt>URL</dt>
  <dd>`/v1/evaluations`</dd>

  <dt>Parameters</dt>
  <dd>
    <ul>
      <li>
        <span class="param">prefix</span>
        <span class="param-flags">optional</span>
        <span class="param-flags">even-length</span>
        Filter evaluations based on an identifier prefix.
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
