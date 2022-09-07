package main

import (
	"flag"

	"github.com/sath-run/engine/cmd/action"
	"github.com/sath-run/engine/pkg/api"
)

var (
	addr = flag.String("addr", "localhost:50051", "the address of gRPC server")
	host = flag.String("host", "localhost:50551", "the address of host")
)

func main() {
	flag.Parse()
	err := action.Init(*addr)
	if err != nil {
		panic(err)
	}

	api.Init(*host)
}
