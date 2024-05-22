FROM golang:alpine@sha256:cdc86d9f363e8786845bea2040312b4efa321b828acdeb26f393faa864d887b0 as builder

COPY . /usr/src/sriov-cni

ENV HTTP_PROXY $http_proxy
ENV HTTPS_PROXY $https_proxy

WORKDIR /usr/src/sriov-cni
RUN apk add --no-cache --virtual build-dependencies build-base=~0.5 && \
    make clean && \
    make build

FROM alpine:3@sha256:77726ef6b57ddf65bb551896826ec38bc3e53f75cdde31354fbffb4f25238ebd
COPY --from=builder /usr/src/sriov-cni/build/sriov /usr/bin/
WORKDIR /

LABEL io.k8s.display-name="SR-IOV CNI"

COPY ./images/entrypoint.sh /

ENTRYPOINT ["/entrypoint.sh"]
