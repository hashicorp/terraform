# Terraform Documentation

This directory contains the portions of [the Terraform website](https://www.terraform.io/) that pertain to the
core functionality, excluding providers and the overall configuration.

The files in this directory are intended to be used in conjunction with
[the `terraform-website` repository](https://github.com/hashicorp/terraform-website), which brings all of the
different documentation sources together and contains the scripts for testing and building the site as
a whole.

## Previewing Changes

You should preview all of your changes locally before creating a pull request. The build includes content from this repository and the [`terraform-website`](https://github.com/hashicorp/terraform-website/) repository, allowing you to preview the entire Terraform documentation site. If `terraform-website` isn't in your `GOPATH`, the preview command will clone it to your machine.

**Set Up Local Environment**

1. [Install Docker](https://docs.docker.com/get-docker/).
2. Create a `~/go` directory manually or by [installing Go](https://golang.org/doc/install).
3. Open terminal and set `GOPATH` as an environment variable:

    Bash: `export $GOPATH=~/go`(bash)

    Zsh: `echo -n 'export GOPATH=~/go' >> ~/.zshrc`
4. Restart your terminal or command line session.

**Launch Site Locally**

1. Navigate into your local `terraform` top-level directory and run `make website`.
2. Open `http://localhost:4567` in your web browser. While the preview is running, you can edit pages and Middleman will automatically rebuild them.
3. When you're done with the preview, press `ctrl-C` in your terminal to stop the server.

## Deploying Changes

Merge the PR to main. The changes will appear in the next major Terraform release.

If you need your changes to be deployed sooner, cherry-pick them to:
- the current release branch (e.g. `v1.0`) and push. They will be deployed in the next minor version release (once every two weeks).
- the `stable-website` branch and push. They will be included in the next site deploy (see below). Note that the release process resets `stable-website` to match the release tag, removing any additional commits. So, we recommend always cherry-picking to the version branch first and then to `stable-website` when needed.

### Deployment
Currently, HashiCorp uses a CircleCI job to deploy the [terraform.io](terraform.io) site. This job can be run manually by many people within HashiCorp, and also runs automatically whenever a user in the HashiCorp GitHub org merges changes to master in the `terraform-website` repository.

New commits in this repository don't automatically deploy the [terraform.io][] site, but an unrelated site deploy will usually happen within a day. If you can't wait that long, you can do a manual CircleCI build or ask someone in the #proj-terraform-docs channel to do so:
- Log in to circleci.com, and  make sure you're viewing the HashiCorp organization.
- Go to the terraform-website project's list of workflows.
- Find the most recent "website-deploy" workflow, and click the "Rerun workflow from start" button (which looks like a refresh button with a numeral "1" inside).
