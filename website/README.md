# Terraform Documentation

This directory contains the portions of [the Terraform website](https://www.terraform.io/) that pertain to the
core functionality, excluding providers and the overall configuration.

The files in this directory are intended to be used in conjunction with
[the `terraform-website` repository](https://github.com/hashicorp/terraform-website), which brings all of the
different documentation sources together and contains the scripts for testing and building the site as
a whole.

## Previewing Changes

You should preview all of your changes locally before creating a pull request. The build includes content from this repository and the `terraform-website` repository, allowing you to preview the entire Terraform documentation site.

**Set Up Local Environment**

1. [Install Docker](https://docs.docker.com/get-docker/).
1. Create a `~/go` directory manually or by [installing Go](https://golang.org/doc/install).
1. Open terminal and set `GOPATH` as an environment variable:

    Bash: `export $GOPATH=~/go`(bash)

    Zsh: `echo -n 'export GOPATH=~/go' >> ~/.zshrc`
1. Restart your terminal or command line session.

**Launch Site Locally**

1. Navigate into your local `terraform` top-level directory and run `make website`.
3. Open `http://localhost:4567` in your web browser. While the preview is running, you can edit pages and Middleman will automatically rebuild them.
4. When you're done with the preview, press ctrl-C in your terminal to stop the server.
