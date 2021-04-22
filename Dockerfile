

FROM golang:1.15

RUN mkdir newrelic-pixie-integration
WORKDIR newrelic-pixie-integration
COPY go.mod .
RUN go mod download

COPY *.go ./
RUN go build

CMD ./newrelic-pixie-integration
