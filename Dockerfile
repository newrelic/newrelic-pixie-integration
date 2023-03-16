ARG base_image=alpine:3.17.2

FROM golang:1.20-alpine as builder

RUN mkdir newrelic-pixie-integration
WORKDIR newrelic-pixie-integration
COPY . ./
RUN go mod download
RUN go build -o /usr/bin/newrelic-pixie-integration cmd/main.go


FROM $base_image AS core

ARG image_version=0.0
ARG agent_version=0.0
ARG version_file=VERSION
ARG agent_bin=newrelic-pixie-integration

WORKDIR /app

LABEL com.newrelic.image.version=$image_version \
      com.newrelic.infra-agent.version=$agent_version \
      com.newrelic.maintainer="infrastructure-eng@newrelic.com" \
      com.newrelic.description="New Relic Infrastructure Pixie integration."

RUN apk add --no-cache --upgrade \
    tini ca-certificates \
    && addgroup -g 2000 newrelic-pixie-integration \
    && adduser -D -H -u 1000 -G newrelic-pixie-integration newrelic-pixie-integration

USER newrelic-pixie-integration

# Add the agent binary
COPY --from=builder /usr/bin/newrelic-pixie-integration ./newrelic-pixie-integration

ENTRYPOINT ["/sbin/tini", "--"]
CMD ["./newrelic-pixie-integration"]
