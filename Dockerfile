FROM centurylink/ca-certs

COPY organizations /

EXPOSE 80

ENTRYPOINT ["/organizations"]
