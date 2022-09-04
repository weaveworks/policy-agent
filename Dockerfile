FROM alpine:3.15

COPY bin/agent /

RUN mkdir /logs && chmod -R 777 /logs

ENTRYPOINT ["/agent"]
