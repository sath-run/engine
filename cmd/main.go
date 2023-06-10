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

func init() {
	flag.StringVar(&dataPath, "data", "", "path of data folder")
}

func main() {
	flag.Parse()
	grpcAddr := "scheduler.sath.run:50051"
	ssl := true
	host := "localhost:33566"
	if strings.ToLower(os.Getenv("SATH_MODE")) == "debug" {
		ssl = false
		grpcAddr = "localhost:50051"
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	err := core.Init(&core.Config{
		GrpcAddress: grpcAddr,
		SSL:         ssl,
	})
	if err != nil {
		log.Fatal(err)
	}

	if strings.ToLower(os.Getenv("SATH_MODE")) == "docker" {
		err := core.Start()
		if err != nil {
			log.Fatal(err)
		}
	}

	api.Init(host)

}
