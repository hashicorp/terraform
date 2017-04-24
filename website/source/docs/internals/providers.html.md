---
layout: "docs"
page_title: "Writing a Provider"
sidebar_current: "docs-provider-guide"
description: |-
  Information on writing a provider for Terraform.
---

# Writing a Terraform Provider
There are two high-level sections of the Terraform codebase: the core section handles all the graphs, diffing, and essentially turning state and config files into a list of resources that need to be created, updated, or destroyed; the providers handle the actual creation, updating, and destruction of those resources. By dividing things like this, it is much easier to contribute new functionality or new APIs to Terraform: you can essentially just trust the core section to do its job, and only worry about your resource.

One way to think of the separation is that the core tells the providers what to do, and the providers make that change against the APIs.

To hook into this behaviour, a provider framework has been built into Terraform. This guide aims to document that framework, and help guide thinking and understanding around it.

## Understanding the Terraform Lifecycle
Terraform runs follow a predictable lifecycle: gather information, detect diffs, apply updates, set state.

Information is gathered from two places: the config and the state. The config is populated from the user’s config file; the state is populated from the statefile. But before the state gets populated from the statefile, the statefile is refreshed, using information about the provider(s) to get a more accurate picture of the world. It then gets passed through an optional per-resource `StateFunc`, which allows resources to modify their representation in the state. Finally, the config and state get passed through an optional `DiffSuppressFunc`, which allows resources to decide whether a config value and a state value should be considered equivalent. This results in our diff.

<!-- insert diagram here -->

Once we have that diff, we know which resources need to be created, updated, or destroyed. The `Create`, `Update`, and `Destroy` functions for the resources are called as appropriate, and the `ResourceData` instance they modify is persisted as the state when the call returns.

## Providers vs Resources and Data Sources
The three major concepts to understand in the provider framework are **Data Sources**, **Resources**, and **Providers**. Providers are the API provider that is being integrated; for example, Amazon’s Web Services APIs all fall under a single provider, GitHub’s APIs all fall under a single provider, and so on. Resources are the specific API resources that are being manipulated via their own CR(U)D interface; for example, an Instance in Amazon’s EC2 API would be its own resource, a repository in GitHub’s API would be its own resource, and so on. Data Sources are read-only resources that are exposed by an API, that Terraform isn’t managing but still wants to be able to reference. Things like which disk images are available in Google Cloud Platform, for example.

A good way to think of Providers and Resources/Data Sources is that Providers define all the common configuration you need to call an API, while Resources and Data Sources actually make the calls.

## Creating a New Provider
Say you’ve found a new API provider that Terraform doesn’t support, and you want to go ahead and implement it. First of all, thank you! It’s always exciting when the community helps Terraform learn new tricks, and the Terraform team appreciates all the contributions the community has made.

