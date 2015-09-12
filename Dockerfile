FROM centurylink/ca-certs

COPY organizations /

EXPOSE 13800

ENTRYPOINT ["/organizations"]
