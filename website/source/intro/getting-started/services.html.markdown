---
layout: "intro"
page_title: "Registering Services"
sidebar_current: "gettingstarted-services"
---

# Registering Services

In the previous page, we ran our first agent, saw the cluster members, and
queried that node. On this page, we'll register our first service and query
that service. We're not yet running a cluster of Terraform agents.

## Defining a Service

A service can be registered either by providing a
[service definition](/docs/agent/services.html),
or by making the appropriate calls to the
[HTTP API](/docs/agent/http.html).

We're going to start by registering a service using a service definition,
since this is the most common way that services are registered. We'll be
building on what we covered in the
[previous page](/intro/getting-started/agent.html).

First, create a directory for Terraform configurations. A good directory
is typically `/etc/terraform.d`. Terraform loads all configuration files in the
configuration directory.

```
$ sudo mkdir /etc/terraform.d
```

Next, we'll write a service definition configuration file. We'll
pretend we have a service named "web" running on port 80. Additionally,
we'll give it some tags, which we can use as additional ways to query
it later.

```
$ echo '{"service": {"name": "web", "tags": ["rails"], "port": 80}}' \
    >/etc/terraform.d/web.json
```

Now, restart the agent we're running, providing the configuration directory:

```
$ terraform agent -server -bootstrap -data-dir /tmp/consul -config-dir /etc/consul.d
==> Starting Terraform agent...
...
    [INFO] agent: Synced service 'web'
...
```

You'll notice in the output that it "synced" the web service. This means
that it loaded the information from the configuration.

If you wanted to register multiple services, you create multiple service
definition files in the Terraform configuration directory.

## Querying Services

Once the agent is started and the service is synced, we can query that
service using either the DNS or HTTP API.

### DNS API

Let's first query it using the DNS API. For the DNS API, the DNS name
for services is `NAME.service.terraform`. All DNS names are always in the
`terraform` namespace. The `service` subdomain on that tells Terraform we're
querying services, and the `NAME` is the name of the service. For the
web service we registered, that would be `web.service.terraform`:

```
$ dig @127.0.0.1 -p 8600 web.service.terraform
...

;; QUESTION SECTION:
;web.service.terraform.		IN	A

;; ANSWER SECTION:
web.service.terraform.	0	IN	A	172.20.20.11
```

As you can see, an A record was returned with the IP address of the node that
the service is available on. A records can only hold IP addresses. You can
also use the DNS API to retrieve the entire address/port pair using SRV
records:

```
$ dig @127.0.0.1 -p 8600 web.service.terraform SRV
...

;; QUESTION SECTION:
;web.service.terraform.	IN	SRV

;; ANSWER SECTION:
web.service.terraform. 0	IN	SRV	1 1 80 agent-one.node.dc1.consul.

;; ADDITIONAL SECTION:
agent-one.node.dc1.terraform. 0	IN	A	172.20.20.11
```

The SRV record returned says that the web service is running on port 80
and exists on the node `agent-one.node.dc1.terraform.`. An additional section
is returned by the DNS with the A record for that node.

Finally, we can also use the DNS API to filter services by tags. The
format for tag-based service queries is `TAG.NAME.service.terraform`. In
the example below, we ask Terraform for all web services with the "rails"
tag. We get a response since we registered our service with that tag.

```
$ dig @127.0.0.1 -p 8600 rails.web.service.terraform
...

;; QUESTION SECTION:
;rails.web.service.terraform.		IN	A

;; ANSWER SECTION:
rails.web.service.terraform.	0	IN	A	172.20.20.11
```

### HTTP API

In addition to the DNS API, the HTTP API can be used to query services:

```
$ curl http://localhost:8500/v1/catalog/service/web
[{"Node":"agent-one","Address":"172.20.20.11","ServiceID":"web","ServiceName":"web","ServiceTags":["rails"],"ServicePort":80}]
```

## Updating Services

Service definitions can be updated by changing configuration files and
sending a `SIGHUP` to the agent. This lets you update services without
any downtime or unavailability to service queries.

Alternatively the HTTP API can be used to add, remove, and modify services
dynamically.

