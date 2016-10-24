---
layout: "http"
page_title: "HTTP API: /v1/status/"
sidebar_current: "docs-http-status"
description: |-
  The '/1/status/' endpoints are used to query the system status.
---

# /v1/status/leader

By default, the agent's local region is used; another region can
be specified using the `?region=` query parameter.

## GET

<dl>
  <dt>Description</dt>
  <dd>
    Returns the address of the current leader in the region.
  </dd>

  <dt>Method</dt>
  <dd>GET</dd>

  <dt>URL</dt>
  <dd>`/v1/status/leader`</dd>

  <dt>Parameters</dt>
  <dd>
    None
  </dd>

  <dt>Returns</dt>
  <dd>

    ```javascript
    "127.0.0.1:4647"
    ```

  </dd>
</dl>

# /v1/status/peers

## GET

<dl>
  <dt>Description</dt>
  <dd>
    Returns the set of raft peers in the region.
  </dd>

  <dt>Method</dt>
  <dd>GET</dd>

  <dt>URL</dt>
  <dd>`/v1/status/peers`</dd>

  <dt>Parameters</dt>
  <dd>
    None
  </dd>

  <dt>Returns</dt>
  <dd>

    ```javascript
    [
    "127.0.0.1:4647",
    ...
    ]
    ```

  </dd>
</dl>


