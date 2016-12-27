# env TESTARGS='-run TestAccPostgresqlSchema_AddPolicy' TF_LOG=warn make test
#
# NOTE: As of PostgreSQL 9.6.1 the -test.parallel=1 is required when
# performing `DROP ROLE`-related actions.  This behavior and requirement
# may change in the future and is likely not required when doing
# non-delete related operations. But for now it is.

POSTGRES?=$(wildcard /usr/local/bin/postgres /opt/local/lib/postgresql96/bin/postgres)
PSQL?=$(wildcard /usr/local/bin/psql /opt/local/lib/postgresql96/bin/psql)
INITDB?=$(wildcard /usr/local/bin/initdb /opt/local/lib/postgresql96/bin/initdb)

PGDATA?=$(GOPATH)/src/github.com/hashicorp/terraform/builtin/providers/postgresql/data

initdb::
	echo "" > pwfile
	$(INITDB) --no-locale -U postgres -A md5 --pwfile=pwfile -D $(PGDATA)

startdb::
	2>&1 \
	$(POSTGRES) \
		-D $(PGDATA) \
		-c log_connections=on \
		-c log_disconnections=on \
		-c log_duration=on \
		-c log_statement=all \
	| tee postgresql.log

cleandb::
	rm -rf $(PGDATA)
	rm -f pwfile

freshdb:: cleandb initdb startdb

test::
	2>&1 PGSSLMODE=disable PGHOST=/tmp PGUSER=postgres make -C ../../.. testacc TEST=./builtin/providers/postgresql | tee test.log

psql::
	$(PSQL) -E postgres postgres
