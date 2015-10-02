FROM centurylink/ca-certs

COPY organizations /
COPY certs /

EXPOSE 13800

ENTRYPOINT ["/organizations"]
