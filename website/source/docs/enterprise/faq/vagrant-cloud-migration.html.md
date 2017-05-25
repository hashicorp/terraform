---
layout: "enterprise"
page_title: "Vagrant Cloud Migration - FAQ - Terraform Enterprise"
sidebar_current: "docs-enterprise-faq-vagrant-cloud-migration"
description: |-
  Vagrant-related functionality will be moved from Terraform Enterprise into its own product, Vagrant Cloud. This migration is currently planned for June 27th, 2017.
---

# Vagrant Cloud Migration

Vagrant-related functionality will be moved from Terraform Enterprise into its own product, Vagrant Cloud. This migration is currently planned for **June 27th, 2017**.

All existing Vagrant boxes will be moved to the new system on that date. All users, organizations, and teams will be copied as well.

## Authentication Tokens

No existing Terraform Enterprise authentication tokens will be transferred. To prevent a disruption of service for Vagrant-related operations, users must create a new authentication token and check "Migrate to Vagrant Cloud" and begin using these tokens for creating and modifying Vagrant boxes. These tokens will be moved on the migration date.

Creating a token via `vagrant login` will also mark a token as "Migrate to Vagrant Cloud".

## More Information

At least 1 month prior to the migration, we will be releasing more information on the specifics and impact of the migration.