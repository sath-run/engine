package main

import (
	"errors"
	"flag"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/sath-run/engine/constants"
	"github.com/sath-run/engine/engine/core"
	"github.com/sath-run/engine/engine/logger"
	"github.com/sath-run/engine/engine/server"
	"github.com/sath-run/engine/meta"
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
		log.Fatalln("Sath " + constants.Version)
	}

	if err := logger.Init(); err != nil {
		log.Fatalf("fail to init logger, %+v\n", err)
	}

	if err := meta.Init(); err != nil {
		log.Fatalf("fail to init DB, %+v\n", err)
	}

	sockfile := "/var/run/sath/engine.sock"

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

	if err := core.Init(&core.Config{
		GrpcAddress: grpcAddr,
		SSL:         ssl,
		DataDir:     dataPath,
	}); err != nil {
		log.Fatal(err)
	}

	if os.Getenv("SATH_MODE") != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	// api will block main thread forever
	server.Init(sockfile)
}
