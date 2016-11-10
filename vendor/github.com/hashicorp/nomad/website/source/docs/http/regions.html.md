---
layout: "http"
page_title: "HTTP API: /v1/regions"
sidebar_current: "docs-http-regions"
description: >
  The '/v1/regions' endpoint lists the known cluster regions.
---

# /v1/regions

## GET

<dl>
  <dt>Description</dt>
  <dd>
    Returns the known region names.
  </dd>

  <dt>Method</dt>
  <dd>GET</dd>

  <dt>URL</dt>
  <dd>`/v1/regions`</dd>

  <dt>Parameters</dt>
  <dd>
    None
  </dd>

  <dt>Returns</dt>
  <dd>

    ```javascript
    ["region1","region2"]
    ```

  </dd>
</dl>
