---
layout: "http"
page_title: "HTTP API: /v1/agent/servers"
sidebar_current: "docs-http-agent-servers"
description: |-
  The '/v1/agent/servers' endpoint is used to query and update the servers list.
---

# /v1/agent/servers

The `servers` endpoint is used to query an agent in client mode for its list
of known servers. Client nodes register themselves with these server addresses
so that they may dequeue work. The `servers` endpoint can be used to keep this
configuration up to date if there are changes in the cluster.

## GET

<dl>
  <dt>Description</dt>
  <dd>
    Lists the known server nodes.
  </dd>

  <dt>Method</dt>
  <dd>GET</dd>

  <dt>URL</dt>
  <dd>`/v1/agent/servers`</dd>

  <dt>Parameters</dt>
  <dd>
    None
  </dd>

  <dt>Returns</dt>
  <dd>

    ```javascript
    [
      "server1.local:4647",
      "server2.local:4647"
    ]
    ```

  </dd>
</dl>

## PUT / POST

<dl>
  <dt>Description</dt>
  <dd>
    Updates the list of known servers to the provided list. Replaces
    all previous server addresses with the new list.
  </dd>

  <dt>Method</dt>
  <dd>PUT or POST</dd>

  <dt>URL</dt>
  <dd>`/v1/agent/servers`</dd>

  <dt>Parameters</dt>
  <dd>
    <ul>
      <li>
        <span class="param">address</span>
        <span class="param-flags">required</span>
        The address of a server node in host:port format. This
        parameter may be specified multiple times to configure
        multiple servers on the client.
      </li>
    </ul>
  </dd>

  <dt>Returns</dt>
  <dd>
    A 200 status code on success.
  </dd>
</dl>


