---
layout: "intro"
page_title: "Web UI"
sidebar_current: "gettingstarted-ui"
---

# Terraform Web UI

Terraform comes with support for a
[beautiful, functional web UI](http://demo.terraform.io) out of the box.
This UI can be used for viewing all services and nodes, viewing all
health checks and their current status, and for reading and setting
key/value data. The UI automatically supports multi-datacenter.

For ease of deployment, the UI is
[distributed](/downloads_web_ui.html)
as static HTML and JavaScript.
You don't need a separate web server to run the web UI. The Terraform
agent itself can be configured to serve the UI.

## Screenshot and Demo

You can view a live demo of the Terraform Web UI
[here](http://demo.terraform.io).

While the live demo is able to access data from all datacenters,
we've also setup demo endpoints in the specific datacenters:
[AMS2](http://ams2.demo.terraform.io) (Amsterdam),
[SFO1](http://sfo1.demo.terraform.io) (San Francisco),
and [NYC1](http://nyc1.demo.terraform.io) (New York).

A screenshot of one page of the demo is shown below so you can get an
idea of what the web UI is like. Click the screenshot for the full size.

<div class="center">
<a href="/images/terraform_web_ui.png">
<img src="/images/terraform_web_ui.png">
</a>
</div>

## Set Up

To set up the web UI,
[download the web UI package](/downloads_web_ui.html)
and unzip it to a directory somewhere on the server where the Terraform agent
is also being run. Then, just append the `-ui-dir` to the `terraform agent`
command pointing to the directory where you unzipped the UI (the
directory with the `index.html` file):

```
$ terraform agent -ui-dir /path/to/ui
...
```

The UI is available at the `/ui` path on the same port as the HTTP API.
By default this is `http://localhost:8500/ui`.
