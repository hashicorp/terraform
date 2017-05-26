# Internal Load Balancing in Google Cloud

This provides a template for setting up internal load balancing in Google Cloud. It directly mirrors the tutorial in the [GCP Internal Load Balancing Documentation](https://cloud.google.com/compute/docs/load-balancing/internal/).

To run the example,

* Log in to gcloud with an account that has permission to create the necessary resources using `gcloud init`.
* Optionally update `variables.tf` to specify a default value for the `project_name` variable, and check other variables.
* Run with a command like this:

```
terraform apply \
	-var="region=us-central1" \
	-var="region_zone=us-central1-b" \
	-var="region_zone_2=us-central1-c" \
	-var="project_name=my-project-id-123" \
```


After you run `terraform apply` on this configuration, it will
automatically output the internal IP address of the load balancer.

Since the load balancer is only reachable from within the network, ssh into the standalone instance using

```
gcloud compute ssh --zone us-central1-b standalone-instance-1
```


Using `curl` on the IP address given, the LB should respond with a simple header:

```html
<!doctype html><html><body><h1>ilb-instance-X</h1></body></html>
```
