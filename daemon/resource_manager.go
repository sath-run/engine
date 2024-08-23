package daemon

import (
	"context"
	"os"
	"sync"

	"github.com/cavaliergopher/grab/v3"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type ResourceManager struct {
	mu          sync.Mutex
	downloaders map[string]*Downloader
}

func NewResourceManager() *ResourceManager {
	return &ResourceManager{
		downloaders: map[string]*Downloader{},
	}
}

func (rm *ResourceManager) Download(ctx context.Context, dst string, url string) *Downloader {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if downloader, ok := rm.downloaders[dst]; ok {
		return downloader
	} else {
		// TODO: clean up downloader after some period of time
		downloader := newDownloader(ctx, dst, url)
		rm.downloaders[dst] = downloader
		return downloader
	}

}

type Downloader struct {
	resp   *grab.Response
	err    error
	logger zerolog.Logger
	Done   chan struct{}
}

func newDownloader(ctx context.Context, dst string, url string) *Downloader {
	tmp := dst + ".sath_tmp"
	client := grab.NewClient()
	req, _ := grab.NewRequest(tmp, url)

	// start download
	resp := client.Do(req)

	dld := &Downloader{
		resp:   resp,
		err:    nil,
		logger: log.With().Str("component", "resource_manager").Str("dst", dst).Logger(),
		Done:   make(chan struct{}),
	}

	dld.logger.Trace().Msg("downloader started")

	go func() {
		select {
		case <-ctx.Done():
			resp.Cancel()
		case <-resp.Done:
		}
		if err := resp.Err(); err != nil {
			dld.err = err
		} else if err := os.Rename(tmp, dst); err != nil {
			dld.err = err
		}
		close(dld.Done)
	}()

	return dld
}

func (dld *Downloader) Total() int64 {
	return dld.resp.Size()
}

func (dld *Downloader) Current() int64 {
	return dld.resp.BytesComplete()
}

func (dld *Downloader) Progress() float64 {
	return dld.resp.Progress()
}

func (dld *Downloader) Err() error {
	return dld.err
}

func (dld *Downloader) Cancel() error {
	return dld.resp.Cancel()
}
