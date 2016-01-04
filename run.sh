#!/bin/sh

./build.sh
docker run --link organizations-postgres:postgres -e POSTGRES_PASSWORD=mysecretpassword -e TLS_CA_PATH=/test-ca.crt -e TLS_CERT_PATH=/test-server.crt -e TLS_KEY_PATH=/test-server.key --name organization_svc --rm organization_svc:latest
