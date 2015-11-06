# Terraform Website

This subdirectory contains the entire source for the [Terraform Website](http://www.terraform.io).
This is a [Middleman](http://middlemanapp.com) project, which builds a static
site from these source files.

## Contributions Welcome

If you find a typo or you feel like you can improve the HTML, CSS, or
JavaScript, we welcome contributions. Feel free to open issues or pull
requests like any normal GitHub project, and we'll merge it in.

## Running the Site Locally

Running the site locally is simple. First you need a working copy of [Ruby >= 2.0](https://www.ruby-lang.org/en/downloads/) and [Bundler](http://bundler.io/). Then you can clone this repo and run `make dev`.

Then open up `http://localhost:4567`. Note that some URLs you may need to append
".html" to make them work (in the navigation).
