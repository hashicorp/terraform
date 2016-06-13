package cassandra

import (
	"fmt"
)

func validateReplicationClass(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if value != ReplicationStrategySimple && value != ReplicationStrategyNetworkTopology {
		errors = append(errors, fmt.Errorf("replication_class must be one of [%s, %s]",
			ReplicationStrategySimple, ReplicationStrategyNetworkTopology))
	}
	return
}
