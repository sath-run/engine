package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/sath-run/engine/constants"
	"github.com/sath-run/engine/engine/core"
	"github.com/sath-run/engine/engine/server"
)

var dataPath string
var grpcAddrArg string
var sslArg bool
var showVersion bool

func init() {
	flag.StringVar(&dataPath, "data", "", "path of data folder")
	flag.StringVar(&grpcAddrArg, "grpc", "", "grpc address for debug mode")
	flag.BoolVar(&sslArg, "ssl", true, "grpc comunication whether or not using ssl")
	flag.BoolVar(&showVersion, "version", false, "show current version")
}

func main() {
	flag.Parse()

	if showVersion {
		fmt.Println("Sath " + constants.Version)
		return
	}

	sockfile := "/var/run/sath.sock"

	// 若sockfile已存在则删除
	if err := os.Remove(sockfile); err != nil && !errors.Is(err, os.ErrNotExist) {
		panic(err)
	}

	var grpcAddr string = "scheduler.sath.run:50051"
	if len(grpcAddrArg) > 0 {
		grpcAddr = grpcAddrArg
	} else if grpcEnv := os.Getenv("SATH_GRPC"); len(grpcEnv) > 0 {
		grpcAddr = grpcEnv
	}

	ssl := false
	if !sslArg || os.Getenv("SATH_MODE") == "debug" {
		ssl = false
	} else {
		ssl = true
	}

	fmt.Println(grpcAddr, ssl)
	err := core.Init(&core.Config{
		GrpcAddress: grpcAddr,
		SSL:         ssl,
		HostDir:     dataPath,
	})
	if err != nil {
		log.Fatal(err)
	}

	if os.Getenv("SATH_MODE") != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	fmt.Println(os.Getenv("SATH_MODE"))

	// api will block main thread forever
	server.Init(sockfile)
}
