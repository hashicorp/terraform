---
layout: "http"
page_title: "HTTP API: /v1/agent/join"
sidebar_current: "docs-http-agent-join"
description: |-
  The '/1/agent/join' endpoint is used to cluster the Nomad servers.
---

# /v1/agent/join

The `join` endpoint is used to cluster the Nomad servers using a gossip pool.
The servers participate in a peer-to-peer gossip, and `join` is used to introduce
a member to the pool. This is only applicable for servers.

## PUT / POST

<dl>
  <dt>Description</dt>
  <dd>
    Initiate a join between the agent and target peers.
  </dd>

  <dt>Method</dt>
  <dd>PUT or POST</dd>

  <dt>URL</dt>
  <dd>`/v1/agent/join`</dd>

  <dt>Parameters</dt>
  <dd>
    <ul>
      <li>
        <span class="param">address</span>
        <span class="param-flags">required</span>
        The address to join. Can be provided multiple times
        to attempt joining multiple peers.
      </li>
    </ul>
  </dd>

  <dt>Returns</dt>
  <dd>

    ```javascript
    {
    "num_joined": 1,
    "error": ""
    }
    ```

  </dd>
</dl>

