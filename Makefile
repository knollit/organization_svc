all: build

build: flatbuffers
	CGO_ENABLED=0 GOOS=linux go build -a --installsuffix cgo --ldflags="-s" -o dest/organization_svc .
	docker build -t knollit/organization_svc:latest .

flatbuffers:
	flatc -g -o $${GOPATH##*:}/src/github.com/knollit/organization_svc *.fbs

clean:
	rm -rf dest

publish: build
	docker tag knollit/organization_svc:latest knollit/organization_svc:$$CIRCLE_SHA1
	docker push knollit/organization_svc:$$CIRCLE_SHA1
	docker push knollit/organization_svc:latest
