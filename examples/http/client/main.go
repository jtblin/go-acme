package main

import (
	"flag"
	"io"
	"net/http"
	"os"

	log "github.com/Sirupsen/logrus"
)

var address string

func main() {
	flag.Parse()
	resp, err := http.Get("https://" + address)
	if err != nil {
		log.Fatalf("Did not connect: %v", err)
	}
	defer resp.Body.Close()
	if _, err = io.Copy(os.Stdout, resp.Body); err != nil {
		log.Fatal(err)
	}
}

func init() {
	flag.StringVar(&address, "address", "", "Address of the server e.g. foo.bar.com:443")
}
