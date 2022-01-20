FROM alpine:3.6

COPY agent /

ENTRYPOINT ["/agent"]
