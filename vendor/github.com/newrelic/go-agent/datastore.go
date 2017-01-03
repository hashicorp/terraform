package newrelic

// DatastoreProduct encourages consistent metrics across New Relic agents.  You
// may create your own if your datastore is not listed below.
type DatastoreProduct string

// Datastore names used across New Relic agents:
const (
	DatastoreCassandra     DatastoreProduct = "Cassandra"
	DatastoreDerby                          = "Derby"
	DatastoreElasticsearch                  = "Elasticsearch"
	DatastoreFirebird                       = "Firebird"
	DatastoreIBMDB2                         = "IBMDB2"
	DatastoreInformix                       = "Informix"
	DatastoreMemcached                      = "Memcached"
	DatastoreMongoDB                        = "MongoDB"
	DatastoreMySQL                          = "MySQL"
	DatastoreMSSQL                          = "MSSQL"
	DatastoreOracle                         = "Oracle"
	DatastorePostgres                       = "Postgres"
	DatastoreRedis                          = "Redis"
	DatastoreSolr                           = "Solr"
	DatastoreSQLite                         = "SQLite"
	DatastoreCouchDB                        = "CouchDB"
	DatastoreRiak                           = "Riak"
	DatastoreVoltDB                         = "VoltDB"
)
