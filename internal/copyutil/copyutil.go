package copyutil

import (
	"io"
	"path/filepath"
	"syscall"

	"github.com/spf13/afero"
	"github.com/spf13/afero/fsutil"
)

func CopyToLayer(base afero.Fs, layer afero.Fs, name string) error {
	bfh, err := base.Open(name)
	if err != nil {
		return err
	}
	defer bfh.Close()

	// First make sure the directory exists
	exists, err := fsutil.Exists(layer, filepath.Dir(name))
	if err != nil {
		return err
	}
	if !exists {
		err = layer.MkdirAll(filepath.Dir(name), 0777) // FIXME?
		if err != nil {
			return err
		}
	}

	// Create the file on the overlay
	lfh, err := layer.Create(name)
	if err != nil {
		return err
	}
	n, err := io.Copy(lfh, bfh)
	if err != nil {
		// If anything fails, clean up the file
		layer.Remove(name)
		lfh.Close()
		return err
	}

	bfi, err := bfh.Stat()
	if err != nil || bfi.Size() != n {
		layer.Remove(name)
		lfh.Close()
		return syscall.EIO
	}

	err = lfh.Close()
	if err != nil {
		layer.Remove(name)
		lfh.Close()
		return err
	}
	return layer.Chtimes(name, bfi.ModTime(), bfi.ModTime())
}
