package main

import (
	"flag"
	"log"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sath-run/engine/cmd/core"
	"github.com/sath-run/engine/pkg/api"
)

var dataPath string
var debugGrpc string

func init() {
	flag.StringVar(&dataPath, "data", "", "path of data folder")
	flag.StringVar(&debugGrpc, "debugGrpc", "localhost", "grpc address for debug mode")
}

func main() {
	flag.Parse()
	grpcAddr := "scheduler.sath.run:50051"
	ssl := true
	// host := "localhost:33566"
	sockfile := "/var/run/sathcli.sock"
	if strings.ToLower(os.Getenv("SATH_MODE")) == "debug" {
		ssl = false
		grpcAddr = debugGrpc + ":50051"
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	// 若sockfile已存在则删除
	os.Remove(sockfile)

	err := core.Init(&core.Config{
		GrpcAddress: grpcAddr,
		SSL:         ssl,
		HostDir:     dataPath,
	})
	if err != nil {
		log.Fatal(err)
	}

	// if strings.ToLower(os.Getenv("SATH_MODE")) == "docker" {
	// 	err := core.Start()
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// }

	api.Init(sockfile)

}
