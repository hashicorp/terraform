# go-backblaze
[![GoDoc](https://godoc.org/gopkg.in/kothar/go-backblaze.v0?status.svg)](https://godoc.org/gopkg.in/kothar/go-backblaze.v0)
[![Build Status](https://travis-ci.org/kothar/go-backblaze.svg)](https://travis-ci.org/kothar/go-backblaze)

A golang client for Backblaze's B2 storage

## Usage

Some simple examples to get you started. Errors are ommitted for brevity

Import the API package
~~~
import "gopkg.in/kothar/go-backblaze.v0"
~~~

Create an API client
~~~
b2, _ := backblaze.NewB2(backblaze.Credentials{
  AccountID:      accountID,
  ApplicationKey: applicationKey,
})
~~~

Create a bucket
~~~
bucket, _ := b2.CreateBucket("test_bucket", backblaze.AllPrivate)
~~~

Uploading a file
~~~
reader, _ := os.Open(path)
name := filepath.Base(path)
metadata := make(map[string]string)

file, _ := bucket.UploadFile(name, metadata, reader)
~~~

All API methods except `B2.AuthorizeAccount` and `Bucket.UploadHashedFile` will
retry once if authorization fails, which allows the operation to proceed if the current
authorization token has expired.

To disable this behaviour, set `B2.NoRetry` to `true`

## b2 command line client

A test applicaiton has been implemented using this package, and can be found in the /b2 directory.
It should provide you with more examples of how to use the API in your own applications.

To install the b2 command, use:

`go install gopkg.in/kothar/go-backblaze.v0/b2`

~~~
$ b2 --help
Usage:
  b2 [OPTIONS] <command>

Application Options:
      --account= The account ID to use [$B2_ACCOUNT_ID]
      --appKey=  The application key to use [$B2_APP_KEY]
  -b, --bucket=  The bucket to access [$B2_BUCKET]
  -d, --debug    Debug API requests
  -v, --verbose  Display verbose output

Help Options:
  -h, --help     Show this help message

Available commands:
  createbucket  Create a new bucket
  delete        Delete a file
  deletebucket  Delete a bucket
  get           Download a file
  list          List files in a bucket
  listbuckets   List buckets in an account
  put           Store a file
~~~

## Links

* GoDoc: [https://godoc.org/gopkg.in/kothar/go-backblaze.v0](https://godoc.org/gopkg.in/kothar/go-backblaze.v0)
* Originally based on pH14's work on the API: [https://github.com/pH14/go-backblaze](https://github.com/pH14/go-backblaze)
