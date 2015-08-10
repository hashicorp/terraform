provider "nsone" {
}


variable "tld" {
    default = "terraform.testing.example"
}

resource "nsone_datasource" "api" {
    name = "terraform_example"
    sourcetype = "nsone_v1"
}

resource "nsone_datafeed" "uswest1" {
    name = "uswest1"
    source_id = "${nsone_datasource.api.id}"
    config {
      label = "uswest1"
    }
}

resource "nsone_datafeed" "useast1" {
    name = "useast1"
    source_id = "${nsone_datasource.api.id}"
    config {
      label = "useast1"
    }
}

resource "nsone_zone" "tld" {
    zone = "${var.tld}"
    ttl = 60
}

resource "nsone_record" "www" {
    zone = "${nsone_zone.tld.zone}"
    domain = "www.${var.tld}"
    type = "CNAME" # Note, normally we'd use ALIAS here
    answers {
      answer = "example-elb-uswest1.aws.amazon.com"
      meta {
        field = "high_watermark"
        feed = "${nsone_datafeed.uswest1.id}"
      }
      meta {
        field = "low_watermark"
        feed = "${nsone_datafeed.uswest1.id}"
      }
      meta {
        field = "connections"
        feed = "${nsone_datafeed.uswest1.id}"
      }
    }
    answers {
      answer = "example-elb-useast1.aws.amazon.com"
      meta {
        field = "high_watermark"
        feed = "${nsone_datafeed.useast1.id}"
      }
      meta {
        field = "low_watermark"
        feed = "${nsone_datafeed.useast1.id}"
      }
      meta {
        field = "connections"
        feed = "${nsone_datafeed.useast1.id}"
      }
    }
}

