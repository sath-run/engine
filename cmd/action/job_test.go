package action

import (
	"context"
	"fmt"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/docker/docker/api/types"
)

func TestDocker(t *testing.T) {
	Init("localhost:50051")
	ctx := context.Background()
	images, err := dockerClient.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		panic(err)
	}
	spew.Dump(images)
}

func TestSingleJob(t *testing.T) {
	Init("localhost:50051")
	res, err := RunSingleJob()
	fmt.Println(res, err)
}
