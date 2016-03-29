# Basic Two-Tier Architecture in Google Cloud

This provides a template for running a simple two-tier architecture on Google Cloud.
The premise is that you have stateless app servers running behind
a load balancer serving traffic.

To simplify the example, this intentionally ignores deploying and
getting your application onto the servers. However, you could do so either via
[startup script](http://terraform.io/docs/providers/google/r/compute_instance.html#metadata_startup_script) or
[provisioners](https://www.terraform.io/docs/provisioners/) and a configuration
management tool, or by pre-baking configured images with
[Packer](https://packer.io/docs/builders/googlecompute.html).

You will need to generate SSH keys as follows:

```sh
$ ssh-keygen -f ~/.ssh/gcloud_id_rsa
# press <Enter> when asked (twice) for a pass-phrase
```

Then [download your credentials from Google Cloud Console](https://www.terraform.io/docs/providers/google/#credentials); suggested path for downloaded file is `~/.gcloud/Terraform.json`.

Optionally update `variables.tf` to specify a default value for the `project_name` variable, and check other variables.

After you run `terraform apply` on this configuration, it will
automatically output the public IP address of the load balancer.
After your instance registers, the LB should respond with a simple header:

```html
<h1>Welcome to instance 0</h1>
```

The index may differ once you increase `count` of `google_compute_instance`
(i.e. provision more instances).

To run, configure your Google Cloud provider as described in

https://www.terraform.io/docs/providers/google/index.html

Run with a command like this:

```
terraform apply \
	-var="region=us-central1" \
	-var="region_zone=us-central1-f" \
	-var="project_name=my-project-id-123" \
	-var="credentials_file_path=~/.gcloud/Terraform.json" \
	-var="public_key_path=~/.ssh/gcloud_id_rsa.pub" \
	-var="private_key_path=~/.ssh/gcloud_id_rsa"
```
