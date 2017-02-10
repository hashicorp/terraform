// Copyright 2016 Circonus, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package api provides methods for interacting with the Circonus API. See the full Circonus API
Documentation at https://login.circonus.com/resources/api for more information.

Raw REST methods

    Get     - retrieve existing item(s)
    Put 	- update an existing item
    Post    - create a new item
    Delete	- remove an existing item

Endpoints (supported)

    Account                 https://login.circonus.com/resources/api/calls/account
    Acknowledgement         https://login.circonus.com/resources/api/calls/acknowledgement
    Alert                   https://login.circonus.com/resources/api/calls/alert
    Annotation              https://login.circonus.com/resources/api/calls/annotation
    Broker                  https://login.circonus.com/resources/api/calls/broker
    Check                   https://login.circonus.com/resources/api/calls/check
    Check Bundle            https://login.circonus.com/resources/api/calls/check_bundle
    Check Bundle Metrics    https://login.circonus.com/resources/api/calls/check_bundle_metrics
    Contact Group           https://login.circonus.com/resources/api/calls/contact_group
    Dashboard               https://login.circonus.com/resources/api/calls/dashboard
    Graph                   https://login.circonus.com/resources/api/calls/graph
    Maintenance [window]    https://login.circonus.com/resources/api/calls/maintenance
    Metric                  https://login.circonus.com/resources/api/calls/metric
    Metric Cluster          https://login.circonus.com/resources/api/calls/metric_cluster
    Outlier Report          https://login.circonus.com/resources/api/calls/outlier_report
    Provision Broker        https://login.circonus.com/resources/api/calls/provision_broker
    Rule Set                https://login.circonus.com/resources/api/calls/rule_set
    Rule Set Group          https://login.circonus.com/resources/api/calls/rule_set_group
    User                    https://login.circonus.com/resources/api/calls/user
    Worksheet               https://login.circonus.com/resources/api/calls/worksheet

Endpoints (not supported)

    Support may be added for these endpoints in the future. These endpoints may currently be used
    directly with the Raw REST methods above.

    CAQL                    https://login.circonus.com/resources/api/calls/caql
    Check Move              https://login.circonus.com/resources/api/calls/check_move
    Data                    https://login.circonus.com/resources/api/calls/data
    Snapshot                https://login.circonus.com/resources/api/calls/snapshot
    Tag                     https://login.circonus.com/resources/api/calls/tag
    Template                https://login.circonus.com/resources/api/calls/template

Verbs

    Fetch   singular/plural item(s) - e.g. FetchAnnotation, FetchAnnotations
    Create  create new item         - e.g. CreateAnnotation
    Update  update an item          - e.g. UpdateAnnotation
    Delete  remove an item          - e.g. DeleteAnnotation, DeleteAnnotationByCID
    Search  search for item(s)      - e.g. SearchAnnotations
    New     new item config         - e.g. NewAnnotation (returns an empty item,
                                           any applicable defautls defined)

    Not all endpoints support all verbs.
*/
package api
