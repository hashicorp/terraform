# Terraform Website

This subdirectory contains the entire source for the [Terraform
Website][terraform]. This is a [Middleman][middleman] project, which builds a
static site from these source files.

## Contributions Welcome!

If you find a typo or you feel like you can improve the HTML, CSS, or
JavaScript, we welcome contributions. Feel free to open issues or pull requests
like any normal GitHub project, and we'll merge it in.

## Running the Site Locally

Running the site locally is simple:

1. Install [Docker](https://docs.docker.com/engine/installation/) if you have not already done so
2. Clone this repo and run `make website`

Then open up `http://localhost:4567`. Note that some URLs you may need to append
".html" to make them work (in the navigation).

[middleman]: https://www.middlemanapp.com
[terraform]: https://www.terraform.io

## Building a Local Copy

Building a local copy (which can be read off the filesystem, rather
than served by a local web server) is somewhat more complicated.

1. Install [Docker](https://docs.docker.com/engine/installation/)
2. Clone this repo
3. run `MM_ENV=local_build make build`

WARNING: In order to avoid accidentally committing huge quantities of
changes, setting `MM_ENV=local_build` will wipe out all changes to
source/ after building, so make sure anything you want to save is
committed.
