---
layout: "http"
page_title: "HTTP API: /v1/jobs"
sidebar_current: "docs-http-jobs"
description: |-
  The '/1/jobs' endpoint is used list jobs and register new ones.
---

# /v1/jobs

The `jobs` endpoint is used to query the status of existing jobs in Nomad
and to register new jobs. By default, the agent's local region is used;
another region can be specified using the `?region=` query parameter.

## GET

<dl>
  <dt>Description</dt>
  <dd>
    Lists all the jobs registered with Nomad.
  </dd>

  <dt>Method</dt>
  <dd>GET</dd>

  <dt>URL</dt>
  <dd>`/v1/jobs`</dd>

  <dt>Parameters</dt>
  <dd>
    <ul>
      <li>
        <span class="param">prefix</span>
        <span class="param-flags">optional</span>
        Filter jobs based on an identifier prefix.
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
        "ID": "binstore-storagelocker",
        "Name": "binstore-storagelocker",
        "Type": "service",
        "Priority": 50,
        "Status": "",
        "StatusDescription": "",
        "CreateIndex": 14,
        "ModifyIndex": 14
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
    Registers a new job.
  </dd>

  <dt>Method</dt>
  <dd>PUT or POST</dd>

  <dt>URL</dt>
  <dd>`/v1/jobs`</dd>

  <dt>Parameters</dt>
  <dd>
    <ul>
      <li>
        <span class="param">Job</span>
        <span class="param-flags">required</span>
        The JSON definition of the job. The general structure is given
        by the [job specification](/docs/jobspec/json.html).
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
