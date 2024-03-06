FROM golang:1.22-alpine@sha256:fc5e5848529786cf1136563452b33d713d5c60b2c787f6b2a077fa6eeefd9114 as builder
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