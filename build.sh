#!/bin/sh

go get
CGO_ENABLED=0 GOOS=linux go build -a --installsuffix cgo --ldflags="-s" -o organizations .
docker build -t organizations:latest .
rm organizations
