package main

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sewnie/otoko/bandcamp"
)

func extractAlbum(
	item *bandcamp.Item,
	src *os.File, dir string,
) error {
	s, err := src.Stat()
	if err != nil {
		return err
	}

	r, err := zip.NewReader(src, s.Size())
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	trunc := item.BandName + " - " + item.Title + " - "
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			return errors.New("unexpected directory")
		}

		// life - demo two - 05 twelve travel.flac -> '05 twelve travel.flac'
		name := f.Name
		if a, found := strings.CutPrefix(name, trunc); found {
			name = a
		} else {
			// It is unknown if Bandcamp lets artists ship their own files,
			// but these additional album covers are the only things I've
			// noticed present in the archives.
			switch filepath.Ext(f.Name) {
			case ".jpg", ".png":
			default:
				return fmt.Errorf("unknown file %s", f.Name)
			}
		}

		dst, err := os.OpenFile(filepath.Join(dir, name),
			os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		z, err := f.Open()
		if err != nil {
			dst.Close()
			return err
		}

		_, err = io.Copy(dst, z)
		dst.Close()
		z.Close()
		if err != nil {
			return err
		}
	}

	return nil
}
