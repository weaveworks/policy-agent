FROM alpine:3.15

COPY build/agent /

ENTRYPOINT ["/agent"]
