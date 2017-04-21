package internal

var (
	datastoreProductMetricsCache = map[string]datastoreProductMetrics{
		"Cassandra": {
			All:   "Datastore/Cassandra/all",
			Web:   "Datastore/Cassandra/allWeb",
			Other: "Datastore/Cassandra/allOther",
		},
		"Derby": {
			All:   "Datastore/Derby/all",
			Web:   "Datastore/Derby/allWeb",
			Other: "Datastore/Derby/allOther",
		},
		"Elasticsearch": {
			All:   "Datastore/Elasticsearch/all",
			Web:   "Datastore/Elasticsearch/allWeb",
			Other: "Datastore/Elasticsearch/allOther",
		},
		"Firebird": {
			All:   "Datastore/Firebird/all",
			Web:   "Datastore/Firebird/allWeb",
			Other: "Datastore/Firebird/allOther",
		},
		"IBMDB2": {
			All:   "Datastore/IBMDB2/all",
			Web:   "Datastore/IBMDB2/allWeb",
			Other: "Datastore/IBMDB2/allOther",
		},
		"Informix": {
			All:   "Datastore/Informix/all",
			Web:   "Datastore/Informix/allWeb",
			Other: "Datastore/Informix/allOther",
		},
		"Memcached": {
			All:   "Datastore/Memcached/all",
			Web:   "Datastore/Memcached/allWeb",
			Other: "Datastore/Memcached/allOther",
		},
		"MongoDB": {
			All:   "Datastore/MongoDB/all",
			Web:   "Datastore/MongoDB/allWeb",
			Other: "Datastore/MongoDB/allOther",
		},
		"MySQL": {
			All:   "Datastore/MySQL/all",
			Web:   "Datastore/MySQL/allWeb",
			Other: "Datastore/MySQL/allOther",
		},
		"MSSQL": {
			All:   "Datastore/MSSQL/all",
			Web:   "Datastore/MSSQL/allWeb",
			Other: "Datastore/MSSQL/allOther",
		},
		"Oracle": {
			All:   "Datastore/Oracle/all",
			Web:   "Datastore/Oracle/allWeb",
			Other: "Datastore/Oracle/allOther",
		},
		"Postgres": {
			All:   "Datastore/Postgres/all",
			Web:   "Datastore/Postgres/allWeb",
			Other: "Datastore/Postgres/allOther",
		},
		"Redis": {
			All:   "Datastore/Redis/all",
			Web:   "Datastore/Redis/allWeb",
			Other: "Datastore/Redis/allOther",
		},
		"Solr": {
			All:   "Datastore/Solr/all",
			Web:   "Datastore/Solr/allWeb",
			Other: "Datastore/Solr/allOther",
		},
		"SQLite": {
			All:   "Datastore/SQLite/all",
			Web:   "Datastore/SQLite/allWeb",
			Other: "Datastore/SQLite/allOther",
		},
		"CouchDB": {
			All:   "Datastore/CouchDB/all",
			Web:   "Datastore/CouchDB/allWeb",
			Other: "Datastore/CouchDB/allOther",
		},
		"Riak": {
			All:   "Datastore/Riak/all",
			Web:   "Datastore/Riak/allWeb",
			Other: "Datastore/Riak/allOther",
		},
		"VoltDB": {
			All:   "Datastore/VoltDB/all",
			Web:   "Datastore/VoltDB/allWeb",
			Other: "Datastore/VoltDB/allOther",
		},
	}
)
