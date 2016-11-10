---
layout: "http"
page_title: "HTTP API: /v1/agent/self"
sidebar_current: "docs-http-agent-self"
description: |-
  The '/1/agent/self' endpoint is used to query the state of the agent.
---

# /v1/agent/self

The `self` endpoint is used to query the state of the target agent.

## GET

<dl>
  <dt>Description</dt>
  <dd>
    Query the state of the target agent.
  </dd>

  <dt>Method</dt>
  <dd>GET</dd>

  <dt>URL</dt>
  <dd>`/v1/agent/self`</dd>

  <dt>Parameters</dt>
  <dd>
    None
  </dd>

  <dt>Returns</dt>
  <dd>

    ```javascript
    {
    "config": {
        "Region": "global",
        "Datacenter": "dc1",
        "NodeName": "",
        "DataDir": "",
        "LogLevel": "DEBUG",
        "BindAddr": "127.0.0.1",
        "EnableDebug": true,
        "Ports": {
            "HTTP": 4646,
            "RPC": 4647,
            "Serf": 4648
        },
        "Addresses": {
            "HTTP": "",
            "RPC": "",
            "Serf": ""
        },
        "AdvertiseAddrs": {
            "RPC": "",
            "Serf": ""
        },
        "Client": {
            "Enabled": true,
            "StateDir": "",
            "AllocDir": "",
            "Servers": null,
            "NodeID": "",
            "NodeClass": "",
            "Meta": null
        },
        "Server": {
            "Enabled": true,
            "Bootstrap": false,
            "BootstrapExpect": 0,
            "DataDir": "",
            "ProtocolVersion": 0,
            "NumSchedulers": 0,
            "EnabledSchedulers": null
        },
        "Telemetry": null,
        "LeaveOnInt": false,
        "LeaveOnTerm": false,
        "EnableSyslog": false,
        "SyslogFacility": "",
        "DisableUpdateCheck": false,
        "DisableAnonymousSignature": true,
        "Revision": "",
        "Version": "0.1.0",
        "VersionPrerelease": "dev",
        "DevMode": true,
        "Atlas": null
    },
    "member": {
        "Name": "Armons-MacBook-Air.local.global",
        "Addr": "127.0.0.1",
        "Port": 4648,
        "Tags": {
            "bootstrap": "1",
            "build": "0.1.0dev",
            "dc": "dc1",
            "port": "4647",
            "region": "global",
            "role": "nomad",
            "vsn": "1"
        },
        "Status": "alive",
        "ProtocolMin": 1,
        "ProtocolMax": 3,
        "ProtocolCur": 2,
        "DelegateMin": 2,
        "DelegateMax": 4,
        "DelegateCur": 4
    },
    "stats": {
        "client": {
            "heartbeat_ttl": "19116443712",
            "known_servers": "0",
            "last_heartbeat": "8222075779",
            "num_allocations": "0"
        },
        "nomad": {
            "bootstrap": "false",
            "known_regions": "1",
            "leader": "true",
            "server": "true"
        },
        "raft": {
            "applied_index": "91",
            "commit_index": "91",
            "fsm_pending": "0",
            "last_contact": "never",
            "last_log_index": "91",
            "last_log_term": "1",
            "last_snapshot_index": "0",
            "last_snapshot_term": "0",
            "num_peers": "0",
            "state": "Leader",
            "term": "1"
        },
        "runtime": {
            "arch": "amd64",
            "cpu_count": "4",
            "goroutines": "58",
            "kernel.name": "darwin",
            "max_procs": "1",
            "version": "go1.4.2"
        },
        "serf": {
            "encrypted": "false",
            "event_queue": "0",
            "event_time": "1",
            "failed": "0",
            "intent_queue": "0",
            "left": "0",
            "member_time": "1",
            "members": "1",
            "query_queue": "0",
            "query_time": "1"
        }
    }
    }
    ```

  </dd>
</dl>

