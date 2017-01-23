<img src="https://app.arukas.io/images/logo-orca.svg" alt="" width="100" /> Arukas CLI
==========

[![Circle CI](https://circleci.com/gh/arukasio/cli.svg?style=shield)](https://circleci.com/gh/arukasio/cli)

The Arukas CLI is used to manage Arukas apps from the command line.
* Website: https://arukas.io

### Binary Releases

The official binary of Arukas CLI: https://github.com/arukasio/cli/releases/

### Dockerized

A dockerized version of Arukas CLI: https://hub.docker.com/r/arukasio/arukas/

## Setup

* Get API key here: https://app.arukas.io/settings/api-keys
* Edit it `.env` file

You can overload and customize specific variables when running scripts.

Simply create `.env` with the environment variables you need,
for example, `ARUKAS_JSON_API_TOKEN` and `ARUKAS_JSON_API_SECRET`

```
# .env
ARUKAS_JSON_API_TOKEN=YOUR_API_TOKEN
ARUKAS_JSON_API_SECRET=YOUR_API_SECRET
```

You can look at `.env.sample` for other variables used by this application.

## License

This project is licensed under the terms of the MIT license.
