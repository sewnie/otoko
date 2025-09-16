package main

import (
	"context"
	"fmt"
	"os"

	bc "github.com/sewnie/otoko/bandcamp"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
	"golang.org/x/sync/semaphore"
)

type downloader struct {
	ctx context.Context

	c *bc.Client
	s *bc.FanSummary

	b *mpb.Progress
}

// func retry[T any](request func(int) (*T, error)) (*T, error) {
// 	attempt := 0
// 	for {
// 		attempt++
//
// 		result, err := request(attempt)
// 		if err != nil {
// 			if attempt < 10 && errors.Is(err, bc.ErrTooManyRequests) {
// 				time.Sleep(10 * time.Second)
// 				continue
// 			}
// 			return nil, err
// 		}
// 		return result, nil
// 	}
// }

func barStyle() mpb.BarStyleComposer {
	return mpb.BarStyle().Lbound("").Filler("█").Tip("▌").Padding("░").Rbound("")
}

func new(ctx context.Context, cookie string) (*downloader, error) {
	c := bc.New(cookie)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("output dir: %w", err)
	}

	s, err := c.GetCollectionSummary()
	if err != nil {
		return nil, fmt.Errorf("cannot authenticate: %w", err)
	}

	_, ok := bc.Extensions[format]
	if !ok {
		return nil, fmt.Errorf("unknown format: %s", format)
	}

	return &downloader{
		ctx: ctx,
		c:   c,
		s:   s,
		b:   mpb.NewWithContext(ctx, mpb.WithWidth(64)),
	}, nil
}

func (d *downloader) downloadCollection() error {
	redownloads, err := d.c.GetRedownloadItems(d.s.ID)
	if err != nil {
		return fmt.Errorf("get collection: %w", err)
	}

	sem := semaphore.NewWeighted(jobs)

	bar := d.b.New(int64(len(redownloads)), mpb.SpinnerStyle(),
		mpb.PrependDecorators(
			decor.Name(d.s.Collection.Username),
			decor.CountersNoUnit("%d / %d", decor.WC{C: decor.DextraSpace}),
		),
		mpb.BarRemoveOnComplete(),
	)

	for _, rd := range redownloads {
		if err := sem.Acquire(d.ctx, 1); err != nil {
			break
		}

		dup := func() bool {
			name := rd.Name()
			for _, dup := range redownloads {
				// Duplicate found, tell downloader to add the date
				if name == dup.Name() && rd.ID != dup.ID {
					return true
				}
			}
			return false
		}()

		go func() {
			defer func() {
				sem.Release(1)
				bar.Increment()
			}()

			if err := d.downloadItem(&rd, dup); err != nil {
				d.b.Write([]byte(fmt.Sprintf("%s failed: %s\n", rd, err)))
			}
		}()
	}

	defer d.b.Wait()

	if err := sem.Acquire(d.ctx, jobs); err != nil {
		return fmt.Errorf("wait: %w", err)
	}

	return nil
}
