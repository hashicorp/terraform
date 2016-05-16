# go-nsone-api

An *experimental* golang client for the NSOne API: https://api.nsone.net/

# Example use

    import (
        "github.com/bobtfish/go-nsone-api"
    )

    func main() {
        api := nsone.New("xxxxxxx")

        ds := nsone.NewDataSource("mydatasource", "nsone_apiv1")
        err := api.CreateDataSource(ds)
        if err != nil {
          panic(err)
        }

        dc1_feed_config := make(map[string]string)
        dc1_feed_config["label"] = "exampledc1"
        df_dc1 := nsone.NewFeed(ds.Id)
        df_dc1.Config = dc1_feed_config
        err = api.CreateDataFeed(df_dc1)
        if err != nil {
            panic(err)
        }    

        dc2_feed_config := make(map[string]string)
        dc2_feed_config["label"] = "exampledc2"
        df_dc2 := nsone.NewFeed(ds.Id)
        df_dc2.Config = dc2_feed_config
        err = api.CreateDataFeed(df_dc2)
        if err != nil {
            panic(err)
        }

        z := nsone.NewZone("foo.com")
        err = api.CreateZone(z)
        if err != nil {
            panic(err)
        }

        r := nsone.NewRecord("foo.com", "www.foo.com", "A")
        answers := make([]nsone.Answer, 2)
        answers[0] = nsone.NewAnswer
        answers[1] = nsone.NewAnswer
        answers[0].Answer = []string{"1.1.1.1"}
        answers[1].Answer = []string{"1.1.1.1"}
        answers[0].Answer.Meta["up"] = nsone.NewMetaFeed(df_dc1.Id)
        answers[1].Answer.Meta["up"] = nsone.NewMetaFeed(df_dc2.Id)
        r.Answers = answers
        err = api.CreateRecord(r)
        if err != nil {
            panic(err)
        }

        api.DeleteZone("foo.com")
    }

# Installing

Just checkout this library to your GOPATH as usual. If you're writing a standard go program
using this library, that should be as simple as saying 'go get'

# Refernce documentation

See [the godoc](http://www.godoc.org/github.com/bobtfish/go-nsone-api)

# Supported features

## Setup zones
    * Links supported
    * Secondary zones supported
    * Metadata *may* be supported, but is untested

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

## NSOne monitoring
    * Retrieve monitoring jobs

# Unsupported features

# Zones
  * Setting up secondary servers is currently unsupported/untested

## Records
  * Static metadata (not linked to a feed) is not yet supported
  * Filter chains are currently unsupported (Terraform will ignore them if present however - so you can set these up manually)

## NSOne monitoring
  * 

## Users / Account management / API keys
  * No support

## Useage / querying APIs
  * No support

# Support

I'm planning to continue developing and supporting this code for my use-cases (which
are those of terraform-provider-nsone). I'll try not to break things I don't have to,
however I'm very likely to have to change some of the structs/functions as I add
the missing functionality and clean up this library.

If you seriously want to use this code then I recommend vendoring it, until I remove the
*experimental* notice as the API has stableized.

# Contributions

I'll do my best to respond to issues and pull requests (and I'm happy to take
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

