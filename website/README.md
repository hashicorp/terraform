# Terraform Website

This subdirectory contains the entire source for the [Terraform Website](https://www.terraform.io/).
This is a [Middleman](http://middlemanapp.com) project, which builds a static
site from these source files.

## Contributions Welcome!

If you find a typo or you feel like you can improve the HTML, CSS, or
JavaScript, we welcome contributions. Feel free to open issues or pull
requests like any normal GitHub project, and we'll merge it in.

## Running the Site Locally

To run the site locally, clone this repository and run:

```shell
$ make website
```

You must have Docker installed for this to work.

Alternatively, you can manually run the website like this:

```shell
$ bundle
$ bundle exec middleman server
```

Then open up `http://localhost:4567`.
