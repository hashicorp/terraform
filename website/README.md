# Terraform Documentation

This directory contains the portions of [the Terraform website](https://www.terraform.io/) that pertain to the core functionality, excluding providers and the overall configuration.

The files in this directory are intended to be used in conjunction with
[the `terraform-website` repository](https://github.com/hashicorp/terraform-website), which brings all of the
different documentation sources together and contains the scripts for testing and building the site as
a whole.

## Suggesting Changes

You can [submit an issue](https://github.com/hashicorp/terraform/issues/new/choose) with documentation requests or submit a pull request with suggested changes.

Click **Edit this page** at the bottom of any Terraform website page to go directly to the associated markdown file in GitHub.

## Modifying Sidebar Navigation

Updates to the sidebar navigation of Terraform docs need to be made in the [`terraform-website`](https://github.com/hashicorp/terraform-website/) repository (preferrably in a PR also updating the submodule commit). You can read more about how to make modifications to the navigation in the [README for `terraform-website`](https://github.com/hashicorp/terraform-website#editing-navigation-sidebars).

## Previewing Changes

You should preview all of your changes locally before creating a pull request. The build includes content from this repository and the [`terraform-website`](https://github.com/hashicorp/terraform-website/) repository, allowing you to preview the entire Terraform documentation site.

**Set Up Local Environment**

1. [Install Docker](https://docs.docker.com/get-docker/).
2. Create a `~/go` directory manually or by [installing Go](https://golang.org/doc/install).
3. Open terminal and set `GOPATH` as an environment variable:

   Bash: `export $GOPATH=~/go`(bash)

   Zsh: `echo -n 'export GOPATH=~/go' >> ~/.zshrc`

4. Restart your terminal or command line session.

**Launch Site Locally**

1. Navigate into your local `terraform` top-level directory and run `make website`.
1. Open `http://localhost:3000` in your web browser. While the preview is running, you can edit pages and Next.js will automatically rebuild them.
1. When you're done with the preview, press `ctrl-C` in your terminal to stop the server.

## Deploying Changes

Merge the PR to main. The changes will appear in the next major Terraform release.

If you need your changes to be deployed sooner, cherry-pick them to:

- the current release branch (e.g. `v1.1`) and push. They will be deployed in the next minor version release (once every two weeks).
- the `stable-website` branch and push. They will be included in the next site deploy (see below). Note that the release process resets `stable-website` to match the release tag, removing any additional commits. So, we recommend always cherry-picking to the version branch first and then to `stable-website` when needed.

Once your PR to `stable-website` is merged, open a PR bumping the submodule commit in [`terraform-website`](https://github.com/hashicorp/terraform-website).

### Deployment

New commits in `hashicorp/terraform` do not automatically deploy the site. Do the following for documentation pull requests:
- **Add a backport label to the PR.** Use the label that corresponds to the latest Terraform patch release (e.g., `1.2-backport`). When you merge your PR to `main`, GitHub bot automatically generates a backport PR to merge your commits into the appropriate release branch.
- **Merge the backport PR.** When all tests pass successfully, merge the backport PR into the release branch. The new content will be added to the site during the next minor release.
- **Cherry-pick changes to `stable-website`.** If you want your changes to show up immediately, check out the latest version of the`stable-website` branch, cherry-pick your changes, and run `git push` to add your changes to the remote `stable-website` branch. Your changes will be live on the site within the hour. 
