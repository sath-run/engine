package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/sath-run/engine/constants"
	"github.com/sath-run/engine/engine/core"
	"github.com/sath-run/engine/engine/logger"
	"github.com/sath-run/engine/engine/server"
	"github.com/sath-run/engine/meta"
)

var dataPath string
var grpcAddrArg string
var sockArg string
var sslArg bool
var showVersion bool

func init() {
	flag.StringVar(&dataPath, "data", "", "path of data folder")
	flag.StringVar(&grpcAddrArg, "grpc", "", "grpc address for debug mode")
	flag.BoolVar(&sslArg, "ssl", true, "grpc comunication whether or not using ssl")
	flag.BoolVar(&showVersion, "version", false, "show current version and exit")
}

func main() {
	flag.Parse()

	if showVersion {
		fmt.Println("Sath " + constants.Version)
		return
	}

	if err := logger.Init(); err != nil {
		log.Fatalf("fail to init logger, %+v\n", err)
	}

	if err := meta.Init(); err != nil {
		log.Fatalf("fail to init DB, %+v\n", err)
	}

	// sockfile := filepath.Join(utils.ExecutableDir, "sath.sock")
	sockfile := "/var/run/sath.sock"

	// servers should unlink the socket pathname prior to binding it.
	// https://troydhanson.github.io/network/Unix_domain_sockets.html
	syscall.Unlink(sockfile)

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
	engine, err := core.Default(&core.Config{
		GrpcAddress: grpcAddr,
		SSL:         ssl,
		DataDir:     dataPath,
	})
	if err != nil {
		log.Fatal(err)
	}

	if os.Getenv("SATH_MODE") != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	// api will block main thread forever
	server.Init(sockfile, engine)
}
