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
      }
      answers {
        answer = "2.2.2.2"
        meta {
          feed = "${nsone_datafeed.exampledc2.id}"
          field = "up"
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
    * A, MX and CNAME records are supported.
    * Other record types MAY work, but are untested.
    * Allows records to be linked to other records
    * Allows multiple answers, each of which can be linked to a data feed

## Data sources
    * Can create datasources with arbitrary config
    * This *should* work for all datasource types, but only nsone_v1 is tested

## Data feeds
    * Create data feeds linked to a data source with a label

# Unsupported features

# Zones
  * Setting up secondary servers to AXFR is currently unsupported

## Records
  * Static metadata (not linked to a feed) is not yet supported
  * Filter chains are currently unsupported (Terraform will ignore them if present however - so you can set these up manually)

## NSOne monitoring
  * No support

## Users / Account management / API keys
  * No support

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

