package main

import (
	"crypto/tls"
	"flag"
	"net/http"

	"github.com/jtblin/go-acme"
	"github.com/jtblin/go-acme/types"
)

var email, domain string

func main() {
	flag.Parse()
	ACME := &acme.ACME{
		BackendName: "s3",
		CAServer:    "https://acme-staging.api.letsencrypt.org/directory",
		DNSProvider: "route53",
		Email:       email,
		Domain:      &types.Domain{Main: domain},
	}
	tlsConfig := &tls.Config{}
	if err := ACME.CreateConfig(tlsConfig); err != nil {
		panic(err)
	}
	listener, err := tls.Listen("tcp", ":8443", tlsConfig)
	if err != nil {
		panic("Listener: " + err.Error())
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })

	// To enable http2, we need http.Server to have reference to tlsConfig
	// https://github.com/golang/go/issues/14374
	server := &http.Server{
		Addr:      ":8443",
		Handler:   mux,
		TLSConfig: tlsConfig,
	}
	server.Serve(listener)
}

func init() {
	flag.StringVar(&email, "email", "", "Email address to register account")
	flag.StringVar(&domain, "domain", "", "Domain for which to generatec ertificates")
}
