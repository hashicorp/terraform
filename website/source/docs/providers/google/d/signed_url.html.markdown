---
layout: "google"
page_title: "Google: google_storage_object_signed_url"
sidebar_current: "docs-google-datasource-signed_url"
description: |-
    Provides signed URL to Google Cloud Storage object.
---

# google\_storage\_object\_signed_url

The Google Cloud storage signed URL data source generates a signed URL for a given storage object. Signed URLs provide a way to give time-limited read or write access to anyone in possession of the URL, regardless of whether they have a Google account.

For more info about signed URL's is available [here](https://cloud.google.com/storage/docs/access-control/signed-urls).

## Example Usage

```
data "google_storage_object_signed_url" "artifact" {
  bucket = "install_binaries"
  path   = "path/to/install_file.bin"

}

resource "google_compute_instance" "vm" {
    name = "vm"
    ...
    
    provisioner "remote-exec" {
        inline = [
                "wget ${data.google_storage_object_signed_url.artifact.signed_url}",
                "chmod +x install_file.bin",
                "./install_file.bin"
                ]
     }
}
```

## Argument Reference

The following arguments are supported:

* `bucket` - (Required) The name of the bucket to read the object from
* `path` - (Required) The full path to the object inside the bucket
* `http_method` - (Optional) What HTTP Method will the signed URL allow (defaults to `GET`)
* `duration` - (Optional) For how long shall the signed URL be valid (defaults to 1 hour `1h`). See [here](https://golang.org/pkg/time/#ParseDuration) for info on valid duration formats.
* `credentials` - (Optional) What Google service account credentials json should be used to sign the URL. This data source checks the following locations for credentials, in order of preference: data source `credentials` attribute, provider `credentials` attribute and finally the GOOGLE_APPLICATION_CREDENTIALS environment variable.
    
> **NOTE** the default google credentials configured by `gcloud` sdk or the service account associated with a compute instance cannot be used, because these do not include the private key required to sign the URL. A valid `json` service account credentials key file must be used, as generated via Google cloud console. 

## Attributes Reference

The following attributes are exported:

* `signed_url` - The signed URL that can be used to access the storage object without authentication.
