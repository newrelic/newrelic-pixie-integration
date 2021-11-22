package main

import (
	"fmt"
	"net"

	"github.com/Shopify/toxiproxy"
	toxiclient "github.com/Shopify/toxiproxy/client"
)

func main() {
	toxiServer := toxiproxy.NewServer()
	ch := make(chan bool, 1)
	go func() {
		toxiServer.Listen("localhost", "8474")
		ch <- true
	}()

	fmt.Println(<-ch)
}

func setUpProxies() error {
	proxyClient := toxiclient.NewClient(net.JoinHostPort("localhost", "8474"))

	_, err := proxyClient.CreateProxy(
		"pixie",
		"0.0.0.0:6000",
		"work.withpixie.ai:443",
	)
	if err != nil {
		return err
	}
	_, err = proxyClient.CreateProxy(
		"nr",
		"0.0.0.0:6001",
		"staging.otlp.nr-data.net:443",
	)

	return err
}
