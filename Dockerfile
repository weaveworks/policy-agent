FROM alpine:3.6

COPY tls.crt /

COPY tls.key /

COPY agent /

ENTRYPOINT ["/agent", "--write-compliance"]