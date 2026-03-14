package main

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func extractAlbum(src *os.File, albumDir string) error {
	s, err := src.Stat()
	if err != nil {
		return err
	}

	albumName := filepath.Base(albumDir)

	r, err := zip.NewReader(src, s.Size())
	if err != nil {
		return err
	}

	if err := os.MkdirAll(albumDir, 0o755); err != nil {
		return err
	}

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			return errors.New("unexpected directory")
		}

		// life - demo two - 05 twelve travel.flac -> '05 twelve travel.flac'
		name := f.Name
		if i := strings.LastIndex(name, albumName+" - "); i > 0 {
			name = name[i+len(albumName)+3:]
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

		dst, err := os.OpenFile(filepath.Join(albumDir, name),
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
