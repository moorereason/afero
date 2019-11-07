package regexpfs

import (
	"os"
	"regexp"

	"github.com/spf13/afero"
)

type File struct {
	f  afero.File
	re *regexp.Regexp
}

func (f *File) Close() error {
	return f.f.Close()
}

func (f *File) Read(s []byte) (int, error) {
	return f.f.Read(s)
}

func (f *File) ReadAt(s []byte, o int64) (int, error) {
	return f.f.ReadAt(s, o)
}

func (f *File) Seek(o int64, w int) (int64, error) {
	return f.f.Seek(o, w)
}

func (f *File) Write(s []byte) (int, error) {
	return f.f.Write(s)
}

func (f *File) WriteAt(s []byte, o int64) (int, error) {
	return f.f.WriteAt(s, o)
}

func (f *File) Name() string {
	return f.f.Name()
}

func (f *File) Readdir(c int) (fi []os.FileInfo, err error) {
	var rfi []os.FileInfo
	rfi, err = f.f.Readdir(c)
	if err != nil {
		return nil, err
	}
	for _, i := range rfi {
		if i.IsDir() || f.re.MatchString(i.Name()) {
			fi = append(fi, i)
		}
	}
	return fi, nil
}

func (f *File) Readdirnames(c int) (n []string, err error) {
	fi, err := f.Readdir(c)
	if err != nil {
		return nil, err
	}
	for _, s := range fi {
		n = append(n, s.Name())
	}
	return n, nil
}

func (f *File) Stat() (os.FileInfo, error) {
	return f.f.Stat()
}

func (f *File) Sync() error {
	return f.f.Sync()
}

func (f *File) Truncate(s int64) error {
	return f.f.Truncate(s)
}

func (f *File) WriteString(s string) (int, error) {
	return f.f.WriteString(s)
}
