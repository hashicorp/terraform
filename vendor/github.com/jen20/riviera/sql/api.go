package sql

import "fmt"

const apiVersion = "2014-04-01-preview"
const apiProvider = "Microsoft.Sql"

func sqlServerDefaultURLPath(resourceGroupName, serverName string) func() string {
	return func() string {
		return fmt.Sprintf("resourceGroups/%s/providers/%s/servers/%s", resourceGroupName, apiProvider, serverName)
	}
}

func sqlElasticPoolDefaultURLPath(resourceGroupName, serverName, elasticPoolName string) func() string {
	return func() string {
		return fmt.Sprintf("resourceGroups/%s/providers/%s/servers/%s/elasticPools/%s", resourceGroupName, apiProvider, serverName, elasticPoolName)
	}
}

func sqlDatabaseDefaultURLPath(resourceGroupName, serverName, databaseName string) func() string {
	return func() string {
		return fmt.Sprintf("resourceGroups/%s/providers/%s/servers/%s/databases/%s", resourceGroupName, apiProvider, serverName, databaseName)
	}
}

func sqlDatabaseFailoverUnplanned(resourceGroupName, serverName, databaseName, linkID string) func() string {
	return func() string {
		return fmt.Sprintf("resourcegroups/%s/providers/%s/servers/%s/databases/%s/replicationLinks/%s/forceFailoverAllowDataLoss", resourceGroupName, apiProvider, serverName, databaseName, linkID)
	}
}

func sqlServerFirewallDefaultURLPath(resourceGroupName, serverName, firewallRuleName string) func() string {
	return func() string {
		return fmt.Sprintf("resourceGroups/%s/providers/%s/servers/%s/firewallRules/%s", resourceGroupName, apiProvider, serverName, firewallRuleName)
	}
}
