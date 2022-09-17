package core

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestDockerPull(t *testing.T) {
	err := Init(&Config{
		GrpcAddress: "localhost:50051",
	})

	if err != nil {
		fmt.Printf("%+v\n", err)
		panic(err)
	}

	ctx := context.Background()
	err = PullImage(ctx, &DockerImageConfig{
		Repository: "zengxinzhy/vinadock",
		Tag:        "latest",
		Digest:     "sha256:b5c96c44fcd3b48f30c0dfee99c97bceaa0037b86e15d0f28404d0c1f25dbcfd",
		Uri:        "",
	}, func(text string) {
		var obj gin.H
		if err := json.Unmarshal([]byte(text), &obj); err != nil {
			panic(err)
		}
		log.Println(obj)
	})

	if err != nil {
		log.Printf("%+v\n", err)
		panic(err)
	}
}
