FROM golang:1.15

RUN mkdir newrelic-pixie-integration
WORKDIR newrelic-pixie-integration
COPY go.mod .
RUN go mod download

COPY . ./
RUN go build cmd/main.go ; mv main newrelic-pixie-integration

CMD ./newrelic-pixie-integration


