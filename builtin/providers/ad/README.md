This is the Active Directory provider. It has Computer resource which supports,
1. Adding the computer in AD
2. Deleting the compute from AD

In order to use this provider,
1. Copy ad folder to $GOPATH/src/github.com/hashicorp/terraform/builtin/providers

2 .Download the dependancies

    i. go get gopkg.in/ldap.v2
    ii. go get gopkg.in/asn1-ber.v1

3. To build the terraform run 

    make dev

This will add the ad to the list of builin providers.

The sample .tf file show how to use this provider and resource.