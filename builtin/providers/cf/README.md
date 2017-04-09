# Terraform Cloud Foundry Provider Dev Docs

This document is in place for developer documentation. User documentation is located [HERE](https://www.terraform.io/docs/providers/cloudfoundry/) on Terraform's website. The project backlog is maintained here by [Pivotal Tracker](https://www.pivotaltracker.com/n/projects/1913977).

## Running the Acceptance Tests

The acceptance tests for this provider require admin access to a running Cloud Foundry.
You can setup a local instance of Cloud Foundry to run the acceptance test using 
[PCFDev](https://github.com/pivotal-cf/pcfdev), which can also be download from the 
[Pivotal Network](https://network.pivotal.io/products/pcfdev). To run the acceptance
tests the provider attributes must be exported as follows.

```
export CF_API_URL=https://api.local.pcfdev.io
export CF_USER=admin
export CF_PASSWORD=admin
export CF_UAA_CLIENT_ID=admin
export CF_UAA_CLIENT_SECRET=admin-client-secret
export CF_SKIP_SSL_VALIDATION=true
```

The following two environment variables enable DEBUG logging as well as a dumping of Cloud Foundry API requests  and responses.

```
export CF_DEBUG=true
export CF_TRACE=true
```
