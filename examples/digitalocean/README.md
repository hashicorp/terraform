# Digital Ocean Droplet launch and setting the Domain records at Digital ocean.

The example launches a Ubuntu 14.04, runs apt-get update and installs nginx. Also demostrates how to create DNS records under Domains at DigitalOcean. 

To run, configure your Digital Ocean provider as described in https://www.terraform.io/docs/providers/digitalocean/index.html

## Prerequisites
You need to export you DigitalOcean API Token as an environment variable

export DIGITALOCEAN_TOKEN="Put Your Token Here" 

## Run this example using:

    terraform plan
    terraform apply 
