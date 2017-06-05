---
layout: "google"
page_title: "Google: google_sql_database_instance"
sidebar_current: "docs-google-sql-database-instance"
description: |-
  Creates a new SQL database instance in Google Cloud SQL.
---

# google\_sql\_database\_instance

Creates a new Google SQL Database Instance. For more information, see the [official documentation](https://cloud.google.com/sql/), or the [JSON API](https://cloud.google.com/sql/docs/admin-api/v1beta4/instances).

~> **NOTE on `google_sql_database_instance`:** - Second-generation instances include a
default 'root'@'%' user with no password. This user will be deleted by Terraform on
instance creation. You should use a `google_sql_user` to define a customer user with
a restricted host and strong password.


## Example Usage

Example creating a SQL Database.

```hcl
resource "google_sql_database_instance" "master" {
  name = "master-instance"

  settings {
    tier = "D0"
  }
}
```

## Argument Reference

The following arguments are supported:

* `region` - (Required) The region the instance will sit in. Note, this does
    not line up with the Google Compute Engine (GCE) regions - your options are
    `us-central`, `asia-west1`, `europe-west1`, and `us-east1`.

* `settings` - (Required) The settings to use for the database. The
    configuration is detailed below.

- - -

* `database_version` - (Optional, Default: `MYSQL_5_6`) The MySQL version to
    use. Can be either `MYSQL_5_6` or `MYSQL_5_7` for second-generation
    instances, or `MYSQL_5_5` or `MYSQL_5_6` for first-generation instances.
    See Google's [Second Generation Capabilities](https://cloud.google.com/sql/docs/1st-2nd-gen-differences)
    for more information.

* `name` - (Optional, Computed) The name of the instance. If the name is left
    blank, Terraform will randomly generate one when the instance is first
    created. This is done because after a name is used, it cannot be reused for
    up to [one week](https://cloud.google.com/sql/docs/delete-instance).

* `master_instance_name` - (Optional) The name of the instance that will act as
    the master in the replication setup. Note, this requires the master to have
    `binary_log_enabled` set, as well as existing backups.

* `project` - (Optional) The project in which the resource belongs. If it
    is not provided, the provider project is used.

* `replica_configuration` - (Optional) The configuration for replication. The
    configuration is detailed below.

The required `settings` block supports:

* `tier` - (Required) The machine tier (First Generation) or type (Second Generation) to use. See
    [tiers](https://cloud.google.com/sql/docs/admin-api/v1beta4/tiers) for more details and
    supported versions.

* `activation_policy` - (Optional) This specifies when the instance should be
    active. Can be either `ALWAYS`, `NEVER` or `ON_DEMAND`.

* `authorized_gae_applications` - (Optional) A list of Google App Engine (GAE)
    project names that are allowed to access this instance.

* `crash_safe_replication` - (Optional) Specific to read instances, indicates
    when crash-safe replication flags are enabled.

* `disk_autoresize` - (Optional, Second Generation, Default: `true`) Configuration to increase storage size automatically.

* `disk_size` - (Optional, Second Generation, Default: `10`) The size of data disk, in GB. Size of a running instance cannot be reduced but can be increased.

* `disk_type` - (Optional, Second Generation, Default: `PD_SSD`) The type of data disk: PD_SSD or PD_HDD.

* `pricing_plan` - (Optional, First Generation) Pricing plan for this instance, can be one of
    `PER_USE` or `PACKAGE`.

* `replication_type` - (Optional) Replication type for this instance, can be one
    of `ASYNCHRONOUS` or `SYNCHRONOUS`.

The optional `settings.database_flags` sublist supports:

* `name` - (Optional) Name of the flag.

* `value` - (Optional) Value of the flag.

The optional `settings.backup_configuration` subblock supports:

* `binary_log_enabled` - (Optional) True iff binary logging is enabled. If
    `logging` is false, this must be as well.

* `enabled` - (Optional) True iff backup configuration is enabled.

* `start_time` - (Optional) `HH:MM` format time indicating when backup
    configuration starts.

The optional `settings.ip_configuration` subblock supports:

* `ipv4_enabled` - (Optional) True iff the instance should be assigned an IP
    address.

* `require_ssl` - (Optional) True iff mysqld should default to `REQUIRE X509`
    for users connecting over IP.

The optional `settings.ip_configuration.authorized_networks[]` sublist supports:

* `expiration_time` - (Optional) The [RFC 3339](https://tools.ietf.org/html/rfc3339)
  formatted date time string indicating when this whitelist expires.

* `name` - (Optional) A name for this whitelist entry.

* `value` - (Optional) A CIDR notation IPv4 or IPv6 address that is allowed to
    access this instance. Must be set even if other two attributes are not for
    the whitelist to become active.

The optional `settings.location_preference` subblock supports:

* `follow_gae_application` - (Optional) A GAE application whose zone to remain
    in. Must be in the same region as this instance.

* `zone` - (Optional) The preferred compute engine
    [zone](https://cloud.google.com/compute/docs/zones?hl=en).

The optional `settings.maintenance_window` subblock for Second Generation
instances declares a one-hour [maintenance window](https://cloud.google.com/sql/docs/instance-settings?hl=en#maintenance-window-2ndgen)
when an Instance can automatically restart to apply updates. It supports:

* `day` - (Optional) Day of week (`1-7`), starting on Monday

* `hour` - (Optional) Hour of day (`0-23`), ignored if `day` not set

* `update_track` - (Optional) Receive updates earlier (`canary`) or later 
(`stable`)

The optional `replica_configuration` block must have `master_instance_name` set
to work, cannot be updated, and supports:

* `ca_certificate` - (Optional) PEM representation of the trusted CA's x509
    certificate.

* `client_certificate` - (Optional) PEM representation of the slave's x509
    certificate.

* `client_key` - (Optional) PEM representation of the slave's private key. The
    corresponding public key in encoded in the `client_certificate`.

* `connect_retry_interval` - (Optional, Default: 60) The number of seconds
    between connect retries.

* `dump_file_path` - (Optional) Path to a SQL file in GCS from which slave
    instances are created. Format is `gs://bucket/filename`.

* `failover_target` - (Optional) Specifies if the replica is the failover target.
    If the field is set to true the replica will be designated as a failover replica.
    If the master instance fails, the replica instance will be promoted as
    the new master instance.

* `master_heartbeat_period` - (Optional) Time in ms between replication
    heartbeats.

* `password` - (Optional) Password for the replication connection.

* `sslCipher` - (Optional) Permissible ciphers for use in SSL encryption.

* `username` - (Optional) Username for replication connection.

* `verify_server_certificate` - (Optional) True iff the master's common name
    value is checked during the SSL handshake.

## Attributes Reference

In addition to the arguments listed above, the following computed attributes are
exported:

* `ip_address.0.ip_address` - The IPv4 address assigned.

* `ip_address.0.time_to_retire` - The time this IP address will be retired, in RFC
    3339 format.

* `self_link` - The URI of the created resource.

* `settings.version` - Used to make sure changes to the `settings` block are
    atomic.
