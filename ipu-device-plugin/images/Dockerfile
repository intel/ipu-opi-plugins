FROM golang:1.22-alpine3.19@sha256:cdc86d9f363e8786845bea2040312b4efa321b828acdeb26f393faa864d887b0 as builder
COPY . /usr/src/ipu-device-plugin

ENV HTTP_PROXY $http_proxy
ENV HTTPS_PROXY $https_proxy
RUN apk add --no-cache --virtual build-dependencies build-base

WORKDIR /usr/src/ipu-device-plugin

RUN make build

FROM alpine:3@sha256:c5b1261d6d3e43071626931fc004f70149baeba2c8ec672bd4f27761f8e1ad6b

COPY --from=builder /usr/src/ipu-device-plugin/build/ipu-dp /usr/bin/
WORKDIR /
LABEL io.k8s.display-name="APF Device Plugin"

COPY ./images/entrypoint.sh /

RUN rm -rf /var/cache/apk/*

ENTRYPOINT ["/entrypoint.sh"]