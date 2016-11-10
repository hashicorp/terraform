---
layout: "http"
page_title: "HTTP API: /v1/agent/force-leave"
sidebar_current: "docs-http-agent-force-leave"
description: |-
  The '/1/agent/force-leave' endpoint is force a gossip member to leave.
---

# /v1/agent/force-leave

The `force-leave` endpoint is used to force a member of the gossip pool from
the "failed" state into the "left" state. This allows the consensus protocol to
remove the peer and stop attempting replication. This is only applicable for
servers.

## PUT / POST

<dl>
  <dt>Description</dt>
  <dd>
    Force a failed gossip member into the left state.
  </dd>

  <dt>Method</dt>
  <dd>PUT or POST</dd>

  <dt>URL</dt>
  <dd>`/v1/agent/force-leave`</dd>

  <dt>Parameters</dt>
  <dd>
    <ul>
      <li>
        <span class="param">node</span>
        <span class="param-flags">required</span>
        The name of the node to force leave.
      </li>
    </ul>
  </dd>

  <dt>Returns</dt>
  <dd>

    A `200` status code on success.
  </dd>
</dl>

