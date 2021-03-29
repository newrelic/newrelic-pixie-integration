FROM golang:1.15

RUN mkdir -p app

COPY go.mod app/
RUN cd app && go mod download

COPY . app/
RUN cd app && go build ./cmd/pixie-integration

CMD ./app/pixie-integration
