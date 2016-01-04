FROM centurylink/ca-certs

COPY organization_svc /
COPY certs /

EXPOSE 13800

ENTRYPOINT ["/organization_svc"]
