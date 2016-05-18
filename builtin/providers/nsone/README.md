# Synopsis

This is an *experimental* provider which allows Terraform to create
and update DNS zones, records, monitoring jobs, data sources and
feeds, and other resources with the (NSONE)[http://nsone.net/] API.

# Example use

    provider "nsone" {
      api_key = "xxxxxxx" # Or set NSONE_APIKEY environment variable
    }

    resource "nsone_datasource" "api" {
      name = "terraform test"
      sourcetype = "nsone_v1"
    }

    resource "nsone_datafeed" "exampledc1" {
      name = "exampledc1"
      source_id = "${nsone_datasource.api.id}"
      config {
        label = "exampledc1"
      }
    }

    resource "nsone_datafeed" "exampledc2" {
      name = "exampledc2"
      source_id = "${nsone_datasource.api.id}"
      config {
        label = "exampledc2"
      }
    }

    resource "nsone_zone" "example" {
      zone = "mycompany.com"
      ttl = 60
    }

    resource "nsone_record" "www" {
      zone = "${nsone_zone.example.zone}"
      domain = "www.${nsone_zone.example.zone}"
      type = "A"
      answers {
        answer = "1.1.1.1"
        meta {
          field = "up"
          feed = "${nsone_datafeed.exampledc1.id}"
        }
        region = "useast"
      }
      answers {
        answer = "2.2.2.2"
        meta {
          feed = "${nsone_datafeed.exampledc2.id}"
          field = "up"
        }
        region = "uswest"
      }
      regions {
        name = "useast"
        georegion = "US-EAST"
      }
      regions {
        name = "uswest"
        georegion = "US-WEST"
      }
      filters {
        filter = "up"
      }
      filters {
        filter = "select_first_n"
        config {
          N = 1
        }
      }
    }

    resource "nsone_record" "star" {
      link = "www.${nsone_zone.example.zone}"
    }

    resource "nsone_zone" "co_uk" {
      zone = "mycompany.co.uk"
      link = "${nsone_zone.example.zone}"
    }

    resource "nsone_monitoringjob" "useast" {
      name = "useast"
      active = true
      regions = [ "lga" ]
      job_type = "tcp"
      frequency = 60
      rapid_recheck = true
      policy = "all"
      config {
        send = "HEAD / HTTP/1.0\r\n\r\n"
        port = 80
        host = "1.1.1.1"
      }
    }

See the examples/ folder in the repository for more detailed examples.

# Installing

    make install

Should do the right thing assuming that you have terraform already installed, and this code
is placed in the right place in your $GOPATH.

# Supported features

## Create and manage DNS zones
    * Normal primary zones supported
    * Linked zones supported
    * Secondary (slave) zones supported

## Manage records in your zones
    * A, MX, ALIAS and CNAME record types are supported.
    * Other record types MAY work, but are untested.
    * Supports NSONE's Linked Records
    * Supports multiple answers, each of which can be connected to data feeds
    * Add filter chains to records, with config
    * Add regions to answers, and the record. Some (not all!) region metadata types are supported.

## Data sources
    * Can create datasources with arbitrary config
    * This *should* work for all datasource types, but only nsone_v1 and nsone_monitoring are tested

## Data feeds
    * Create data feeds connected to a data source with a label

## NSONE monitoring
    * Create and manage monitoring jobs.
    * Connect monitoring notifications to data feeds and use monitors to control record up/down status.

## Users / Account management / API keys
  * Creation of users, API keys and teams is fully supported

# Unsupported features

## Zones
  * Whitelisting of secondary servers to allow AXFR is currently unsupported

## Records
  * Metadata support in regions is limited
  * Record-wide metadata is unsupported

## NSONE monitoring
  * Notification list management is not supported

# Resources provided

## nsone_zone

Creates a DNS zone in NSONE.

### Inputs

  * zone - The name of the zone to define [Required]
  * link - The name of another zone to link this zone to (so that they serve the same DNS records) [Optional]
  * ttl - The TTL for the SOA record for this zone [Optional]
  * nx_ttl - The TTL for NXDOMAIN answers in this zone [Optional]
  * refresh - Frequency for slaves to try to refresh this zone [Optional]
  * retry - Time between slave retries if "refresh" has expired [Optional]
  * expiry - Time after an expired "refresh" to keep "retrying" before giving up [Optional]
  * primary - The master nameserver hostname to AXFR this zone from, creates a secondary zone. [Optional]

### Outputs

  * id - The internal ID for the zone in the NSONE API
  * hostmaster - The hostmaster for the zone

## nsone_record

Creates a DNS record and optional traffic management config in NSONE.

### Inputs

  * zone - The name of the zone that this record lives in [Required]
  * domain - The full domain name of this record [Required]
  * ttl - A TTL specific to this record [Optional]
  * type - The type of the record [Required]
  * meta - Record wide metadata [Currently unsupported]
  * link - The domain of a record to link this record to (so that they serve the same answers) [Optional]
  * answers - The set of answers that it's possible to return. This stanza can be repeated.
    * answer - The DNS RDATA of the answer to return (e.g. "1.2.3.4" for an A record, or "some.example.com" for a CNAME). FIXME - Test/fix other record types. [Required]
    * region - The name of the region (from 'regions', below) to assign this answer to. Regions may be used to specify metadata that should apply across all answers in the region. [Optional]
    * meta - Add metadata key/value pairs to this answer, used for filtering.  This stanza can be repeated. Get the current set of supported metadata types from the /metatypes NSONE API endpoint. [Optional]
      * field - The metadata field name to update from a feed. [Required]
      * feed - The id of the data feed which updates this field. [Optional, conflicts with value]
      * value - The static value to set for this metadata field. [Optional, conflicts with feed]
  * regions - The set of regions into which answers may be grouped.  Each region has its own metadata. This stanza can be repeated. [Optional]
    * name - The name of this region (the name provided in an answer) [Required]
    * georegion - The name of the geographic region which corresponds to this region. Allowed values are: US-WEST, US-EAST, US-CENTRAL, EUROPE, AFRICA, ASIAPAC, SOUTH-AMERICA. [Optional]
    * country - The name of the country which corresponds to this region. FIXME countries? [Optional]
    * us_state - The name of the US state which corresponds to this region. FIXME states? [Optional]
    * FIXME - Add the rest!
    * FIXME - Add support for having feeds at the region level!
  * filters - The Filter Chain to apply to the answers, consisting of a list of filter algorithms. This stanza can be repeated. Order matters when creating a Filter Chain. [Optional]
    * filter - The type of this filter. Get possible filters from the /filtertypes NSONE API endpoint. [Required]
    * disabled - If this filter should be disabled. [Optional]
    * config - A map of configuration key/value pairs specific to this filter. Get the possible/required keys and values from the /filtertypes NSONE API endpoint. [Optional]

### Outputs

  * id - The internal NSONE ID for this record
  * ttl - The record's TTL

## nsone_datasource

NSONE Data Sources are conduits for pushing updates to DNS record/answer metadata through individual data feeds to NSONE's platform.

### Inputs

  * name - The name to associate with this data source. [Required]
  * sourcetype - The type of data source to create. Get the current set of supported data source types from the /data/sourcetypes NSONE API endpoint. nsone_v1, nsone_monitoring are currently tested. [Required]

### Outputs

  * id - The internal NSONE id of this data source. This is used when associating feeds with this source (nsone_datafeed's source_id).

## nsone_datafeed

Multiple data feeds may be associated with an NSONE Data Source -- for example, feeds keyed to individual servers, monitoring jobs, etc.

### Inputs

  * source_id - The internal NSONE id of the data source this feed is attached to. [Required]
  * name - The user friendly name for this feed [Required]
  * config - A map of configuration key/value pairs for this data feed. The keys and values required vary depending on the type of source. Get the current set of supported config keys from the /data/sourcetypes NSONE API endpoint. [Optional]

### Outputs

  * id - The internal NSONE id of this data feed. This is passed into resource_record's answer.meta.feed field

## nsone_monitoringjob

NSONE's Monitoring jobs enable up/down monitoring of your different service endpoints, and can feed directly into DNS records to drive DNS failover.

### Inputs

  * name - The friendly name of this monitoring job [Required]
  * active - If the job is active [Bool, Required]
  * regions - NSONE Monitoring regions to run the job in. List of valid regions is available from the /monitoring/regions NSONE API endpoint. [Required]
  * job_type - One of the job types from the /monitoring/jobtypes NSONE API endpoint. [Required]
  * frequency - How often to run the job in seconds [Int, Required]
  * rapid_recheck - If the check should be immediately re-run if it fails [Bool, Required]
  * policy - The policy of how many regions need to fail to make the check fail, this is one of: quorum, one, all. [Required]
  * notes - Operator notes about what this monitoring job does. [Optional]
  * config - A map of configuration for this job_type, see the /monitoring/jobtypes NSONE API endpoint for more info. [Required]
  * notify_delay - How long this job needs to be failing for before notifying [Int, Optional]
  * notify_repeat - How often to repeat the notification if unfixed [Int, Optional]
  * notify_failback - Notify when fixed [Bool, Optional]
  * notify_regional - Notify (when using multiple regions, and quorum or all policies) if an individual region fails checks [Bool, Optional]
  * notify_list - Notification list id to send notifications to when this monitoring job fails [Optional]
  * rules - List of rules determining failure conditions.  Each entry must have the following inputs: [Optional]
    * value - Value to compare to [Required]
    * comparison - Type of comparison to perform [Required]
    * key - The output key from the job, to which the value will be compared - see the /monitoring/jobtypes NSONE API endpoint for list of valid keys for each job type [Required]

### Outputs

  * id - The internal NSONE id of this monitoring job. This is passed into resource_datafeed's config.jobid

## nsone_user

### Inputs

N.B. This *also* has all the inputs from the nsone_team resource, which you can use *instead* of assigning to key to one or more teams.

 * username - The user's login username [Required]
 * email - The user's email address [Required]
 * name - The user's full name [Required]
 * notify
   * billing - Whether the user should receive billing notifications [Bool, Optional]
 * teams - List of the nsone_team ids to attach to this user's permissions. [Optional]

### Outputs

  * id - The internal NSONE id of this user.
  * FIXME - Add current registration/login status?

## nsone_apikey

### Inputs

N.B. This *also* has all the inputs from the nsone_team resource, which you can use *instead* of assigning to key to one or more teams.

 * teams - List of the nsone_team ids to attach to this API key's permissions.

### Outputs

 * key - The API key that has been generated.
 * id - The internal NSONE id of this api key.

## nsone_team

Permissions are all optional -- by default, a user is granted an unspecified permission.

### Inputs

  * name - The name of this team. [Required]
  * dns_view_zones [Bool]
  * dns_manage_zones [Bool]
  * dns_zones_allow_by_default [Bool]
  * dns_zones_deny - List of zones [Optional]
  * dns_zones_allow - List of zones [Optional]
  * data_push_to_datafeeds [Bool]
  * data_manage_datasources [Bool]
  * data_manage_datafeeds [Bool]
  * account_manage_users [Bool]
  * account_manage_payment_methods [Bool]
  * account_manage_plan [Bool]
  * account_manage_teams [Bool]
  * account_manage_apikeys [Bool]
  * account_manage_account_settings [Bool]
  * account_view_activity_log [Bool]
  * account_view_invoices [Bool]
  * monitoring_manage_lists [Bool]
  * monitoring_manage_jobs [Bool]
  * monitoring_view_jobs [Bool]

### Outputs

  * id - The internal NSONE id of this team.

# Support / contributions

I'm planning to continue developing and supporting this code for my use-cases,
and I'll do my best to respond to issues and pull requests (and I'm happy to take
patches to improve the code or add missing feaures!).

Also, please be warned that I am *not* a competent Go programmer, so please expect
to find hideous / insane / non-idiomatic code if you look at the source. I'd be
extremely happy to accept patches or suggestions from anyone more experience than me:)

Please *feel free* to contact me via github issues or Twitter or irc in #terraform (t0m)
if you have *any* issues with, or would like advice on using this code.

# Copyright

Copyright (c) Tomas Doran 2015

# LICENSE

Apache2 - see the included LICENSE file for more information

