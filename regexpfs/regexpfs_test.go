package regexpfs

import (
	"regexp"
	"testing"

	"github.com/spf13/afero/fsutil"
	"github.com/spf13/afero/memmapfs"
	"github.com/spf13/afero/rofs"
)

func TestFilter(t *testing.T) {
	fs := New(&memmapfs.MemMapFs{}, regexp.MustCompile(`\.txt$`))
	_, err := fs.Create("/file.html")
	if err == nil {
		t.Errorf("Did not fail to create file")
	}
}

func TestFilterROChain(t *testing.T) {
	rofs := rofs.NewReadOnlyFs(&memmapfs.MemMapFs{})
	fs := &Fs{re: regexp.MustCompile(`\.txt$`), source: rofs}
	_, err := fs.Create("/file.txt")
	if err == nil {
		t.Errorf("Did not fail to create file")
	}
}

func TestFilterReadDir(t *testing.T) {
	mfs := &memmapfs.MemMapFs{}
	fs1 := &Fs{re: regexp.MustCompile(`\.txt$`), source: mfs}
	fs := &Fs{re: regexp.MustCompile(`^a`), source: fs1}

	mfs.MkdirAll("/dir/sub", 0777)
	for _, name := range []string{"afile.txt", "afile.html", "bfile.txt"} {
		for _, dir := range []string{"/dir/", "/dir/sub/"} {
			fh, _ := mfs.Create(dir + name)
			fh.Close()
		}
	}

	files, _ := fsutil.ReadDir(fs, "/dir")
	if len(files) != 2 { // afile.txt, sub
		t.Errorf("Got wrong number of files: %#v", files)
	}

	f, _ := fs.Open("/dir/sub")
	names, _ := f.Readdirnames(-1)
	if len(names) != 1 {
		t.Errorf("Got wrong number of names: %v", names)
	}
}
