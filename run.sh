#!/bin/sh

./build.sh
docker run --link organizations-postgres:postgres -e POSTGRES_PASSWORD=mysecretpassword -e TLS_CA_PATH=/ca.crt -e TLS_CERT_PATH=/organization7.api-proj.com.crt -e TLS_KEY_PATH=/organization7.api-proj.com.key --name organizations --rm organizations:latest
