---
layout: "docs"
page_title: "Providers vs. Resources & Data Sources"
sidebar_current: "docs-internals-provider-guide-vs-resources"
description: |-
  Understanding the difference between providers and resources and data sources.
---

# Providers vs Resources & Data Sources
The three major concepts to understand in the provider framework are **Data Sources**, **Resources**, and **Providers**. Providers are the API provider that is being integrated; for example, Amazon’s Web Services APIs all fall under a single provider, GitHub’s APIs all fall under a single provider, and so on. Resources are the specific API resources that are being manipulated via their own CR(U)D interface; for example, an Instance in Amazon’s EC2 API would be its own resource, a repository in GitHub’s API would be its own resource, and so on. Data Sources are read-only resources that are exposed by an API, that Terraform isn’t managing but still wants to be able to reference. Things like which disk images are available in Google Cloud Platform, for example.

A good way to think of Providers and Resources/Data Sources is that Providers define all the common configuration you need to call an API, while Resources and Data Sources actually make the calls.
