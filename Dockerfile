FROM alpine:3.15

COPY bin/agent /

RUN chmod -R 777 /var/log/

ENTRYPOINT ["/agent"]
