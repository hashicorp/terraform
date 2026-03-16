# How to test the `pg` backend

## Create a Postgres resources in a Docker container

1.Run this command to launch a Docker container using the `postgres` image that can use SSL:

```bash
docker run \
  --name pg_backend_testing \
  --rm \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=password \
  -p 5432:5432 \
  postgres:latest \
  -c ssl=on \
  -c ssl_cert_file=/etc/ssl/certs/ssl-cert-snakeoil.pem \
  -c ssl_key_file=/etc/ssl/private/ssl-cert-snakeoil.key
```

Note that for testing we use the user `"postgres"`, and this value is reused in command below.

2. Use `exec` to access a shell inside the Docker container:

```bash
% docker exec -it $(docker ps -aqf "name=^pg_backend_testing$") bash
```

3. Run a command to create a Postgres database called `terraform_backend_pg_test`

```bash
root@<container-id>:/# createdb -U postgres terraform_backend_pg_test
root@<container-id>:/# exit
```

## Set up environment variables needed for tests

Set the following environment variables:

```
DATABASE_URL=postgresql://localhost:5432/terraform_backend_pg_test?sslmode=require
PGUSER=postgres
PGPASSWORD=password
```

The `DATABASE_URL` value is a connection string and should not include the username and password. Instead,
the username and password must be supplied by separate environment variables to let some tests override those
values.

## Run the tests!

The setup above should be sufficient for running the tests. Each time you want to run the tests you will need to re-launch the container and
create the `terraform_backend_pg_test` database that's expected by the tests.