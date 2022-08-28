FROM alpine:3.15

COPY bin/agent /

RUN mkdir /log/ && chmod -R 777 /log/

ENTRYPOINT ["/agent"]
