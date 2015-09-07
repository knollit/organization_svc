#!/bin/sh

./build.sh
docker run --link organizations-postgres:postgres -e POSTGRES_PASSWORD=mysecretpassword --name organizations --rm organizations:latest
