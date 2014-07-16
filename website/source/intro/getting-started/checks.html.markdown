---
layout: "intro"
page_title: "Registering Health Checks"
sidebar_current: "gettingstarted-checks"
---

# Health Checks

We've now seen how simple it is to run Terraform, add nodes and services, and
query those nodes and services. In this section we will continue by adding
health checks to both nodes and services, a critical component of service
discovery that prevents using services that are unhealthy.

This page will build upon the previous page and assumes you have a
two node cluster running.

## Defining Checks

Similarly to a service, a check can be registered either by providing a
[check definition](/docs/agent/checks.html)
, or by making the appropriate calls to the
[HTTP API](/docs/agent/http.html).

We will use the check definition, because just like services, definitions
are the most common way to setup checks.

Create two definition files in the Terraform configuration directory of
the second node.
The first file will add a host-level check, and the second will modify the web
service definition to add a service-level check.

```
$ echo '{"check": {"name": "ping", "script": "ping -c1 google.com >/dev/null", "interval": "30s"}}' >/etc/terraform.d/ping.json

$ echo '{"service": {"name": "web", "tags": ["rails"], "port": 80,
  "check": {"script": "curl localhost:80 >/dev/null 2>&1", "interval": "10s"}}}' >/etc/terraform.d/web.json
```

The first definition adds a host-level check named "ping". This check runs
on a 30 second interval, invoking `ping -c1 google.com`. If the command
exits with a non-zero exit code, then the node will be flagged unhealthy.

The second command modifies the web service and adds a check that uses
curl every 10 seconds to verify that the web server is running.

Restart the second agent, or send a `SIGHUP` to it. We should now see the
following log lines:

```
==> Starting Terraform agent...
...
    [INFO] agent: Synced service 'web'
    [INFO] agent: Synced check 'service:web'
    [INFO] agent: Synced check 'ping'
    [WARN] Check 'service:web' is now critical
```

The first few log lines indicate that the agent has synced the new
definitions. The last line indicates that the check we added for
the `web` service is critical. This is because we're not actually running
a web server and the curl test is failing!

## Checking Health Status

Now that we've added some simple checks, we can use the HTTP API to check
them. First, we can look for any failing checks. You can run this curl
on either node:

```
$ curl http://localhost:8500/v1/health/state/critical
[{"Node":"agent-two","CheckID":"service:web","Name":"Service 'web' check","Status":"critical","Notes":"","ServiceID":"web","ServiceName":"web"}]
```

We can see that there is only a single check in the `critical` state, which is
our `web` service check.

Additionally, we can attempt to query the web service using DNS. Terraform
will not return any results, since the service is unhealthy:

```
 dig @127.0.0.1 -p 8600 web.service.terraform
...

;; QUESTION SECTION:
;web.service.terraform.		IN	A
```

This section should have shown that checks can be easily added. Check definitions
can be updated by changing configuration files and sending a `SIGHUP` to the agent.
Alternatively the HTTP API can be used to add, remove and modify checks dynamically.
The API allows for a "dead man's switch" or [TTL based check](/docs/agent/checks.html).
TTL checks can be used to integrate an application more tightly with Terraform, enabling
business logic to be evaluated as part of passing a check.

