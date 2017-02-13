package main

import (
	"flag"
	"log"
	"time"

	"net/http"
	_ "net/http/pprof"

	"github.com/cloudfoundry-incubator/scalable-syslog/api"
	"github.com/cloudfoundry-incubator/scalable-syslog/scheduler/app"
)

func main() {
	healthHostport := flag.String("health", ":8080", "The hostport to listen for health requests")
	pprofHostport := flag.String("pprof", ":6060", "The hostport to listen for pprof")

	cupsProvider := flag.String("cups-url", "", "The URL of the CUPS provider")
	cupsCAFile := flag.String("cups-ca", "", "The file path for the CA cert")
	cupsCertFile := flag.String("cups-cert", "", "The file path for the client cert")
	cupsKeyFile := flag.String("cups-key", "", "The file path for the client key")
	cupsCommonName := flag.String("cups-cn", "", "The common name used for the TLS config")
	skipCertVerify := flag.Bool("cups-skip-cert-verify", false, "The option to allow insecure SSL connections")

	caFile := flag.String("ca", "", "The file path for the CA cert")
	certFile := flag.String("cert", "", "The file path for the adapter server cert")
	keyFile := flag.String("key", "", "The file path for the adapter server key")
	commonName := flag.String("cn", "", "The common name used for the TLS config")

	adapterIPs := flag.String("adapter-ips", "", "Comma separated list of adapter IP addresses")
	adapterPort := flag.String("adapter-port", "", "The port of the adapter API")

	flag.Parse()

	adapterAddrs, err := app.ParseAddrs(*adapterIPs, *adapterPort)
	if err != nil {
		log.Fatalf("No adapter addresses: %s", err)
	}

	cupsTLSConfig, err := api.NewMutualTLSConfig(*cupsCertFile, *cupsKeyFile, *cupsCAFile, *cupsCommonName)
	if err != nil {
		log.Fatalf("Invalid TLS config: %s", err)
	}
	cupsTLSConfig.InsecureSkipVerify = *skipCertVerify

	tlsConfig, err := api.NewMutualTLSConfig(*certFile, *keyFile, *caFile, *commonName)
	if err != nil {
		log.Fatalf("Invalid TLS config: %s", err)
	}

	app.Start(
		app.WithHealthAddr(*healthHostport),
		app.WithCUPSUrl(*cupsProvider),
		app.WithHTTPClient(api.NewHTTPSClient(cupsTLSConfig, 5*time.Second)),
		app.WithAdapterAddrs(adapterAddrs),
		app.WithTLSConfig(tlsConfig),
	)

	log.Println(http.ListenAndServe(*pprofHostport, nil))
}
