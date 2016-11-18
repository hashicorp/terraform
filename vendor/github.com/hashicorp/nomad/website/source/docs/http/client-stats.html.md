---
layout: "http"
page_title: "HTTP API: /v1/client/stats"
sidebar_current: "docs-http-client-stats"
description: |-
  The '/v1/client/stats` endpoint is used to query the actual resources consumed
  on the node.
---

# /v1/client/stats

The client `stats` endpoint is used to query the actual resources consumed on a node.
The API endpoint is hosted by the Nomad client and requests have to be made to
the nomad client whose resource usage metrics are of interest.

## GET

<dl>
  <dt>Description</dt>
  <dd>
     Query the actual resource usage of a Nomad client
  </dd>

  <dt>Method</dt>
  <dd>GET</dd>

  <dt>URL</dt>
  <dd>`/v1/client/stats`</dd>

  <dt>Returns</dt>
  <dd>

  ```javascript
   {
     "CPU": [
       {
         "CPU": "cpu0",
         "Idle": 89.2156862745098,
         "System": 4.901960784313726,
         "Total": 10.784313725490197,
         "User": 5.88235294117647
       },
       {
         "CPU": "cpu1",
         "Idle": 100,
         "System": 0,
         "Total": 0,
         "User": 0
       },
       {
         "CPU": "cpu2",
         "Idle": 94.05940594059405,
         "System": 2.9702970297029703,
         "Total": 5.9405940594059405,
         "User": 2.9702970297029703
       },
       {
         "CPU": "cpu3",
         "Idle": 99.00990099009901,
         "System": 0,
         "Total": 0.9900990099009901,
         "User": 0.9900990099009901
       }
     ],
     "CPUTicksConsumed": 119.5762958648806,
     "DiskStats": [
       {
         "Available": 16997969920,
         "Device": "/dev/disk1",
         "InodesUsedPercent": 85.84777164286838,
         "Mountpoint": "/",
         "Size": 120108089344,
         "Used": 102847975424,
         "UsedPercent": 85.62951586835626
       }
     ],
     "Memory": {
       "Available": 3724746752,
       "Free": 2446233600,
       "Total": 8589934592,
       "Used": 4865187840
     },
     "Timestamp": 1465839167993064200,
     "Uptime": 101149
   }
  ```
  </dd>
</dl>
