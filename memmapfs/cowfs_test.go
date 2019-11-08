package memmapfs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/spf13/afero/cowfs"
	"github.com/spf13/afero/fsutil"
	"github.com/spf13/afero/osfs"
	"github.com/spf13/afero/rofs"
)

func TestCopyOnWrite(t *testing.T) {
	osFs := osfs.NewOsFs()
	writeDir, err := fsutil.TempDir(osFs, "", "copy-on-write-test")
	if err != nil {
		t.Fatal("error creating tempDir", err)
	}
	defer osFs.RemoveAll(writeDir)

	compositeFs := cowfs.New(rofs.New(osfs.NewOsFs()), osFs)

	dir := filepath.Join(writeDir, "some/path")

	err = compositeFs.MkdirAll(dir, 0744)
	if err != nil {
		t.Fatal(err)
	}
	_, err = compositeFs.Create(filepath.Join(dir, "newfile"))
	if err != nil {
		t.Fatal(err)
	}

	// https://github.com/spf13/afero/issues/189
	// We want the composite file system to behave like the OS file system
	// on Mkdir and MkdirAll
	for _, fs := range []afero.Fs{osFs, compositeFs} {
		err = fs.Mkdir(dir, 0744)
		if err == nil || !os.IsExist(err) {
			t.Errorf("Mkdir: Got %q for %T", err, fs)
		}

		// MkdirAll does not return an error when the directory already exists
		err = fs.MkdirAll(dir, 0744)
		if err != nil {
			t.Errorf("MkdirAll:  Got %q for %T", err, fs)
		}

	}
}

func TestCopyOnWriteFileInMemMapBase(t *testing.T) {
	base := &Fs{}
	layer := &Fs{}

	if err := fsutil.WriteFile(base, "base.txt", []byte("base"), 0755); err != nil {
		t.Fatalf("Failed to write file: %s", err)
	}

	ufs := cowfs.New(base, layer)

	_, err := ufs.Stat("base.txt")
	if err != nil {
		t.Fatal(err)
	}
}
