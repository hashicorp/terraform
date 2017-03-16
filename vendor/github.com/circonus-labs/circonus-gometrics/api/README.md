## Circonus API package

Full api documentation (for using *this* package) is available at [godoc.org](https://godoc.org/github.com/circonus-labs/circonus-gometrics/api). Links in the lists below go directly to the generic Circonus API documentation for the endpoint.

### Straight [raw] API access

* Get
* Post (for creates)
* Put (for updates)
* Delete

### Helpers for currently supported API endpoints

> Note, these interfaces are still being actively developed. For example, many of the `New*` methods only return an empty struct; sensible defaults will be added going forward. Other, common helper methods for the various endpoints may be added as use cases emerge. The organization
of the API may change if common use contexts would benefit significantly.

* [Account](https://login.circonus.com/resources/api/calls/account)
    * FetchAccount
    * FetchAccounts
    * UpdateAccount
    * SearchAccounts
* [Acknowledgement](https://login.circonus.com/resources/api/calls/acknowledgement)
    * NewAcknowledgement
    * FetchAcknowledgement
    * FetchAcknowledgements
    * UpdateAcknowledgement
    * CreateAcknowledgement
    * DeleteAcknowledgement
    * DeleteAcknowledgementByCID
    * SearchAcknowledgements
* [Alert](https://login.circonus.com/resources/api/calls/alert)
    * FetchAlert
    * FetchAlerts
    * SearchAlerts
* [Annotation](https://login.circonus.com/resources/api/calls/annotation)
    * NewAnnotation
    * FetchAnnotation
    * FetchAnnotations
    * UpdateAnnotation
    * CreateAnnotation
    * DeleteAnnotation
    * DeleteAnnotationByCID
    * SearchAnnotations
* [Broker](https://login.circonus.com/resources/api/calls/broker)
    * FetchBroker
    * FetchBrokers
    * SearchBrokers
* [Check Bundle](https://login.circonus.com/resources/api/calls/check_bundle)
    * NewCheckBundle
    * FetchCheckBundle
    * FetchCheckBundles
    * UpdateCheckBundle
    * CreateCheckBundle
    * DeleteCheckBundle
    * DeleteCheckBundleByCID
    * SearchCheckBundles
* [Check Bundle Metrics](https://login.circonus.com/resources/api/calls/check_bundle_metrics)
    * FetchCheckBundleMetrics
    * UpdateCheckBundleMetrics
* [Check](https://login.circonus.com/resources/api/calls/check)
    * FetchCheck
    * FetchChecks
    * SearchChecks
* [Contact Group](https://login.circonus.com/resources/api/calls/contact_group)
    * NewContactGroup
    * FetchContactGroup
    * FetchContactGroups
    * UpdateContactGroup
    * CreateContactGroup
    * DeleteContactGroup
    * DeleteContactGroupByCID
    * SearchContactGroups
* [Dashboard](https://login.circonus.com/resources/api/calls/dashboard) -- note, this is a work in progress, the methods/types may still change
    * NewDashboard
    * FetchDashboard
    * FetchDashboards
    * UpdateDashboard
    * CreateDashboard
    * DeleteDashboard
    * DeleteDashboardByCID
    * SearchDashboards
* [Graph](https://login.circonus.com/resources/api/calls/graph)
    * NewGraph
    * FetchGraph
    * FetchGraphs
    * UpdateGraph
    * CreateGraph
    * DeleteGraph
    * DeleteGraphByCID
    * SearchGraphs
* [Metric Cluster](https://login.circonus.com/resources/api/calls/metric_cluster)
    * NewMetricCluster
    * FetchMetricCluster
    * FetchMetricClusters
    * UpdateMetricCluster
    * CreateMetricCluster
    * DeleteMetricCluster
    * DeleteMetricClusterByCID
    * SearchMetricClusters
* [Metric](https://login.circonus.com/resources/api/calls/metric)
    * FetchMetric
    * FetchMetrics
    * UpdateMetric
    * SearchMetrics
* [Maintenance window](https://login.circonus.com/resources/api/calls/maintenance)
    * NewMaintenanceWindow
    * FetchMaintenanceWindow
    * FetchMaintenanceWindows
    * UpdateMaintenanceWindow
    * CreateMaintenanceWindow
    * DeleteMaintenanceWindow
    * DeleteMaintenanceWindowByCID
    * SearchMaintenanceWindows
* [Outlier Report](https://login.circonus.com/resources/api/calls/outlier_report)
    * NewOutlierReport
    * FetchOutlierReport
    * FetchOutlierReports
    * UpdateOutlierReport
    * CreateOutlierReport
    * DeleteOutlierReport
    * DeleteOutlierReportByCID
    * SearchOutlierReports
* [Provision Broker](https://login.circonus.com/resources/api/calls/provision_broker)
    * NewProvisionBroker
    * FetchProvisionBroker
    * UpdateProvisionBroker
    * CreateProvisionBroker
* [Rule Set](https://login.circonus.com/resources/api/calls/rule_set)
    * NewRuleset
    * FetchRuleset
    * FetchRulesets
    * UpdateRuleset
    * CreateRuleset
    * DeleteRuleset
    * DeleteRulesetByCID
    * SearchRulesets
* [Rule Set Group](https://login.circonus.com/resources/api/calls/rule_set_group)
    * NewRulesetGroup
    * FetchRulesetGroup
    * FetchRulesetGroups
    * UpdateRulesetGroup
    * CreateRulesetGroup
    * DeleteRulesetGroup
    * DeleteRulesetGroupByCID
    * SearchRulesetGroups
* [User](https://login.circonus.com/resources/api/calls/user)
    * FetchUser
    * FetchUsers
    * UpdateUser
    * SearchUsers
* [Worksheet](https://login.circonus.com/resources/api/calls/worksheet)
    * NewWorksheet
    * FetchWorksheet
    * FetchWorksheets
    * UpdateWorksheet
    * CreateWorksheet
    * DeleteWorksheet
    * DeleteWorksheetByCID
    * SearchWorksheets

---

Unless otherwise noted, the source files are distributed under the BSD-style license found in the LICENSE file.
