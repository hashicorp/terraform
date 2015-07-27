# Example

    provider "nsone" {
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

