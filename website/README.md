# Terraform Documentation

This directory contains the portions of [the Terraform website](https://www.terraform.io/) that pertain to the core functionality, excluding providers and the overall configuration.

The website uses the files in this directory in conjunction with
[the `terraform-website` repository](https://github.com/hashicorp/terraform-website). The `terraform-website` repository brings all of the documentation together and contains the scripts for testing and building the entire site.

## Suggesting Changes

You can [submit an issue](https://github.com/hashicorp/terraform/issues/new/choose) with documentation requests or submit a pull request with suggested changes.

Click **Edit this page** at the bottom of any Terraform website page to go directly to the associated markdown file in GitHub.

## Validating Content

Content changes are automatically validated against a set of rules as part of the pull request process. If you want to run these checks locally to validate your content before committing your changes, you can run the following command:

```
npm run content-check
```

If the validation fails, actionable error messages will be displayed to help you address detected issues.

## Modifying Sidebar Navigation

You must update the sidebar navigation when you add or delete documentation .mdx files. If you do not update the navigation, the website deploy preview fails.

To update the sidebar navigation, you must edit the appropriate `nav-data.json` file. This repository contains the sidebar navigation files for the following documentation sets:

- Terraform Language: [`language-nav-data.json`](https://github.com/hashicorp/terraform/blob/main/website/data/language-nav-data.json)
- Terraform CLI: [`cli-nav-data.json`](https://github.com/hashicorp/terraform/blob/main/website/data/cli-nav-data.json)
- Introduction to Terraform: [`intro-nav-data.json`](https://github.com/hashicorp/terraform/blob/update-readme/website/data/intro-nav-data.json)

For more details about how to update the sidebar navigation, refer to [Editing Navigation Sidebars](https://github.com/hashicorp/terraform-website#editing-navigation-sidebars) in the `terraform-website` repository.

## Adding Redirects

You must add a redirect when you move, rename, or delete documentation pages. Refer to https://github.com/hashicorp/terraform-docs-common#redirects for details.

## Previewing Changes

You should preview all of your changes locally before creating a pull request. The build includes content from this repository and the [`terraform-website`](https://github.com/hashicorp/terraform-website/) repository, allowing you to preview the entire Terraform documentation site.

**Set Up Local Environment**

1. [Install Docker](https://docs.docker.com/get-docker/).
2. [Install Go](https://golang.org/doc/install) or create a `~/go` directory manually.
3. Open terminal and set `GOPATH` as an environment variable:

   Bash: `export $GOPATH=~/go`(bash)

   Zsh: `echo -n 'export GOPATH=~/go' >> ~/.zshrc`

4. Restart your terminal or command line session.

**Launch Site Locally**

1. Navigate into your local `terraform` top-level directory and run `make website`.
1. Open `http://localhost:3000` in your web browser. While the preview is running, you can edit pages and Next.js automatically rebuilds them.
1. Press `ctrl-C` in your terminal to stop the server and end the preview.

## Deploying Changes

Merging a PR to `main` queues up documentation changes for the next minor product release. Your changes are not immediately available on the website.

The website generates versioned documentation by pointing to the HEAD of the release branch for that version. For example, the `v1.2.x` documentation on the website points to the HEAD of the `v1.2` release branch in the `terraform` repository. To update existing documentation versions, you must also backport your changes to that release branch. Backported changes become live on the site within one hour.

### Backporting

**Important:** Editing old versions (not latest) should be rare. We backport to old versions when there is an egregious error. Egregious errors include inaccuracies that could cause security vulnerabilities or extreme inconvenience for users.

Backporting involves cherry-picking commits to one or more release branches within a docs repository. You can backport (cherry-pick) commits to a version branch by adding the associated backport label to your pull request. For example, if you need to add a security warning to the v1.1 documentation, you must add the `1.1-backport` label. When you merge a pull request with one or more backport labels, GitHub Actions opens a backport PR to cherry-pick your changes to the associated release branches. You must manually merge the backport PR to finish backporting the changes.

To make your changes available on the latest docs version:

1. Add the backport label for the latest version.

   <img width="317" alt="Screen Shot 2022-08-09 at 11 06 17 AM" src="https://user-images.githubusercontent.com/83350965/183686586-f94e58f3-fd62-48cf-88bd-fa886fe4724f.png">

1. Merge the pull request. GitHub Actions autogenerates a backport pull request, linked to the original.

   <img width="726" alt="Screen Shot 2022-08-09 at 11 08 52 AM" src="https://user-images.githubusercontent.com/83350965/183687165-350b0e9b-a888-409e-91e2-81d82eac0a4e.png">

1. Merge the auto-generated backport pull request.

   You can review and merge your own backport pull request without waiting for another review if the changes in the backport pull request are effectively equivalent to the original. You can make minor adjustments to resolve merge conflicts, but you should not merge a backport PR that contains major content or functionality changes from the original, approved pull request. If you are not sure whether it is okay to merge a backport pull request, post a comment on the original pull request to discuss with the team.
