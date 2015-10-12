# Basic OpenStack architecture with networking

This provides a template for running a simple architecture on an OpenStack
cloud.

To simplify the example, this intentionally ignores deploying and
getting your application onto the servers. However, you could do so either via
[provisioners](https://www.terraform.io/docs/provisioners/) and a configuration
management tool, or by pre-baking configured images with
[Packer](http://www.packer.io).

After you run `terraform apply` on this configuration, it will output the
floating IP address assigned to the instance. After your instance started,
this should respond with the default nginx web page.

First set the required environment variables for the OpenStack provider by
sourcing the [credentials file](http://docs.openstack.org/cli-reference/content/cli_openrc.html).

```
source openrc
```

Afterwards run with a command like this:

```
terraform apply \
  -var 'external_gateway=c1901f39-f76e-498a-9547-c29ba45f64df' \
  -var 'pool=public'
```

To get a list of usable floating IP pools run this command:

```
$ nova floating-ip-pool-list
+--------+
| name   |
+--------+
| public |
+--------+
```

To get the UUID of the external gateway run this command:

```
$ neutron net-show FLOATING_IP_POOL
+---------------------------+--------------------------------------+
| Field                     | Value                                |
+---------------------------+--------------------------------------+
| admin_state_up            | True                                 |
| id                        | c1901f39-f76e-498a-9547-c29ba45f64df |
| mtu                       | 0                                    |
| name                      | public                               |
| port_security_enabled     | True                                 |
| provider:network_type     | vxlan                                |
| provider:physical_network |                                      |
| provider:segmentation_id  | 1092                                 |
| router:external           | True                                 |
| shared                    | False                                |
| status                    | ACTIVE                               |
| subnets                   | 42b672ae-8d51-4a18-a028-ddae7859ec4c |
| tenant_id                 | 1bde0a49d2ff44ffb44e6339a8cefe3a     |
+---------------------------+--------------------------------------+
```
