# How to test the `kubernetes` backend

## Create a Kubernetes cluster

1. Install `kind`, e.g [install kind via Homebrew](https://formulae.brew.sh/formula/kind)
1. Provision a new cluster with the command `kind create cluster --name=terraform`
    * You can check for the cluster using `kind get clusters`

## Set up environment variables for testing

Creating the cluster in the steps above should have created and/or added an entry into the `~/.kube/config` configuration file.

Create the `KUBE_CONFIG_PATH` environment variable to help the backend locate that file:

```bash
export KUBE_CONFIG_PATH=~/.kube/config
```

## Run the tests!

The setup above should be sufficient for running the tests. Make sure your kind cluster exists and is running whenever you run the tests.
