---
layout: "http"
page_title: "HTTP API: /v1/nodes"
sidebar_current: "docs-http-nodes"
description: |-
  The '/1/nodes' endpoint is used to list the client nodes.
---

# /v1/nodes

The `nodes` endpoint is used to query the status of client nodes.
By default, the agent's local region is used; another region can
be specified using the `?region=` query parameter.

## GET

<dl>
  <dt>Description</dt>
  <dd>
    Lists all the client nodes registered with Nomad.
  </dd>

  <dt>Method</dt>
  <dd>GET</dd>

  <dt>URL</dt>
  <dd>`/v1/nodes`</dd>

  <dt>Parameters</dt>
  <dd>
    <ul>
      <li>
        <span class="param">prefix</span>
        <span class="param-flags">optional</span>
        Filter nodes based on an identifier prefix.
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
        "ID": "c9972143-861d-46e6-df73-1d8287bc3e66",
        "Datacenter": "dc1",
        "Name": "web-8e40e308",
        "NodeClass": "",
        "Drain": false,
        "Status": "ready",
        "StatusDescription": "",
        "CreateIndex": 3,
        "ModifyIndex": 4
    },
    ...
    ]
    ```

  </dd>
</dl>
