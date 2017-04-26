---
layout: "circonus"
page_title: "Circonus: circonus_contact_group"
sidebar_current: "docs-circonus-resource-circonus_contact_group"
description: |-
  Manages a Circonus Contact Group.
---

# circonus\_contact_group

The ``circonus_contact_group`` resource creates and manages a
[Circonus Contact Group](https://login.circonus.com/user/docs/Alerting/ContactGroups).


## Usage

```hcl
resource "circonus_contact_group" "myteam-alerts" {
  name = "MyTeam Alerts"

  email {
    user = "/user/1234"
  }

  email {
    user = "/user/5678"
  }

  email {
    address = "user@example.com"
  }

  http {
    address = "https://www.example.org/post/endpoint"
    format = "json"
    method = "POST"
  }

  irc {
    user = "/user/6331"
  }

  slack {
    channel = "#myteam"
    team = "T038UT13D"
  }

  sms {
    user = "/user/1234"
  }

  sms {
    address = "8005551212"
  }

  victorops {
    api_key = "xxxx"
    critical = 2
    info = 5
    team = "myteam"
    warning = 3
  }

  xmpp {
    user = "/user/9876"
  }

  aggregation_window = "5m"

  alert_option {
    severity = 1
    reminder = "5m"
    escalate_to = "/contact_group/4444"
  }

  alert_option {
    severity = 2
    reminder = "15m"
    escalate_after = "2h"
    escalate_to = "/contact_group/4444"
  }

  alert_option {
    severity = 3
    reminder = "24m"
    escalate_after = "3d"
    escalate_to = "/contact_group/4444"
  }
}
```

## Argument Reference

* `aggregation_window` - (Optional) The aggregation window for batching up alert
  notifications.

* `alert_option` - (Optional) There is one `alert_option` per severity, where
  severity can be any number between 1 (high) and 5 (low).  If configured, the
  alerting system will remind or escalate alerts to further contact groups if an
  alert sent to this contact group is not acknowledged or resolved.  See below
  for details.

* `email` - (Optional) Zero or more `email` attributes may be present to
  dispatch email to Circonus users by referencing their user ID, or by
  specifying an email address.  See below for details on supported attributes.

* `http` - (Optional) Zero or more `http` attributes may be present to dispatch
  [Webhook/HTTP requests](https://login.circonus.com/user/docs/Alerting/ContactGroups#WebhookNotifications)
  by Circonus.  See below for details on supported attributes.

* `irc` - (Optional) Zero or more `irc` attributes may be present to dispatch
  IRC notifications to users.  See below for details on supported attributes.

* `long_message` - (Optional) The bulk of the message used in long form alert
  messages.

* `long_subject` - (Optional) The subject used in long form alert messages.

* `long_summary` - (Optional) The brief summary used in long form alert messages.

* `name` - (Required) The name of the contact group.

* `pager_duty` - (Optional) Zero or more `pager_duty` attributes may be present
  to dispatch to
  [Pager Duty teams](https://login.circonus.com/user/docs/Alerting/ContactGroups#PagerDutyOptions).
  See below for details on supported attributes.

* `short_message` - (Optional) The subject used in short form alert messages.

* `short_summary` - (Optional) The brief summary used in short form alert
  messages.

* `slack` - (Optional) Zero or more `slack` attributes may be present to
  dispatch to Slack teams.  See below for details on supported attributes.

* `sms` - (Optional) Zero or more `sms` attributes may be present to dispatch
  SMS messages to Circonus users by referencing their user ID, or by specifying
  an SMS Phone Number.  See below for details on supported attributes.

* `tags` - (Optional) A list of tags attached to the Contact Group.

* `victorops` - (Optional) Zero or more `victorops` attributes may be present
  to dispatch to
  [VictorOps teams](https://login.circonus.com/user/docs/Alerting/ContactGroups#VictorOps).
  See below for details on supported attributes.

## Supported Contact Group `alert_option` Attributes

* `escalate_after` - (Optional) How long to wait before escalating an alert that
  is received at a given severity.

* `escalate_to` - (Optional) The Contact Group ID who will receive the
  escalation.

* `reminder` - (Optional) If specified, reminders will be sent after a user
  configurable number of minutes for open alerts.

* `severity` - (Required) An `alert_option` must be assigned to a given severity
  level.  Valid severity levels range from 1 (highest severity) to 5 (lowest
  severity).

## Supported Contact Group `email` Attributes

Either an `address` or `user` attribute is required.

* `address` - (Optional) A well formed email address.

* `user` - (Optional) An email will be sent to the email address of record for
  the corresponding user ID (e.g. `/user/1234`).

A `user`'s email address is automatically maintained and kept up to date by the
recipient, whereas an `address` provides no automatic layer of indirection for
keeping the information accurate (including LDAP and SAML-based authentication
mechanisms).

## Supported Contact Group `http` Attributes

* `address` - (Required) URL to send a webhook request to.

* `format` - (Optional) The payload of the request is a JSON-encoded payload
  when the `format` is set to `json` (the default).  The alternate payload
  encoding is `params`.

* `method` - (Optional) The HTTP verb to use when making a request.  Either
  `GET` or `POST` may be specified. The default verb is `POST`.

## Supported Contact Group `irc` Attributes

* `user` - (Required) When a user has configured IRC on their user account, they
  will receive an IRC notification.

## Supported Contact Group `pager_duty` Attributes

* `contact_group_fallback` - (Optional) If there is a problem contacting
  PagerDuty, relay the notification automatically to the specified Contact Group
  (e.g. `/contact_group/1234`).

* `service_key` - (Required) The PagerDuty Service Key.

* `webhook_url` - (Required) The PagerDuty webhook URL that PagerDuty uses to
  notify Circonus of acknowledged actions.

## Supported Contact Group `slack` Attributes

* `contact_group_fallback` - (Optional) If there is a problem contacting Slack,
  relay the notification automatically to the specified Contact Group
  (e.g. `/contact_group/1234`).

* `buttons` - (Optional) Slack notifications can have acknowledgement buttons
  built into the notification message itself when enabled.  Defaults to `true`.

* `channel` - (Required) Specify what Slack channel Circonus should send alerts
  to.

* `team` - (Required) Specify what Slack team Circonus should look in for the
  aforementioned `channel`.

* `username` - (Optional) Specify the username Circonus should advertise itself
  as in Slack.  Defaults to `Circonus`.

## Supported Contact Group `sms` Attributes

Either an `address` or `user` attribute is required.

* `address` - (Optional) SMS Phone Number to send a short notification to.

* `user` - (Optional) An SMS page will be sent to the phone number of record for
  the corresponding user ID (e.g. `/user/1234`).

A `user`'s phone number is automatically maintained and kept up to date by the
recipient, whereas an `address` provides no automatic layer of indirection for
keeping the information accurate (including LDAP and SAML-based authentication
mechanisms).

## Supported Contact Group `victorops` Attributes

* `contact_group_fallback` - (Optional) If there is a problem contacting
  VictorOps, relay the notification automatically to the specified Contact Group
  (e.g. `/contact_group/1234`).

* `api_key` - (Required) The API Key for talking with VictorOps.

* `critical` - (Required)
* `info` - (Required)
* `team` - (Required)
* `warning` - (Required)

## Supported Contact Group `xmpp` Attributes

Either an `address` or `user` attribute is required.

* `address` - (Optional) XMPP address to send a short notification to.

* `user` - (Optional) An XMPP notification will be sent to the XMPP address of
  record for the corresponding user ID (e.g. `/user/1234`).

## Import Example

`circonus_contact_group` supports importing resources.  Supposing the following
Terraform:

```hcl
provider "circonus" {
  alias = "b8fec159-f9e5-4fe6-ad2c-dc1ec6751586"
}

resource "circonus_contact_group" "myteam" {
  name = "My Team's Contact Group"

  email {
    address = "myteam@example.com"
  }

  slack {
    channel = "#myteam"
    team = "T024UT03C"
  }
}
```

It is possible to import a `circonus_contact_group` resource with the following command:

```
$ terraform import circonus_contact_group.myteam ID
```

Where `ID` is the `_cid` or Circonus ID of the Contact Group
(e.g. `/contact_group/12345`) and `circonus_contact_group.myteam` is the name of
the resource whose state will be populated as a result of the command.
