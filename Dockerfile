FROM alpine:3.15

COPY bin/agent /

ENTRYPOINT ["/agent"]
