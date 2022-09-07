package main

import (
	"flag"

	"github.com/sath-run/engine/cmd/action"
)

var (
	addr = flag.String("addr", "121.196.174.105:50051", "the address to connect to")
	dimg = flag.String("dimg", "zengxinzhy/vf", "docker image")
)

func main() {
	flag.Parse()
	err := action.Init(*addr)
	if err != nil {
		panic(err)
	}
}
