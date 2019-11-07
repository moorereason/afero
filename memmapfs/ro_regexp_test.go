package memmapfs

import (
	"testing"

	"github.com/spf13/afero/readonlyfs"
)

func TestFilterReadOnly(t *testing.T) {
	mfs := &MemMapFs{}
	fs := readonlyfs.NewReadOnlyFs(mfs)
	_, err := fs.Create("/file.txt")
	if err == nil {
		t.Errorf("Did not fail to create file")
	}
	// t.Logf("ERR=%s", err)
}

func TestFilterReadonlyRemoveAndRead(t *testing.T) {
	mfs := &MemMapFs{}
	fh, err := mfs.Create("/file.txt")
	fh.Write([]byte("content here"))
	fh.Close()

	fs := readonlyfs.NewReadOnlyFs(mfs)
	err = fs.Remove("/file.txt")
	if err == nil {
		t.Errorf("Did not fail to remove file")
	}

	fh, err = fs.Open("/file.txt")
	if err != nil {
		t.Errorf("Failed to open file: %s", err)
	}

	buf := make([]byte, len("content here"))
	_, err = fh.Read(buf)
	fh.Close()
	if string(buf) != "content here" {
		t.Errorf("Failed to read file: %s", err)
	}

	err = mfs.Remove("/file.txt")
	if err != nil {
		t.Errorf("Failed to remove file")
	}

	fh, err = fs.Open("/file.txt")
	if err == nil {
		fh.Close()
		t.Errorf("File still present")
	}
}
