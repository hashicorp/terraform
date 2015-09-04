# Google Compute Engine VPN Example

This example joins two GCE networks via VPN. The firewall rules have been set up
so that you can create an instance in each network and have them communicate
using their internal IP addresses. 

See this [example](https://cloud.google.com/compute/docs/vpn) for more 
information.

Run this example using 

```
terraform apply \
	-var="region1=us-central1" \
	-var="region2=europe-west1" \
	-var="project=my-project-id-123" 
```
