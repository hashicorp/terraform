# Synopsis

This is a *experimental* provider which allows Terraform to create and update domains and domain
records with the (NSOne)[http://nsone.net/] API.

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

## Setup zones
    * Normal primary zones supported
    * Links to other zones supported
    * Secondary (slave) zones supported

## Setup records in those zones
    * A, MX, ALIAS and CNAME records are supported.
    * Other record types MAY work, but are untested.
    * Allows records to be linked to other records
    * Allows multiple answers, each of which can be linked to data feeds
    * Add filter chains to records, with config
    * Add regions to answers, and the record. Some (not all!) region metadata fields are supported.

## Data sources
    * Can create datasources with arbitrary config
    * This *should* work for all datasource types, but only nsone_v1 and nsone_monitoring are tested

## Data feeds
    * Create data feeds linked to a data source with a label

## NSOne monitoring
    * Create and manage monitoring jobs.
    * Link these to data feeds and use them to control record up/down status.

## Users / Account management / API keys
  * Creation of users, API keys and teams is fully supported

# Unsupported features

# Zones
  * Setting up secondary servers to AXFR is currently unsupported

## Records
  * Metadata support in regions is limited
  * Record wide metadata is unsupported

## NSOne monitoring
  * Notification is not supported

# Resources provided

## nsone_zone

Declares a zone in nsone.

### Inputs

  * zone - The name of the zone to define [Required]
  * link - The name of another zone to link this zone to (so that they serve the same answers) [Optional]
  * ttl - The default TTL for records in this zone [Optional]
  * nx_ttl - The TTL for NXDOMAIN answers in this zone [Optional]
  * retry - [Optional]
  * expiry - [Optional]
  * primary - The master server to AXFR this zone from, creates a secondary zone. [Optional]

### Outputs

  * id - The internal ID for the zone in the NSOne API
  * hostmaster - The hostmaster for the zone

## nsone_record

### Inputs

  * zone - The name of the zone that this record lives in [Required]
  * domain - The full domain name of this record [Required]
  * ttl - A TTL specific to this record [Optional]
  * type - The type of the record [Required]
  * meta - ??? [Optional]
  * link - The domain of a record to link this record to (so that they serve the same answers) [Optional]
  * answers - The set of answers that it's possible to return. This stanza can be repeated.
    * answer - The answer to return. FIXME - Test/fix other record types. [Required]
    * region - The name of the region (from 'regions', below) to assign this answer to. This is used for subsequent geo-filtering. [Optional]
    * meta - Add metadata to this answer, used for filtering. [Optional]
      * field - The metadata field name to update from a feed. [Required]
      * feed - The id of the feed which updates this field. [Optional, conflicts with value]
      * value - The static value to set for this metadata field. [Optional, conflicts with feed]
  * regions - The set of regions to which you can add static metadata, and then associate with answers. This stanza can be repeated.
    * name - The name of this region (the name provided in an answer) [Required]
    * georegion - The name of the geographic region which corresponds to this region. Allowed values are: US-WEST, US-EAST, US-CENTRAL, EUROPE, AFRICA, ASIAPAC, SOUTH-AMERICA. [Optional]
    * country - The name of the country which corresponds to this region. FIXME countries? [Optional]
    * us_state - The name of the US state which corresponds to this region. FIXME states? [Optional]
    * FIXME - Add the rest!
    * FIXME - Add support for having feeds at the region level!
  * filters - The list of filters to apply to results. This stanza can be repeated. [Optional]
    * filter - The name of this filter. Get possible names from FIXME [Required]
    * disabled - If this filter should be disabled. [Optional]
    * config - A map of configuration speciic to this filter. Get the possible/required keys and values from FIXME [Optional]

### Outputs

  * id - The internal NSOne ID for this record
  * ttl - The record's TTL

## nsone_datasource

### Inputs

  * name - The name to associate with this data source. [Required]
  * sourcetype - The type of data source to create. FIXME URL for valid types. nsone_v1, nsone_monitoring are currently tested. [Required]

### Outputs

  * id - The internal NSOne id of this data source. This is used when associating feeds with this source (nsone_datafeed's source_id).

## nsone_datafeed

### Inputs

  * source_id - The internal NSOne id of the data source this feed is getting data from. [Required]
  * name - The friendly name for this feed [Required]
  * config - A map of configuration for this feed. The keys and values required vary depending on the type of source. See the information at FIXME for more details. [Optional]

### Outputs

  * id - The internal NSOne id of this data feed. This is passed into resource_record's answer.meta.feed field

## nsone_monitoringjob

### Inputs

  * name - The friendly name of this monitoring job [Required]
  * active - If the job is active [Bool, Required]
  * regions - Regions to run the job in. List of valid regions from FIXME [Required]
  * job_type - One of the job types from FIXME. [Required]
  * frequency - How often to run the job in seconds [Int, Required]
  * rapid_recheck - If the check should be immediately re-run if it fails [Bool, Required]
  * policy - The policy of how many regions need to fail to make the check fail, this is one of: quorum, one, all. [Required]
  * notes - Notes about what this monitoring job does.
  * config - A map of configuration for this job_type, see FIXME for more info on job types [Required]
  * notify_delay - How long this job needs to be failing for before notifying [Integer]
  * notify_repeat - How often to repeat the notification if unfixed [Integer]
  * notify_failback - Notify when fixed [Bool]
  * notify_regional - Notify (when using multiple regions, and quorum or all policies) if an individual region fails checks [Bool]
  * notify_list - List of FIXMEs to notify when this monitoring job fails
  * rules
    * value
    * comparison
    * key

### Outputs

  * id - The internal NSOne id of this monitoring check. This is passed into resource_datafeed's config.jobid

## nsone_user

### Inputs

N.B. This *also* has all the inputs from the nsone_team resource, which you can use *instead* of assigning to key to one or more teams.

 * username - The user's login username [Required]
 * email - The user's email address [Required]
 * name - The user's full name [Required]
 * notify
   * billing 
 * teams - List of the nsone_team s to attach to this user's permissions.

### Outputs

  * id - The internal NSOne id of this user.
  * FIXME - Add current registration/login status?

## nsone_apikey

### Inputs

N.B. This *also* has all the inputs from the nsone_team resource, which you can use *instead* of assigning to key to one or more teams.

 * teams - List of the nsone_team s to attach to this API key's permissions.

### Outputs

 * key - The API key that has been generated.
 * id - The internal NSOne id of this api key.

## nsone_team

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

  * id - The internal NSOne id of this team.

# Support / contributions

I'm planning to continue developing and supporting this code for my use-cases,
and I'll do my best to respond to issues and pull requests (and I'm happy to take
patches to improve the code or add missing feaures!).

Also, please be warned that I am *not* a competent Go programmer, so please expect
to find hideous / insane / non-idiomatic code if you look at the source. I'd be
extremely happy to accept patches or suggestions from anyone more experience than me:)

Please *feel free* to contract me via github issues or Twitter or irc in #terraform (t0m)
if you have *any* issues with, or would like advice on using this code.

# Copyright

Copyright (c) Tomas Doran 2015

# LICENSE

Apache2 - see the included LICENSE file for more information

