package core

import (
	"context"
	"io"
	"time"

	"github.com/pkg/errors"
	"github.com/sath-run/engine/engine/core/conns"
	pb "github.com/sath-run/engine/engine/core/protobuf"
	"github.com/sath-run/engine/engine/logger"
)

type Heartbeat struct {
	c            *conns.Connection
	reconnecting chan bool
	closing      chan struct{}
}

func NewHeartbeat(c *conns.Connection) *Heartbeat {
	hb := Heartbeat{
		c:            c,
		reconnecting: make(chan bool),
		closing:      make(chan struct{}),
	}
	ticker := time.NewTicker(30 * time.Second)

	var stream pb.Engine_RouteCommandClient
	var err error
	go func() {
		for {
			select {
			case <-hb.closing:
				ticker.Stop()
				close(hb.reconnecting)
				close(hb.closing)
				return
			case <-hb.reconnecting:
				ctx, _ := c.AppendToOutgoingContext(context.Background())
				stream, err = c.RouteCommand(ctx)
				if err != nil {
					logger.Error(err)
				}
			case <-ticker.C:
				if s := stream; s != nil {
					if err = s.Send(&pb.CommandResponse{}); errors.Is(err, io.EOF) {
						// if stream is disconnected, reconnect
						hb.Connect(false)
					} else if err != nil {
						logger.Error(err)
					}
				} else {
					hb.Connect(false)
				}
			}
		}
	}()
	hb.Connect(true)
	return &hb
}

func (hb *Heartbeat) Connect(wait bool) bool {
	if wait {
		hb.reconnecting <- true
		return true
	} else {
		select {
		case hb.reconnecting <- true:
			return true
		default:
			return false
		}
	}
}

func (hb *Heartbeat) Close() {
	hb.closing <- struct{}{}
}
