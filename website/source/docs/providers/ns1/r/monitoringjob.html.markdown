---
description: 'Provides an NS1 monitoring job resource.'
layout: ns1
page_title: 'NS1: ns1_monitoringjob'
sidebar_current: 'docs-ns1-resource-monitoringjob'
---

# ns1\_monitoringjob

NS1's Monitoring jobs enable up/down monitoring of your different service endpoints, and can feed directly into DNS records to drive DNS failover.

## Example Usage

    # Add a minimal monitoring job
    resource "ns1_monitoringjob" "foobar" {
      zone = "${var.ns1_domain}"
    }

    # Add a complete monitoring job
    resource "ns1_monitoringjob" "foobar" {
      zone        = "${var.ns1_domain}"
      ttl         = 10800
      refresh     = 3600
      retry       = 300
      expiry      = 2592000
      nx_ttl      = 1234
    }

## Argument Reference

See [the NS1 API Docs](https://ns1.com/api/) for details about valid
values. Many are originally defined in
[RFC-1035](https://tools.ietf.org/html/rfc1035).

The following arguments are supported:

  * `name` - (Required) The friendly name of this monitoring job.
  * `job_type` - (Required) One of the job types from the `/monitoring/jobtypes` NS1 API endpoint.
  * `regions` - (Required) NS1 Monitoring regions to run the job in. List of valid regions is available from the `/monitoring/regions` NS1 API endpoint.
  * `frequency` - (Required) How often to run the job in seconds. Int.
  * `config` - (Required) A map of configuration for this job_type, see the `/monitoring/jobtypes` NS1 API endpoint for more info.

  * `active` - If the job is active. Bool. Default: `true`.
  * `policy` - The policy of how many regions need to fail to make the check fail, this is one of: `"quorum"`, `"one"`, `"all"`. Default: `"quorum"`.
  * `rapid_recheck` - If the check should be immediately re-run if it fails. Bool. Default: `false`.
  * `notes` - Operator notes about what this monitoring job does.
  * `notify_delay` - How long this job needs to be failing for before notifying. Int.
  * `notify_repeat` - How often to repeat the notification if unfixed.  Int.
  * `notify_failback` - Notify when fixed. Bool.
  * `notify_regional` - Notify (when using multiple regions, and quorum or all policies) if an individual region fails checks. Bool.
  * `notify_list` - Notification list id to send notifications to when this monitoring job fails.
  * `rules` - List of rules determining failure conditions.  Each entry must have the following inputs:
    * `value` - (Required) Value to compare to.
    * `comparison` - (Required) Type of comparison to perform.
    * `key` - (Required) The output key from the job, to which the value will be compared - see the `/monitoring/jobtypes` NS1 API endpoint for list of valid keys for each job type.

## Attributes Reference

The following attributes are exported:

  * `id` - The internal NS1 ID of this monitoring job. This is passed into a `resource_datafeed` `config.jobid`.
