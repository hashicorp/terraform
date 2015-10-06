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

resource "nsone_datasource" "monitoring" {
    name = "terraform_example_monitoring"
    sourcetype = "nsone_monitoring"
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
        ssl = true
    }
    rules {
        value = "200 OK"
        comparison =  "contains"
        key = "output"
    }
}

