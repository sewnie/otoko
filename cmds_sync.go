package main

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/sewnie/otoko/bandcamp"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
	"golang.org/x/sync/semaphore"
)

type syncCmd struct {
	Jobs   int64  `kong:"short=j,help='Count of permitted concurrent downloads',default=6"`
	Format string `kong:"short=f,help='Audio format to download; one of ${enum}',default=mp3-320,enum='mp3-v0,mp3-320,flac,aac-hi,vorbis,alac,wav,aiff-lossless'"`
	Force  bool   `kong:"help='Overwrite existing files even if they already exist locally'"`
	Strict bool   `kong:"help='Ensure file format when checking album or track download validity'"`
	DryRun bool   `kong:"short=n,help='Only evaluate collection and report status'"`

	Output   string            `kong:"arg,help='Path to the directory where tracks and albums will be saved',type=path"`
	Tralbums []bandcamp.ItemID `kong:"arg,optional,help='Specific track or album IDs to sync, defaults to all'"`
}

func (cmd *syncCmd) Run(c *Client) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	items, err := c.GetCollection(c.Fan.ID)
	if err != nil {
		return err
	}
	if len(cmd.Tralbums) > 0 {
		items = slices.DeleteFunc(items, func(item bandcamp.Item) bool {
			ret := !slices.Contains(cmd.Tralbums, item.ID)
			return ret
		})
	}

	names := make(map[*bandcamp.Item]string, len(items))
	for i := range items {
		item := &items[i]
		name := filepath.Join(
			sanitizer.Replace(item.BandName),
			sanitizer.Replace(item.Title))

		for j := range items {
			// Duplicate found, add ID to the filename, as I have
			// some tralbums that are released in the same day, with
			// the same name :/
			if item.BandName == items[j].BandName &&
				item.Title == items[j].Title &&
				item.ID != items[j].ID {
				name += " (" + strconv.FormatInt(int64(item.ID), 10) + ")"
			}
		}

		if item.Type == bandcamp.ItemTypeTrack {
			name += bandcamp.Extensions[cmd.Format]
		}
		names[item] = name
	}

	sem := semaphore.NewWeighted(cmd.Jobs)

	prog := mpb.NewWithContext(ctx, mpb.WithWidth(64))
	log.SetOutput(prog)

	bar := prog.New(int64(len(names)), mpb.SpinnerStyle(),
		mpb.PrependDecorators(
			decor.Name(c.Fan.Username),
			decor.CountersNoUnit("%d / %d", decor.WC{C: decor.DextraSpace}),
		),
		mpb.BarRemoveOnComplete(),
	)

	for item, name := range names {
		if err := sem.Acquire(ctx, 1); err != nil {
			break
		}

		go func() {
			defer func() {
				sem.Release(1)
				bar.Increment()
			}()

			if err := cmd.Download(c.Client, name, item, prog); err != nil {
				fmt.Fprintf(prog, "%s failed: %s\n", name, err)
			}
		}()
	}

	if err := sem.Acquire(ctx, cmd.Jobs); err != nil {
		return fmt.Errorf("wait: %w", err)
	}

	prog.Wait()

	return nil
}

// Bandcamp has their own file sanitizer, with unknown replacements.
// Trial and error is the only way to figure out which characters
// are part of it.
var sanitizer = strings.NewReplacer(
	"//", "-",
	"/", "-",
	"?", "-",
	"<", "-",
	">", "-",
	":", "-",
	"|", "-",
	"\"", "-",
	"*", "-",
)

func (cmd *syncCmd) valid(name string, item *bandcamp.Item) bool {
	if item.Type == bandcamp.ItemTypeTrack {
		if !cmd.Strict {
			// Rather than stripping prefix, optimize for string
			// length
			m, _ := filepath.Glob(
				name[:len(name)-len(bandcamp.Extensions[cmd.Format])+1] + "*")
			return len(m) != 0
		}

		_, err := os.Stat(name)
		return err != nil
	}

	if !cmd.Strict {
		f, err := os.ReadDir(name)
		if err != nil {
			return false
		}

		return len(f)-1 >= len(item.Tracks)
	}

	for _, track := range item.Tracks {
		name := filepath.Join(name, fmt.Sprintf("%02d %s%s",
			track.Number, sanitizer.Replace(track.Title),
			bandcamp.Extensions[cmd.Format]))
		_, err := os.Stat(name)
		if err != nil {
			return false
		}
	}
	return true
}

func (cmd *syncCmd) Download(
	client *bandcamp.Client,
	name string,
	item *bandcamp.Item,
	prog *mpb.Progress,
) error {
	name = filepath.Join(cmd.Output, name)

	// Prefer to glob file checks incase albums or tracks are
	// transcoded as another format. Obviously this works against
	// the user in the case that they want to download in new format,
	// but this is a fine sacrifice to make.
	if cmd.valid(name, item) {
		return nil
	}

	if cmd.DryRun {
		log.Printf("Would download %s (%s)", filepath.Base(name), item)
		return nil
	} else if item.Type == bandcamp.ItemTypeAlbum {
		if err := os.RemoveAll(name); err != nil {
			return fmt.Errorf("corrupt remove: %w", err)
		}
	}

	download, err := client.GetItemDownload(item, cmd.Format)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}

	req, err := http.NewRequest("GET", download.URL, nil)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &bandcamp.StatusError{StatusCode: resp.StatusCode}
	}

	if resp.ContentLength == 0 {
		return fs.ErrNotExist
	}

	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	extract := name
	if item.Type == bandcamp.ItemTypeAlbum {
		extract += ".zip"
		defer os.Remove(extract)
	}

	f, err := os.Create(extract)
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}
	defer f.Close()

	bar := prog.New(resp.ContentLength, barStyle(),
		mpb.PrependDecorators(
			decor.Name(item.String(), decor.WCSyncSpaceR),
			decor.Counters(decor.SizeB1024(0), "% .1f / % .1f", decor.WCSyncSpaceR),
		),
		mpb.AppendDecorators(
			decor.AverageSpeed(decor.SizeB1024(0), "%.2f"),
		),
		mpb.BarRemoveOnComplete(),
	)
	r := bar.ProxyReader(resp.Body)
	defer bar.Abort(true) // Sometimes I/O locks up?
	defer r.Close()

	if _, err := io.Copy(f, r); err != nil {
		return fmt.Errorf("copy: %w", err)
	}

	// Item is already "extracted" as it the download file is
	// just the encoded track
	if item.Type == bandcamp.ItemTypeTrack {
		return nil
	}

	err = extractAlbum(f, name)
	if errors.Is(err, zip.ErrFormat) {
		return fmt.Errorf("email missing from bandcamp account: %s", download.Email)
	}
	return err
}

func barStyle() mpb.BarStyleComposer {
	return mpb.BarStyle().Lbound("").Filler("█").Tip("▌").Padding("░").Rbound("")
}
