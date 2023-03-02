package core_test

import (
	"log"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/sath-run/engine/cmd/core"
)

func TestFileUpload(t *testing.T) {
	err := core.Init(&core.Config{
		GrpcAddress: "localhost:50051",
		SSL:         false,
	})
	if err != nil {
		log.Fatal(err)
	}
	files, err := core.ProcessOutputs(".", "587962", []string{"docker.go"})
	if err != nil {
		log.Fatal(err)
	}
	spew.Dump(files)
}
