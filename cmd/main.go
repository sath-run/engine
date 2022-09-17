package main

import (
	"flag"

	"github.com/sath-run/engine/cmd/core"
	"github.com/sath-run/engine/pkg/api"
)

var (
	addr = flag.String("addr", "localhost:50051", "the address of gRPC server")
	host = flag.String("host", "localhost:33566", "the address of host")
)

func main() {
	flag.Parse()
	err := core.Init(&core.Config{
		GrpcAddress: *addr,
	})
	if err != nil {
		panic(err)
	}

	api.Init(*host)
}
