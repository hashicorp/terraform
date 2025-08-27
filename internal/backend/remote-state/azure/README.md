# How to test the `azure` backend

For HashiCorp engineers:

* Create a [temporary Azure subscription](https://docs.prod.secops.hashicorp.services/doormat/azure/create_temp_subscription/) via Doormat.
* You can only access the Azure console through Doormat.
    * View your accounts in Doormat, [here](https://doormat.hashicorp.services/).
    * Navigate to the Azure Subscriptions tab
    * Click the icon in the Console Access Link column that corresponds to the subscription you wish to use.
* Follow the guidance in the [Azure + TFC page](https://docs.prod.secops.hashicorp.services/doormat/azure/azure_tfc/) to set up the necessary resources:
    * **Create an Azure application** in the temporary subscription
    * **Set up a client secret**: Via the page for the Azure application you've just created, create a client secret following [these instructions](https://techcommunity.microsoft.com/discussions/azureadvancedthreatprotection/app-secret-application-secret-azure-ad---azure-ad-app-secrets/3775325/replies/3776572#M3429).
        * Make sure to make a copy the client secret's value (not id) before closing the page.
    * **Give the application permissions**: From [this page listing your Subscriptions](https://portal.azure.com/#view/Microsoft_Azure_Billing/SubscriptionsBladeV2), click the temporary subscription you created. On the homepage for that subscription, click the `Access Control (IAM)` tab in the left menu.
        * Click `Add > Add role assignment`
        * Under `Role`, click `Privileged administrator roles` and then find and click on the `Contributor` role.
        * Navigate to `Members`. Make sure `Assign access to: User, group, or service principal` is selected.
            * Click `Select members` and then use the search window that opens to search for the **display name of your Application**. It should be displayed indicating that you are selecting an application, i.e. a service principal.
        * Click `Review + assign` to update the role assignments associated with your subscription.
        * The new assignment should be visible under the `Role assignments` tab when viewing the `Access Control (IAM)` tab for your subscription.

## Setting environment variables

You need to set the environment variables below.
Navigate to the `Overview` page for your Application to get the necessary values.

* `TF_ACC=1`
* `TF_AZURE_TEST=1`
* `ARM_SUBSCRIPTION_ID` - the id of the temporary subscription in use. This can be obtained from [the Subscriptions page](https://portal.azure.com/#view/Microsoft_Azure_Billing/SubscriptionsBladeV2).
* `ARM_TENANT_ID` - the Directory (tenant) ID visible in the Overview page.
* `ARM_LOCATION` - a region within Azure. See [this list](https://gist.github.com/ausfestivus/04e55c7d80229069bf3bc75870630ec8), or pick "eastus" if you aren't concerned about the choice.
* `ARM_CLIENT_SECRET` - the client secret value (not id) created in the steps above.
* `ARM_CLIENT_ID` - the Application (client) ID visible in the Overview page.

There are other environment variables that can be used in tests, but the list above is the minimal number of environment variables to be able to start running tests.

## Run the tests!

Run tests in the `internal/backend/remote-state/azure` package with the above environment variables set.
Look out for errors; if any errors complain about an environment variable being unset make sure it is set and/or
run that test individually instead of in parallel with other tests. Look out for tests that are skipped due to the tests not being run in a given environment (e.g. GitHub Actions).