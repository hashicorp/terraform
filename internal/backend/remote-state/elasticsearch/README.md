# How to test the `elasticsearch` backend

## Create an Elasticsearch resource in a Docker container

1. Generate PKI for `elasticsearch` container

```bash
./testdata/gencerts.sh
```

2. Get the latest elastic version

```bash
  es_versions="$(curl -s "https://artifacts.elastic.co/releases/stack.json")"
  es_latest_version=$(echo "$es_versions" | awk -F'"' '/"version": *"/ {print $4}' | grep -E '^[0-9]+\.[0-9]+\.[0-9]+( GA)?$' | awk -F'.' '{ printf("%d %d %d %s\n", $1, $2, $3, $0) }' | sort -n -k1,1 -k2,2 -k3,3 | awk '{print $4}' | tail -n 1)
  # Remove the GA prefix from the version, if present
  es_latest_version=$(echo "$es_latest_version" | awk '{ gsub(/ GA$/, "", $0); print }')
```

3. Run this command to launch a Docker container using the `elasticsearch` version from step 2 and certificates/keys from step 1:

```bash
docker run \
  --name es_backend_testing \
  --rm \
  -p 9200:9200 \
  -m 1GB \
  -e ELASTIC_PASSWORD=changeme \
  -e discovery.type=single-node \
  -e xpack.security.enabled=true \
  -e xpack.security.http.ssl.enabled=true \
  -e xpack.security.http.ssl.key=/usr/share/elasticsearch/config/certs/server.key \
  -e xpack.security.http.ssl.certificate_authorities=/usr/share/elasticsearch/config/certs/ca.cert.pem \
  -e xpack.security.http.ssl.certificate=/usr/share/elasticsearch/config/certs/server.crt \
  -e xpack.security.transport.ssl.enabled=true \
  -e xpack.security.transport.ssl.certificate_authorities=/usr/share/elasticsearch/config/certs/ca.cert.pem \
  -e xpack.security.transport.ssl.certificate=/usr/share/elasticsearch/config/certs/server.crt \
  -e xpack.security.transport.ssl.key=/usr/share/elasticsearch/config/certs/server.key \
  -v ./testdata/certs:/usr/share/elasticsearch/config/certs:ro \
  docker.elastic.co/elasticsearch/elasticsearch:$es_latest_version
```

4. Wait for cluster to be green

```bash
while ! curl -sf https://localhost:9200 -u elastic:changeme -k; do
  echo "Waiting for service..."
  sleep 10
done
```

## Set up environment variables needed for tests

Set the following environment variables:

```
ELASTICSEARCH_URL=https://localhost:9200
```

## Run the tests!

The setup above should be sufficient for running the tests.
