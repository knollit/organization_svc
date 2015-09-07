#!/bin/sh

docker run --name organizations-postgres -e POSTGRES_PASSWORD=mysecretpassword -d org-db
