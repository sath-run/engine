package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/sath-run/engine/api"
	"github.com/sath-run/engine/constants"
	"github.com/sath-run/engine/daemon"
	"github.com/sath-run/engine/meta"
	"github.com/sath-run/engine/utils"
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

	if err := meta.Init(); err != nil {
		log.Fatal().Err(err).Send()
	}

	sockfile := utils.SockFile()
	os.MkdirAll(filepath.Dir(sockfile), os.ModePerm)

	// servers should unlink the socket pathname prior to binding it.
	// https://troydhanson.github.io/network/Unix_domain_sockets.html
	syscall.Unlink(sockfile)

	var grpcAddr string = "scheduler.sath.run:50051"
	if len(grpcAddrArg) > 0 {
		grpcAddr = grpcAddrArg
	} else if grpcEnv := os.Getenv("SATH_GRPC"); len(grpcEnv) > 0 {
		grpcAddr = grpcEnv
	}

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	ssl := false
	if !sslArg || os.Getenv("SATH_MODE") == "debug" {
		ssl = false
		zerolog.SetGlobalLevel(zerolog.Level(constants.LogLevel))
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	} else {
		ssl = true
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	engine, err := daemon.Default(ctx, &daemon.Config{
		GrpcAddress: grpcAddr,
		SSL:         ssl,
		DataDir:     dataPath,
	})
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	if os.Getenv("SATH_MODE") != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	// api will block main thread forever
	api.Init(sockfile, engine)
}
