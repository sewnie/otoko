package main

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	bc "github.com/sewnie/otoko/bandcamp"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

var ErrDownloadBadEmail = errors.New("purchased item email isn't added to your bandcamp fan account, check the email by going to the download page manually and add it as necessary")

var filenameSanitizer = strings.NewReplacer(
	"/", "_",
	"?", "_",
	"<", "_",
	">", "_",
	":", "_",
	"|", "_",
	"\"", "_",
	"*", "_",
)

func path(i *bc.RedownloadItem, addID bool) (path string) {
	band := filenameSanitizer.Replace(i.BandName)
	title := filenameSanitizer.Replace(i.Title)

	if addID {
		title += " (" + strconv.FormatInt(int64(i.ID), 10) + ")"
	}

	path = filepath.Join(dir, band, title)

	if i.Type == bc.Track {
		path += bc.Extensions[format]
	}
	return
}

func (d *downloader) request(i *bc.DownloadItem) (*http.Response, error) {
	req, err := http.NewRequestWithContext(d.ctx, "GET", i.URL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := d.c.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusForbidden {
		return nil, errors.New("requested too many downloads or denied permission")
	} else if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %s", resp.Status)
	}

	if resp.ContentLength == 0 {
		return nil, errors.New("expected file")
	}

	return resp, nil
}

func (d *downloader) downloadItem(redownload *bc.RedownloadItem, addID bool) error {
	name := path(redownload, addID)

	download, err := d.c.GetDownloadItem(redownload, format)
	if err != nil {
		return fmt.Errorf("fetch %s download: %w", download, err)
	}

	if _, err = os.Stat(name); err == nil {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	extract := name + ".zip"
	if download.Type == bc.Track {
		extract = name
	}

	f, err := os.Create(extract)
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}
	defer func() {
		f.Close()
		if download.Type == bc.Album {
			os.Remove(extract)
		}
	}()

	resp, err := d.request(download)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	b := d.b.New(resp.ContentLength, barStyle(),
		mpb.PrependDecorators(
			decor.Name(download.String(), decor.WCSyncSpaceR),
			decor.Counters(decor.SizeB1024(0), "% .1f / % .1f", decor.WCSyncSpaceR),
		),
		mpb.AppendDecorators(
			decor.AverageSpeed(decor.SizeB1024(0), "%.2f"),
		),
		mpb.BarRemoveOnComplete(),
	)
	r := b.ProxyReader(resp.Body)
	defer b.Abort(true)
	defer r.Close()

	if _, err := io.Copy(f, r); err != nil {
		return err
	}

	if download.Type == bc.Track {
		return nil
	}

	if err := ExtractAlbum(f, name); err != nil {
		if errors.Is(err, zip.ErrFormat) {
			return fmt.Errorf("%w: %s", ErrDownloadBadEmail, download.Name())
		}
		return fmt.Errorf("%s: %w", name, err)
	}

	return nil
}
