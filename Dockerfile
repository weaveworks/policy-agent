FROM alpine:3.15

COPY bin/agent /

RUN mkdir /log/ && chown -R 1000:1000 /log

ENTRYPOINT ["/agent"]
