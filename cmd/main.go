package main

import (
	"flag"
	"log"

	"github.com/sath-run/engine/cmd/core"
	"github.com/sath-run/engine/pkg/api"
)

var (
	addr = flag.String("addr", "localhost:50051", "the address of gRPC server")
	host = flag.String("host", "localhost:33566", "the address of host")
	ssl  = flag.Bool("ssl", false, "Use ssl connection")
)

func main() {
	flag.Parse()

	err := core.Init(&core.Config{
		GrpcAddress: *addr,
		SSL:         *ssl,
	})
	if err != nil {
		log.Fatal(err)
	}

	api.Init(*host)
}
