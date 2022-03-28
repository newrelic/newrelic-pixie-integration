module github.com/newrelic/newrelic-pixie-integration

require (
	github.com/Shopify/toxiproxy v2.1.4+incompatible
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/newrelic/infrastructure-agent v0.0.0-20210422145429-ffccdd3cde2b
	github.com/sirupsen/logrus v1.8.1
	go.opentelemetry.io/proto/otlp v0.12.1
	golang.org/x/text v0.3.6 // indirect
	google.golang.org/grpc v1.44.0
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	px.dev/pxapi v0.2.1
)

go 1.16
