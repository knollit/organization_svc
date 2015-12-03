#!/bin/sh

flatc -g *.fbs
go get
CGO_ENABLED=0 GOOS=linux go build -a --installsuffix cgo --ldflags="-s" -o organization_svc .
docker build -t organizations:latest .
rm organization_svc
