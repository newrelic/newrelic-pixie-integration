FROM golang:1.20-alpine as builder

RUN mkdir newrelic-pixie-integration
WORKDIR newrelic-pixie-integration

# We don't expect the go.mod/go.sum to change frequently.
# So splitting out the mod download helps create another layer
# that should cache well.
COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . ./
RUN go build -o /usr/bin/newrelic-pixie-integration cmd/main.go


FROM alpine:3.18.0

ARG image_version=0.0

WORKDIR /app

LABEL com.newrelic.image.version=$image_version \
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
