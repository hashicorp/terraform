---
layout: "http"
page_title: "HTTP API: /v1/client/allocation/stats"
sidebar_current: "docs-http-client-allocation-stats"
description: |-
  The '/v1/client/allocation/` endpoint is used to query the actual resources
  consumed by an allocation.
---

# /v1/client/allocation

The client `allocation` endpoint is used to query the actual resources consumed
by an allocation.  The API endpoint is hosted by the Nomad client and requests
have to be made to the nomad client whose resource usage metrics are of
interest.

## GET

<dl>
  <dt>Description</dt>
  <dd>
     Query resource usage of an allocation running on a client.
  </dd>

  <dt>Method</dt>
  <dd>GET</dd>

  <dt>URL</dt>
  <dd>`/v1/client/allocation/<ID>/stats`</dd>

  <dt>Returns</dt>
  <dd>

  ```javascript
    {
      "ResourceUsage": {
        "CpuStats": {
          "Measured": [
            "System Mode",
            "User Mode",
            "Percent"
          ],
          "Percent": 105.77854560628487,
          "SystemMode": 6.860067935411291,
          "ThrottledPeriods": 0,
          "ThrottledTime": 0,
          "TotalTicks": 714.0051828424228,
          "UserMode": 98.9184820888787
        },
        "MemoryStats": {
          "Cache": 0,
          "KernelMaxUsage": 0,
          "KernelUsage": 0,
          "MaxUsage": 0,
          "Measured": [
            "RSS",
            "Swap"
          ],
          "RSS": 14098432,
          "Swap": 0
        }
      },
      "Tasks": {
        "redis": {
          "Pids": {
            "27072": {
              "CpuStats": {
                "Measured": [
                  "System Mode",
                  "User Mode",
                  "Percent"
                ],
                "Percent": 6.8607999603563385,
                "SystemMode": 5.880684245133524,
                "ThrottledPeriods": 0,
                "ThrottledTime": 0,
                "TotalTicks": 0,
                "UserMode": 0.9801144039714172
              },
              "MemoryStats": {
                "Cache": 0,
                "KernelMaxUsage": 0,
                "KernelUsage": 0,
                "MaxUsage": 0,
                "Measured": [
                  "RSS",
                  "Swap"
                ],
                "RSS": 13418496,
                "Swap": 0
              }
            },
            "27073": {
              "CpuStats": {
                "Measured": [
                  "System Mode",
                  "User Mode",
                  "Percent"
                ],
                "Percent": 98.91774564592852,
                "SystemMode": 0.9793836902777665,
                "ThrottledPeriods": 0,
                "ThrottledTime": 0,
                "TotalTicks": 0,
                "UserMode": 97.93836768490729
              },
              "MemoryStats": {
                "Cache": 0,
                "KernelMaxUsage": 0,
                "KernelUsage": 0,
                "MaxUsage": 0,
                "Measured": [
                  "RSS",
                  "Swap"
                ],
                "RSS": 679936,
                "Swap": 0
              }
            }
          },
          "ResourceUsage": {
            "CpuStats": {
              "Measured": [
                "System Mode",
                "User Mode",
                "Percent"
              ],
              "Percent": 105.77854560628487,
              "SystemMode": 6.860067935411291,
              "ThrottledPeriods": 0,
              "ThrottledTime": 0,
              "TotalTicks": 714.0051828424228,
              "UserMode": 98.9184820888787
            },
            "MemoryStats": {
              "Cache": 0,
              "KernelMaxUsage": 0,
              "KernelUsage": 0,
              "MaxUsage": 0,
              "Measured": [
                "RSS",
                "Swap"
              ],
              "RSS": 14098432,
              "Swap": 0
            }
          },
          "Timestamp": 1465865820750959600
        }
      },
      "Timestamp": 1465865820750959600
    }
  ```
  </dd>
</dl>
