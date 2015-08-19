# 'simple' example of 2 regions (us-east-1 and us-west-1) balancing traffic
# between both, with geotargeting (so requests go to the 'nearest' datacenter),
# monitoring (so that if a dc fails, it gets disabled), and an API endpoint you
# can push data to to do load shedding.

# More complex configs than this are possible, but this is near enough to
# a real case to be interesting.

# Additional comments inline.

provider "nsone" {
}

variable "tld" {
    default = "terraform.testing.example"
}

resource "nsone_datasource" "api" {
    name = "terraform_example_api"
    sourcetype = "nsone_v1"
}

output datafeed-uri {
    value = "https://api.nsone.net/v1/feed/${nsone_datasource.api.id}"
}

# Setup the datacenters as equal capacity:
# curl -X POST -H 'X-NSONE-Key: ...' -d '{"useast1": {"high_watermark":100, "low_watermark": 70}' $(terraform output datafeed-uri)
# curl -X POST -H 'X-NSONE-Key: ...' -d '{"uswest1": {"high_watermark":100, "low_watermark": 70}' $(terraform output datafeed-uri)
# Scale useast1 up?
# curl -X POST -H 'X-NSONE-Key: ...' -d '{"useast1": {"high_watermark":200, "low_watermark": 150}' $(terraform output datafeed-uri)
# You get load shedding when you push a connection count above the low watermark:
# curl -X POST -H 'X-NSONE-Key: ...' -d '{"uswest1": {"connections": 75}' $(terraform output datafeed-uri)

resource "nsone_datasource" "monitoring" {
    name = "terraform_example_monitoring"
    sourcetype = "nsone_monitoring"
}

resource "nsone_datafeed" "uswest1_feed" {
    name = "uswest1_feed"
    source_id = "${nsone_datasource.api.id}"
    config {
      label = "uswest1"
    }
}

resource "nsone_datafeed" "useast1_feed" {
    name = "useast1_feed"
    source_id = "${nsone_datasource.api.id}"
    config {
      label = "useast1"
    }
}

resource "nsone_zone" "tld" {
    zone = "${var.tld}"
    ttl = 60
}

# In real life, you may want to get the ELB names from other terraform config, for example:
# resource "terraform_remote_state" "uswest1" {
#    backend = "_local"
#    config {
#        path = "../uswest1/terraform.tfstate"
#    }
#  }
#
# and then below:
# answer = "${terraform_remote_state.uswest1.output.elb_dns_name}"

resource "nsone_record" "www" {
    zone = "${nsone_zone.tld.zone}"
    domain = "www.${var.tld}"
    type = "ALIAS"
    answers {
      answer = "example-elb-uswest1.aws.amazon.com"
      region = "uswest"
      meta {
        field = "up"
        feed = "${nsone_datafeed.uswest1_monitoring.id}"
      }
      meta {
        field = "high_watermark"
        feed = "${nsone_datafeed.uswest1_feed.id}"
      }
      meta {
        field = "low_watermark"
        feed = "${nsone_datafeed.uswest1_feed.id}"
      }
      meta {
        field = "connections"
        feed = "${nsone_datafeed.uswest1_feed.id}"
      }
    }
    answers {
      answer = "example-elb-useast1.aws.amazon.com"
      region = "useast"
      meta {
        field = "up"
        feed = "${nsone_datafeed.useast1_monitoring.id}"
      }
      meta {
        field = "high_watermark"
        feed = "${nsone_datafeed.useast1_feed.id}"
      }
      meta {
        field = "low_watermark"
        feed = "${nsone_datafeed.useast1_feed.id}"
      }
      meta {
        field = "connections"
        feed = "${nsone_datafeed.useast1_feed.id}"
      }
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
        disabled = true # This disables the nsone monitoring from setting things as down
    }
    filters {
        filter = "shed_load"
        config {
            metric = "connections"
        }
    }
    filters {
        filter = "geotarget_regional"
    }
    filters {
      filter = "select_first_region"
    }
    filters {
        filter = "shuffle"
    }
    filters {
        filter = "select_first_n"
        config {
          N = 3
        }
    }
}

resource "nsone_monitoringjob" "useast" {
    name = "useast"
    active = true
    regions = [ "lga" ] # You may want more regions than this if you pay nsone money
    job_type = "tcp"
    frequency = 60
    rapid_recheck = true
    policy = "quorum" # Doesn't take effect until you monitor from 3 regions
    config {
        send = "HEAD / HTTP/1.0\r\n\r\n"
        port = 80
        host = "example-elb-useast1.aws.amazon.com"
    }
    rules {
        value = "200 OK"
        comparison =  "contains"
        key = "output"
    }
}

resource "nsone_monitoringjob" "uswest" {
    name = "uswest"
    active = true
    regions = [ "sjc" ]
    job_type = "tcp"
    frequency = 60
    rapid_recheck = true
    policy = "quorum"
    config {
        send = "HEAD / HTTP/1.0\r\n\r\n"
        port = 80
        host = "example-elb-uswest1.aws.amazon.com"
    }
    rules {
        value = "200 OK"
        comparison =  "contains"
        key = "output"
    }
}

resource "nsone_datafeed" "uswest1_monitoring" {
    name = "uswest1_monitoring"
    source_id = "${nsone_datasource.monitoring.id}"
    config {
      jobid = "${nsone_monitoringjob.uswest.id}"
    }
}

resource "nsone_datafeed" "useast1_monitoring" {
    name = "useast1_monitoring"
    source_id = "${nsone_datasource.monitoring.id}"
    config {
      jobid = "${nsone_monitoringjob.useast.id}"
    }
}