To get started, you’re going to want to look at the [`Provider`](https://godoc.org/github.com/hashicorp/terraform/helper/schema#Provider) type. Don’t worry if it looks complex or confusing; we’re going to break it down and walk through it over the course of this guide. There are a lot of methods, but the good news is it’s not your job to call or define _any_ of them. You can safely ignore them.

### Creating the Package
To define a new provider, you need to create a new package. If you are contributing this to the Terraform codebase, create a directory named after your plugin in the `builtin/providers` directory of Terraform. If you’re planning on maintaining it outside of Terraform as a standalone plugin, create a new repository for it&emdash;we recommend `github.com/{yourname}/terraform-provider-{name}`, as that will simplify things down the road.

In your new package, create a `provider.go` file. In that file, you’re going to create a `Provider` function that takes no arguments and returns a `terraform.ResourceProvider`:

```go
func Provider() terraform.ResourceProvider {
    return &schema.Provider{
        // your provider definition here
    }
}
```

The `*schema.Provider` you’re returning fills the `terraform.ResourceProvider` interface; you don’t have to use it, you could use your own custom implementation of the interface, but that’s tricky and hard to get right, so we highly recommend you use the built-in `*schema.Provider`.

For the moment, just set the properties of the returned `*schema.Provider` to empty maps (leave `ConfigureFunc` as nil). We’ll take care of the properties later; let’s get that `ConfigureFunc` defined.

### Configuring Your Provider
Most API providers take some form of configuration: an API endpoint to use, credentials to use, and so on. To accomplish this, we’re going to set up the `ConfigureFunc` property on your Provider. If an API client needs to be instantiated before it’s used in your Resources, this is the place to do it.

The premise is pretty simple: you’re going to define a function that takes a `ResourceData` struct (more on that [later](TODO)) and returns an `interface{}` or an `error`. If you don’t return an error, the interface you return will be stored and passed to all your resources as a meta parameter. If you do return an error, that will bubble up to the Terraform CLI and abort whatever command is being run. **Note:** please return errors instead of using `panic` or `os.Exit`.

### Registering Your Provider
Because Terraform uses a plugin architecture, we need to register our new plugin with Terraform before Terraform knows how to use it. (Fun fact: even the providers built into the codebase need to be registered!)

If you’re contributing your provider to the Terraform codebase, you need to create a new folder in the `builtin/bins` directory in the Terraform codebase named `provider-{name}`. Inside, you’ll put your `main.go` file. If you’re maintaining your own plugin, you can put the `main.go` file in the root of the repo containing the code for your provider, or really anywhere you want, as long as it’s in package main.

Inside that `main.go` file, you’re going to define a `main` function that serves your plugin. This is pretty straightforward, so check out any of the [built-in plugins](https://github.com/hashicorp/terraform/tree/master/builtin/bins) for an example to follow. The only thing you should need to customise is the import path to the package containing your `Provider` function and the `ProviderFunc` property to point to that `Provider` function.

## Creating a New Resource
Now that you’ve got a Provider, it’s time to add some Resources to that Provider to make it actually _do_ something. To do this, you need to write some code in the package your Provider is defined in.

In the built-in plugins for Terraform that are shipped as part of the codebase, we have a file naming guideline: we use `resource_{resource_name}.go` as the template to name our files, and tend to stick to  single Resource per file. For example, AWS has `resource_aws_instance.go`, because it uses the AWS API to manage the instance resource. If you want to contribute your code back to the Terraform repo, it makes things easier if you follow this template.

Inside this file, you’re going to define a function that takes no arguments and returns a `*schema.Resource`.  While you can technically name your function anything, we recommend (especially if you plan on contributing your code back to the Terraform repo!) that you name it the camelCase version of the file name. For example, `resource_compute_instance.go` would contain `resourceComputeInstance`. We’ll be defining a _lot_ of functions in a Provider, and this kind of naming scheme really helps keep things organised and easy to find.

### Registering the Resource With the Provider
Now that you’ve got your function, switch back to your Provider definition, and add the Resource to the Provider’s `ResourcesMap` property. The key will be the name of the Resource used in state and configuration files, and (by convention) should match the `{resource_name}` part used in the filename for the file that defines the Resource. Nothing will necessarily break if you don’t do this, but if you happen to accidentally have a collision with any other resource in any other provider, things will likely break. This convention helps keep things neat and orderly while avoiding conflicts.

### Defining the Resource Properties
Each Resource has a set of properties called the “schema” that is stored in the state. Think of it as the type definition for the Resource. For example, an AWS instance resource has a `name` property, to control the name of the instance, GitHub’s repository resource has a `default_branch` property to set the default branch for the repository, and so on.

These properties are defined in the `Schema` property of the Resource definition. It takes a map with the keys being the property name and the values being `*Schema` structs. The `*Schema` structs define some type information (what kind of data to expect, etc.) along with some [advanced behaviour](TODO) for resources that helps Terraform do the right thing without you needing to write a bunch of code.

~> **Note:** “id” is a reserved property name. Don’t call your property “id”.

### Calling the API
Now that we have a Resource fully defined, it’s time to make it do something. We’re going to use the Provider’s API client (possibly [configured](TODO) by the Provider’s `ConfigureFunc`) to create, read, update, destroy, and check for the existence of some resources. We do this by defining functions to perform those operations for us. Terraform will then decide which of the functions to call and on which resources. All the functions have to do is know how to determine and set the state of a resource using the Provider’s API.

Each function takes a `*ResourceData` struct and an `interface{}` as arguments, and returns an `error`. The only exception is `ExistsFunc`, which takes the same arguments, but returns a boolean or an `error`.

The `*ResourceData` struct contains the state of the Resource as it should be. It’s an amalgamation of several different sources of data, which are explained further in [Understanding ResourceData](TODO).

The `interface{}` is the same `interface{}` returned by the Provider’s `ConfigureFunc`. Typically, this is where you’d put the configured API client, for example.

Of all these functions, only the `Update` and `Exists` functions are optional. If the `Update` function isn’t set, the Resource will be treated as something that cannot be updated. If the `Exists` function isn’t set, Terraform will not check if the Resource already/still exists.

The `Create` function should use the passed `*ResourceData` to make an API call that will create a Resource in the desired state. It should then update the `*ResourceData` with the state as reported by the server, to detect any drift and fill in any `Computed` properties.

The `Read` function should  retrieve the ID (or other identifying information) of the resource to be read from the passed `*ResourceData`. It should then retrieve the Resource from the API, and set the `*ResourceData`'s fields to match what the server reported. It’s important to note that even if it’s required to set state as user input, the `Read` function should still overwrite that state, so Terraform can understand the current state of the world, not just the desired state.

The `Update` function is called when the state of one of your resources has diverged from the config file, but the resource already exists. If the API supports partial updates of resources (for example, using `PATCH` requests) your update function  can use the passed `*ResourceData`’s `HasChange` method to detect which properties need to be updated, and make the necessary API calls; otherwise, use the `*ResourceData`’s `Get` methods to construct your request, just like when creating a resource. Terraform makes no effort to roll back failed updates, so if your `Update` function makes multiple API calls or performs other half-updates, you should make sure to [set partial state mode](TODO) on the `*ResourceData` and use it carefully.

The `Delete` function should use the passed `*ResourceData` to retrieve the ID (or other identifying information necessary for the API call) of the Resource to be deleted, then make the API call or calls necessary to delete that Resource. Sometimes APIs can be picky about the order Resources are deleted in; Terraform always deletes resources that depend on other resources before deleting the depended upon resources.

The `Exists` function should use the passed `*ResourceData` containing the ID (or other identifying information necessary for the API call) of the Resource to check the existence of, then make the API call necessary to check for its existence. If the Resource exists, the `Exists` function should return true. It’s important that the `Exists` function never modify the passed `*ResourceData`. It’s also important that the `Exists` function always returns errors it runs into while checking whether the Resource exists or not; Resources should only be treated as “gone” if they can’t be found, not if there’s a failed API call.

## Creating a New Data Source
Once we have a Provider, we can add some Data Sources to that provider, which will help it pull information out of your infrastructure to be used in your Resources.  To do this, you need to write some code in the package your Provider is defined in.

In the built-in plugins for Terraform that are shipped as part of the codebase, we have a file naming guideline: we use `data_source_{api}_{data_source_name}.go` as the template to name our files, and tend to stick to  single Data Source per file. For example, AWS has `data_source_aws_acm_certificate.go`, because it reads certificates from AWS’ ACM service. Some providers only have a single API, so we use the provider name in that case. For example, Docker has `data_source_docker_registry_image.go`. If you want to contribute your code back to the Terraform repo, it makes things easier if you follow this convention.

Inside this file, you’re going to define a function that takes no arguments and returns a `*schema.Resource`, just like Resources.  While you can technically name your function anything, we recommend (especially if you plan on contributing your code back to the Terraform repo!) that you name it the camelCase version of the file name. For example, `data_source_aws_eip.go` would contain `dataSourceAwsEip`. We’ll be defining a _lot_ of functions in a Provider, and this kind of naming scheme really helps keep things organised and easy to find.

### Registering the Data Source  With the Provider
Now that you’ve got your function, switch back to your Provider definition, and add the Data Source to the Provider’s `DataSourcesMap` property. The key will be the name of the Data Source used in state and configuration files, and (by convention) should match the `{data_source_name}` part used in the filename for the file that defines the Resource. Nothing will necessarily break if you don’t do this, but if you happen to accidentally have a collision with any other resource in any other provider, things will likely break. This convention helps keep things neat and orderly while avoiding conflicts.

### Defining the Data Source Properties
Each Data Source has a set of properties called the “schema” that is stored in the state, just like Resources do. Think of it as the type definition for the Data Source. For example, an AWS elastic IP data source has a `public_ip` property, to allow configurations to access the IP, Docker’s registry image data source has a `sha256_digest` property to access the checksum of the image, and so on.

These properties are defined in the `Schema` property of the Data Source definition. It takes a map with the keys being the property name and the values being `*Schema` structs. The `*Schema` structs define some type information (what kind of data to expect, etc.) along with some [advanced behaviour](TODO) for resources that helps Terraform do the right thing without you needing to write a bunch of code.

~> **Note:** “id” is a reserved property name. Don’t call your property “id”.

### Calling the API
Now that we have a Data Source fully defined, it’s time to make it do something. We’re going to use the Provider’s API client (possibly [configured](TODO) by the Provider’s `ConfigureFunc`) to read some resources. Though the Data Source, as a `*Resource` type, has `Create`, `Read`, `Update`, `Delete`, and `Exists` properties, only the function defined in the `Read` property will be used. The function takes a `*ResourceData` struct and an `interface{}` as arguments, and returns an `error`. 

The `*ResourceData` struct represents the state of the Resource as it should be. It’s an amalgamation of several different sources of data, which are explained further in [Understanding ResourceData](TODO). For Data Sources, the `*ResourceData` struct is used to pull the identifying information necessary to retrieve the information requested from the API, and as a place to store the information the API returns.

The `interface{}` is the same `interface{}` returned by the Provider’s `ConfigureFunc`. Typically, this is where you’d put the configured API client, for example.

The `Read` function should then use the passed `*ResourceData` to retrieve the ID (or other identifying information necessary for the API call) of the Data Source to be read. It should then retrieve the Data Source from the API, and set the `*ResourceData` to match.

## Resources vs Data Sources
Resources and Data Sources are incredibly similar, and for the most part, can be treated as working the same&mdash;they’re implemented using the same type, `*Resource`. An easy shorthand is to think of Data Sources as read-only Resources. If you find yourself creating a Resource that only has useful behaviour in the Read function, it should probably be a Data Source.

Data Sources shouldn’t be considered part of provisioning an environment&mdash;Terraform will make no effort to ensure they exist before it tries to reference them, and it doesn’t have any concept of how they should be configured. They’re simply a way to allow configurations to reference information that another tool&mdash;your CI tool, your cloud provider, etc.&mdash;owns.

## Working With Asynchronous APIs
Some resources may take time to create—it may take 30 seconds to spin up an instance, or 20 minutes to provision a hosted database instance like RDS. The convention and promise of Terraform is to only consider resource creation or update as done when these pending operations are actually done. Often the API returns one or more fields that allow you to inspect the state of the resource during its creation. The same principle applies to deletion.

This means we intentionally let the user wait rather than provide resources that may not be ready for use yet by the user or some other dependent resources or provisioners.

There are two helpers to make the work with such APIs easier.

### Working with `StateChangeConf`
The `StateChangeConf` function in the `helper/resource` package allows you to watch an object and wait for it to achieve the state specified in the configuration using the `Refresh()` function. It does exponential backoff by default to avoid throttling and it can also deal with inconsistent or eventually consistent APIs to certain extent.

### Working with `Retry`
The `Retry` function in the `helper/resource` package is just a wrapper around the `StateChangeConf` function. It’s useful for situations where you only need to retry, and there are only two states the resource can be in, based on the API response (success or failure).

## Understanding the Schema
The concept of a Schema is pretty central to the way Terraform’s Providers work, so it’s worth exploring some of the advanced tools Providers have at their disposal to help Terraform do the right thing. The idea of the Schema, itself, can be reductively explained as the “type system” of a Terraform Resource or Data Source, but it goes a little deeper than that. It has all the information you’d expect: the type of information (a string, an integer, a list, etc.) that it expects, whether that information is optional or computed, etc.; but it also has a bunch of special features that can be defined for each property.

### Working With Computed
There are some fields on an Resource or Data Source that can’t be influenced by the user at all, which the API provider is the source of truth for. Things like the timestamp a resource was created, or the ID of a resource. For these special properties, Terraform wants to notice when users mistakenly set them in config files and throw an error, instead of silently ignoring them. To make this possible, any Resource or Data Source property that should be stored in state but that the user should not be able to configure should have its `Computed` property set to true.

To complicate things, there are some fields on a Resource that the user _can_ set, but if they don’t, the API will pick a value for them. Things like the IP address assigned to a compute instance, or version of a disk image to use when creating a disk. For these properties, Terraform wants to know that if the config file doesn’t ask for anything specific, the user is happy with whatever the server returns, but if there is something in the config file, Terraform needs to ensure the server reflects that value. To make this possible, any Resource property that the server gets to pick a default for but the user should be able to override should have _both_ its `Computed` and `Optional` properties set to `true`. It’s important to note that if the server does not respect the value the user asks for and generates one on its own, Terraform will consider that a diff, and will keep trying to correct it. This often leads to perpetual diff bugs, so it’s important that only properties the user can actually _set_ have their `Optional` property set to `true`.

### Working With `ForceNew`
Some Resources are completely immutable--if you want to change anything about them, you need to just tear them down and build up again from scratch. Sometimes properties need to be set on Resources when they’re created, and can’t be changed after. For example, you need to decide which region you want a compute instance in before you stand it up, and once you create it, you can’t change it&mdash;you can only tear down that instance and stand up a new one.

To help in this common scenario, Resources properties have a `ForceNew` property that, when set to `true`,  indicates to Terraform that if it notices a diff, it should just tear down the old one and stand up a new one.

### Deprecating and Removing Properties
Things change sometimes, and that’s okay. Sometimes a Provider supports input that can’t be supported later. Sometimes a property needs to be renamed. In these situations, Terraform provides two properties that can optionally be set on any field for a Resource.

If `Deprecated` is set to anything but the empty string, when a config file sets or references that field, Terraform will display that string to the user. This is useful for notifying users that a field is going away or should no longer be used without actually breaking their config. A good value for this property would be a message indicating that the field is deprecated, and pointing the user to suggestions for what to do instead and/or more information.

If `Removed` is set to anything but the empty string, when a config file sets or references that field, Terraform will throw an error, stop execution, and display that string to the user. This is useful for offering users more context or information, instead of a config field just disappearing on them. A good value for this property would be a message indicating that the field has been removed, and pointing the user to suggestions for what to do instead and/or more information.

### Working With Defaults
Information on what Default, DefaultFunc, and InputDefault do, what they mean, and when to use them.

### Working With Sets and Lists
Information on Set and List types and how to use them, along with some of their quirks.

### Understanding `ResourceData`
When working with the `Create`, `Read`, `Update`, `Destroy`, and `Exists` methods on your Provider, it’s almost impossible to not run into `ResourceData`. It’s a type that is used all over, so it’s important to understand it. But `ResourceData` can also be tricky, because it’s not abstracting a single logical concept in Terraform.

#### How to Conceptualise `ResourceData`
The most useful way to think of `ResourceData` is not as a placeholder for any one concept in Terraform—it’s not just information about the config or the state, for example—but as the desired state of an object. If you think of Terraform as the convergence of your config file and your infrastructure, `ResourceData` is how you express  that understanding of what your infrastructure should look like.

#### What `ResourceData` Abstracts
To get practical, `ResourceData` abstracts four separate concepts into a single source of truth:

* The current state
* The config file
* The diff being applied
* Any calls to `ResourceData.Set`

Those are expressed in the order of their priority. That is, the config overrides any values set in the state, and every time you call `Set`, it overrides everything else.

`ResourceData` often gets confused for either the state or the config, but it’s important to realise that it’s a step removed from these concepts, and is used more as an understanding of what the infrastructure _should_ be.

#### Retrieving Properties
To retrieve properties from `ResourceData`, use the `Get` or `GetOk` methods. `Get` takes a property name or address (e.g., `myprop.0.value`) and returns it as an `interface{}`. It’s important to note that the framework will always guarantee consistency about the underlying type of that `interface{}`. If the key doesn’t exist in your schema, `Get` returns `nil`. If the key exists in your schema, but not in the config, `Get` returns the empty value for that type.

`ResourceData` also has a `GetOk` method that functions identically to its `Get` method, but with an extra return parameter. The new return parameter returns `true` if the property is set to a non-zero value, but with caveats. This is where it’s important to remember that `ResourceData` is an amalgamation of input sources; providers cannot determine whether the property is set in the config, in the state, in the diff, or by using `ResourceData.Set`. The only information providers have available to them is that the property has been set at some point, and what its value is right now.

#### Detecting Changes
When there’s a difference between what our infrastructure is and what we want it to be, we want to be able to see what changed and what it should be. To aid in this, `ResourceData` provides a `HasChange` method and a `GetChange` method. Each takes a property key, just like `Get`. `HasChange` returns `true` if there’s a change to that property; the change could be from drift (someone or something modifying the infrastructure outside of Terraform) or from config changes. The `GetChange` method returns two values; a basic understanding is that the first value is what the property _used to be_, and the second value is what the property _should be changed to_.  A slightly more nuanced understanding is that the first value is what is in the state (representing the state of the infrastructure as it exists) and the second value is what is in the config (representing what the user wants the infrastructure to be).

You’ll notice there’s no way to tell whether the state changed or the config changed; this is a common misconception. Terraform never diffs states or configs, it only ever diffs what is and what should be.

Be sure to call `HasChange` before calling `GetChange`; there are some cases where `GetChange` would return both values as equal, but reflect a change. This can happen, for example, when a boolean property has a default value of `false` and was not specified, then gets set explicitly to `false`.

`GetChange` is only really necessary when you need to know what the previous value of the field was; if you only need the new value, `Get` is sufficient. `GetChange` is commonly used in cases where the API requires you to explicitly remove and add items—for example, when there’s a change in tags and the API only offers `AddTag` & `RemoveTag` methods without a way to just change all tags at once. In that case, you need to know what the previous tag value was, so it can be removed.

#### Setting State
At the end of your `Create`, `Get`, `Update`, and `Delete` functions, whatever is in your `ResourceData` object is set as state. To manipulate this, the `Set` method is provided. It takes the key of a property, just like `Get`, as an argument, and the value to set it to.

It’s important that all your properties get set using the `Set` method, as otherwise Terraform will be unable to perform some of its important functions, but will give the appearance of operating normally. For example, changes to the config file will be detected, but changes made outside Terraform will be silently ignored. This breaks Terraform’s promise of reflecting your infrastructure as code, so it’s important that the current state of the infrastructure get persisted using the `Set` method.

#### Partial Updates
This section of the guide is under development and will be coming soon.

#### Working With IDs
Terraform uses IDs to reference resources. It’s part of the key used when accessing the resource, either in the provider or in interpolation. It’s also what the user supplies when running [`terraform import`](TODO) to identify the resource they wish to import. A good ID is immutable and easy for a user to locate.

Terraform special-cases the `id` property on resources. Do not use `id` as a key for a property. Terraform uses that internally, and trying to access an `id` property with the `Get` method will always return a zero value.

To retrieve the ID, use the `GetId` method on `ResourceData`. To set it, use the `SetId` method, passing the string to use as an ID. Only strings may be used as IDs.

You should always set the ID as soon as you possibly can. Even with asynchronous APIs, you shouldn’t wait for the API to finish creating the resource, just set the ID as soon as it’s provided by the API. This is important because Terraform needs to record the fact that the resource exists, even if other Terraform calls fail and cause Terraform to exit. Otherwise, the ID could remain empty and leave the user in a situation where the resource was created, but Terraform doesn’t know about it.

### Working With Either/Or Properties
This section of the guide is under development and will be coming soon.

### Customizing State
This section of the guide is under development and will be coming soon.

### Customizing Validation
This section of the guide is under development and will be coming soon.

### Customizing Diffs
This section of the guide is under development and will be coming soon.

### Storing Sensitive Information
This section of the guide is under development and will be coming soon.

### Versioning Your Schema & Helping Users Upgrade
This section of the guide is under development and will be coming soon.

## Making Resources Importable
This section of the guide is under development and will be coming soon.

## Testing Resources
This section of the guide is under development and will be coming soon.

### Testing Resource Behaviour
This section of the guide is under development and will be coming soon.

### Testing Resource Imports
This section of the guide is under development and will be coming soon.
