ARG GOLANG_VERSION=1.16
ARG base_image=alpine:3.13


FROM golang:${GOLANG_VERSION} as builder

RUN mkdir newrelic-pixie-integration
WORKDIR newrelic-pixie-integration
COPY go.mod .
RUN go mod download

COPY . ./
RUN go build cmd/main.go ; mv main /usr/bin/newrelic-pixie-integration

CMD ./newrelic-pixie-integration



FROM $base_image AS core

ARG image_version=0.0
ARG agent_version=0.0
ARG version_file=VERSION
ARG agent_bin=newrelic-pixie-integration

# Add the agent binary
COPY --from=builder /usr/bin/newrelic-pixie-integration /usr/bin/newrelic-pixie-integration


LABEL com.newrelic.image.version=$image_version \
      com.newrelic.infra-agent.version=$agent_version \
      com.newrelic.maintainer="infrastructure-eng@newrelic.com" \
      com.newrelic.description="New Relic Infrastructure Pixie integration."

RUN apk --no-cache upgrade

RUN apk add --no-cache --upgrade \
    ca-certificates=20211220-r0 \
    && mkdir /lib64 \
    && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2 \
    && apk add --no-cache tini=0.19.0-r0

# Tini is now available at /sbin/tini
ENTRYPOINT ["/sbin/tini", "--"]
CMD ["/usr/bin/newrelic-pixie-integration"]
