---
layout: "enterprise"
page_title: "GitLab - VCS Integrations - Terraform Enterprise"
sidebar_current: "docs-enterprise-vcs-gitlab"
description: |-
  GitLab.com, GitLab Community, and GitLab Enterprise repositories can be integrated with Terraform Enterprise by using push command.
---

# GitLab.com, GitLab Community, & GitLab Enterprise

GitLab can be used to import Terraform configuration, automatically
queuing runs when changes are merged into a repository's default branch.
Additionally, plans are run when a pull request is created or updated. Terraform
Enterprise will update the pull request with the result of the Terraform plan
providing quick feedback on proposed changes.

## Registering an OAuth Application & Client

### Creating and Updating an GitLab OAuth Application

You will need to register Terraform Enterprise as an OAuth Application within your GitLab account.

Proceed to https://gitlab.com/profile/applications. Fill out the required information and set the `Redirect URI` to a placeholder (i.e. http://example.com), as you will need to register the GitLab OAuth Client with Terraform Enterprise before it can give you this value.

When you save the form, you will be redirected to the OAuth Application view. Copy your Application Key and Secret as you will need them to connect GitLab to Terraform Enterprise.


### Creating a Terraform Enterprise OAuth Client

In a new tab, navigate to https://atlas.hashicorp.com/settings and in the left-side panel, click on the organization that you’d like to administer your GitLab connection, then click on “Configuration” in the left-side panel.

In the “Add OAuthClient” pane, select your GitLab installation type (GitLab.com, GitLab Community Edition, or GitLab Enterprise) and fill in your application key and secret. In the base URL field, enter the root URL of your GitLab instance (i.e. https://gitlab.com for GitLab.com). In the API URL field, enter the base API URL (i.e. https://gitlab.com/api/v3 for GitLab.com). Create the OAuth client.

Once you have created your client, you will be redirected back to the configurations page for your chosen organization. On that page, find the “OAuth Clients” pane and copy the `Callback URL` for your GitLab OAuth Client. In a new tab, navigate back to https://gitlab.com/profile/applications select the terraform-enterprise OAuth Application and click edit. Enter the `Callback URL` you just copied in the field labeled `Redirect URI`. Save the application.

Your OAuth Client should now be enabled for your Organization to use within Terraform Enterprise.

## Using Terraform Enterprise with GitLab

There are two ways to connect your preferred VCS Host to Terraform Enterprise. You can generate an OAuth token both at the user and organization level.

### Linking your Terraform Enterprise Organization

Return to the settings page for the organization in which you created the OAuth Client (https://atlas.hashicorp.com/settings/organizations/your-organization/configuration). Find the section entitled `Organization Connections to OAuth Client` and click connect beneath your GitLab installation. You will be briefly redirected to GitLab in order to authenticate the client. Once you are redirected back to Terraform Enterprise, you should see that the token was created with a unique identifier. There is also an option to destroy the token and disconnect the organization from your preferred GitLab installation. You are now ready to use your organization's token to manage builds and configurations within Terraform Enterprise.

### Linking your Terraform Enterprise User Account

Navigate to https://atlas.hashicorp.com/settings/connections and click on “Connect “GitLab.com to Atlas”. You will briefly be redirected to GitLab in order to authenticate your OAuth Client. Once redirected back to Terraform Enterprise, You should see a green flash banner with the message: "Successfully Linked to GitLab".

## Connecting Configurations

Once you have linked a GitLab installation to your account or organization,
you are ready to begin creating Packer Builds and Terraform Environments linked
to your desired GitLab repository.

Terraform Enterprise environments are linked to individual GitLab  repositories.
However, a single GitLab repository can be linked to multiple environments
allowing a single set of Terraform configuration to be used across multiple
environments.

Environments can be linked when they're initially created using the New
Environment process. Existing environments can be linked by setting GitLab
details in their **Integrations**.

To link a Terraform Enterprise environment to a GitLab repository, you need
three pieces of information:

- **GitLab repository** - The location of the repository being imported in the
format _username/repository_.

- **GitLab branch** - The branch from which to ingress new versions. This
defaults to the value GitLab  provides as the default branch for this repository.

- **Path to directory of Terraform files** - The repository's subdirectory that
contains its terraform files. This defaults to the root of the repository.

### Connecting a GitLab Repository to a Terraform Environment

Navigate to https://atlas.hashicorp.com/configurations/import and select Link to GitLab.com (or your preferred GitLab installation). A Menu will appear asking you to name the environment. Then use the autocomplete field for repository and select the repository for which you'd like to create a webhook & environment. If necessary, fill out information about the VCS branch to pull from as well as the directory where the Terraform files live within the repository. `Click Create and Continue`.

Upon success, you will be redirected to the environment's runs page (https://atlas.hashicorp.com/terraform/your-organization/environments/your-environment/changes/runs). A message will display letting you know that the repository is ingressing from GitLab and once finished you will be able to Queue, Run, & Apply a Terraform Plan. Depending on your webhook settings, changes will be triggered through git events on the specified branch. The events currently supported are repository and branch push, merge request, and merge.

### Connecting a GitLab Repository to a Packer Build Configuration

Navigate to https://atlas.hashicorp.com/builds/new and select the organization for which you'd like to create a build configuration. Name your build & select `Connect build configuration to a Git Repository`. A form will appear asking you to select your Git Host. Select your preferred GitLab integration. Choose the repository for which you'd like to create a webhook. Fill out any other information in the form such as preferred branch to build from (your default branch will be selected should this field be left blank), Packer directory, and Packer Template.

Upon clicking `Create` you will be redirected to the build configuration (https://atlas.hashicorp.com/packer/your-organization/build-configurations/your-build-configuration). On this page, you will have the opportunity to make any changes to your packer template, push changes via the CLI, or manually queue a Packer build. Depending on your webhook settings, changes will be triggered through git events on the specified branch. The events currently supported are repository and branch push, merge request, and merge.
