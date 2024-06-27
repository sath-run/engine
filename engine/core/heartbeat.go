package core

import (
	"context"
	"io"
	"time"

	"github.com/pkg/errors"
	pb "github.com/sath-run/engine/engine/core/protobuf"
	"github.com/sath-run/engine/engine/logger"
)

type Heartbeat struct {
	reconn_chan chan bool
}

func NewHeartbeat(ctx context.Context, client pb.EngineClient) *Heartbeat {
	hb := Heartbeat{
		reconn_chan: make(chan bool),
	}
	ticker := time.NewTicker(30 * time.Second)

	var stream pb.Engine_RouteCommandClient
	var err error

	go func() {
		for {
			select {
			case <-hb.reconn_chan:
				stream, err = client.RouteCommand(ctx)
				if err != nil {
					logger.Error(err)
				}
			case <-ticker.C:
				if err = stream.Send(&pb.CommandResponse{}); errors.Is(err, io.EOF) {
					// if stream is disconnected, reconnect
					hb.Connect()
				} else if err != nil {
					logger.Error(err)
				}
			}
		}
	}()
	hb.Connect()
	return &hb
}

func (hb *Heartbeat) Connect() bool {
	select {
	case hb.reconn_chan <- true:
		return true
	default:
		return false
	}
}
