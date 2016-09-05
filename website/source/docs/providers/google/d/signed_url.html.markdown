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

```hcl
data "google_storage_object_signed_url" "artifact" {
  bucket = "install_binaries"
  path   = "path/to/install_file.bin"

}

resource "google_compute_instance" "vm" {
    name = "vm"
    ...
    
    provisioner "remote-exec" {
        inline = [
                "wget '${data.google_storage_object_signed_url.artifact.signed_url}' -O install_file.bin",
                "chmod +x install_file.bin",
                "./install_file.bin"
                ]
     }
}
```

## Full Example

```hcl
data "google_storage_object_signed_url" "get_url" {
  bucket       = "fried_chicken"
  path         = "path/to/file"
  content_md5  = "pRviqwS4c4OTJRTe03FD1w=="
  content_type = "text/plain"
  duration     = "2d"
  credentials  = "${file("path/to/credentials.json")}"
  
  extension_headers {
    x-goog-if-generation-match = 1
  }
}
```

## Argument Reference

The following arguments are supported:

* `bucket` - (Required) The name of the bucket to read the object from
* `path` - (Required) The full path to the object inside the bucket
* `http_method` - (Optional) What HTTP Method will the signed URL allow (defaults to `GET`)
* `duration` - (Optional) For how long shall the signed URL be valid (defaults to 1 hour - i.e. `1h`). 
     See [here](https://golang.org/pkg/time/#ParseDuration) for info on valid duration formats.
* `credentials` - (Optional) What Google service account credentials json should be used to sign the URL. 
     This data source checks the following locations for credentials, in order of preference: data source `credentials` attribute, provider `credentials` attribute and finally the GOOGLE_APPLICATION_CREDENTIALS environment variable.
     
> **NOTE** the default google credentials configured by `gcloud` sdk or the service account associated with a compute instance cannot be used, because these do not include the private key required to sign the URL. A valid `json` service account credentials key file must be used, as generated via Google cloud console. 
     
* `content_type` - (Optional) If you specify this in the datasource, the client must provide the `Content-Type` HTTP header with the same value in its request.
* `content_md5` - (Optional) The [MD5 digest](https://cloud.google.com/storage/docs/hashes-etags#_MD5) value in Base64.
     Typically retrieved from `google_storage_bucket_object.object.md5hash` attribute.
     If you provide this in the datasource, the client (e.g. browser, curl) must provide the `Content-MD5` HTTP header with this same value in its request.
* `extension_headers` - (Optional) As needed. The server checks to make sure that the client provides matching values in requests using the signed URL. 
     Any header starting with `x-goog-` is accepted but see the [Google Docs](https://cloud.google.com/storage/docs/xml-api/reference-headers) for list of headers that are supported by Google.
    

## Attributes Reference

The following attributes are exported:

* `signed_url` - The signed URL that can be used to access the storage object without authentication.
