package main

import (
	"crypto/tls"
	"flag"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	log "github.com/Sirupsen/logrus"
	"github.com/jtblin/go-acme"
	pb "github.com/jtblin/go-acme/examples/grpc/helloworld"
	"github.com/jtblin/go-acme/types"
	"golang.org/x/net/context"
)

var address, email, domain string
var verbose bool

type server struct{}

func (s *server) SayHello(cxt context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	return &pb.HelloReply{Message: "Hello " + in.Name}, nil
}

func main() {
	flag.Parse()
	if verbose {
		log.SetLevel(log.DebugLevel)
	}

	ACME := &acme.ACME{
		Email:       email,
		DNSProvider: "route53",
		Domain:      &types.Domain{Main: domain},
		Logger:      log.New(),
	}
	tlsConfig := &tls.Config{}
	if err := ACME.CreateConfig(tlsConfig); err != nil {
		panic(err)
	}
	ta := credentials.NewTLS(tlsConfig)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		panic("failed to listen: " + err.Error())
	}
	grpcServer := grpc.NewServer(grpc.Creds(ta))
	pb.RegisterGreeterServer(grpcServer, &server{})
	if err = grpcServer.Serve(listener); err != nil {
		panic(err)
	}
}

func init() {
	flag.StringVar(&address, "address", ":8443", "Listener address e.g. :443")
	flag.StringVar(&email, "email", "", "Email address to register account")
	flag.StringVar(&domain, "domain", "", "Domain for which to generate certificates")
	flag.BoolVar(&verbose, "verbose", false, "Verbose logging")
}
