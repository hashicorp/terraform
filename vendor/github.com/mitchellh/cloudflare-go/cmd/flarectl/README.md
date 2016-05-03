# flarectl

A CLI application for interacting with a CloudFlare account.

# Usage

You must set your API key and account email address in the environment variables `CF_API_KEY` and `CF_API_EMAIL`.

```
$ export CF_API_KEY=abcdef1234567890
$ export CF_API_EMAIL=someone@example.com
$ flarectl
NAME:
   flarectl - CloudFlare CLI

USAGE:
   flarectl [global options] command [command options] [arguments...]

VERSION:
   2015.12.0

COMMANDS:
   user, u	User information
   zone, z	Zone information
   dns, d	DNS records
   railgun, r	Railgun information
   help, h	Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h		show help
   --version, -v	print the version
```


