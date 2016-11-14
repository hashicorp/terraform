---
layout: "docs"
page_title: "Frequently Asked Questions"
sidebar_current: "docs-faq"
description: |-
    Frequently asked questions and answers for Nomad
---

# Frequently Asked Questions

## Q: What is Checkpoint? / Does Nomad call home?

Nomad makes use of a HashiCorp service called [Checkpoint](https://checkpoint.hashicorp.com)
which is used to check for updates and critical security bulletins.
Only anonymous information, which cannot be used to identify the user or host, is
sent to Checkpoint. An anonymous ID is sent which helps de-duplicate warning messages.
This anonymous ID can can be disabled. Using the Checkpoint service is optional and can be disabled.

See [`disable_anonymous_signature`](/docs/agent/config.html#disable_anonymous_signature)
and [`disable_update_check`](/docs/agent/config.html#disable_update_check).

## Q: How does Atlas integration work?

Nomad makes use of a HashiCorp service called [SCADA](http://scada.hashicorp.com)
(Supervisory Control And Data Acquisition). The SCADA system allows clients to maintain
long-running connections to Atlas. Atlas can in turn provide auto-join facilities for
Nomad agents (supervisory control) and an dashboard showing the state of the system (data acquisition).

Using the SCADA service is optional. SCADA is only enabled by opt-in.

## Q: Is Nomad eventually or strongly consistent?

Nomad makes use of both a [consensus protocol](/docs/internals/consensus.html) and
a [gossip protocol](/docs/internals/gossip.html). The consensus protocol is strongly
consistent, and is used for all state replication and scheduling. The gossip protocol
is used to manage the addresses of servers for automatic clustering and multi-region
federation. This means all data that is managed by Nomad is strongly consistent.
