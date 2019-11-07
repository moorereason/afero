package regexpfs

import (
	"os"
	"regexp"
	"syscall"
	"time"

	"github.com/spf13/afero"
	"github.com/spf13/afero/fsutil"
)

// The Fs filters files (not directories) by regular expression. Only
// files matching the given regexp will be allowed, all others get a ENOENT error (
// "No such file or directory").
//
type Fs struct {
	re     *regexp.Regexp
	source afero.Fs
}

func New(source afero.Fs, re *regexp.Regexp) afero.Fs {
	return &Fs{source: source, re: re}
}

func (r *Fs) matchesName(name string) error {
	if r.re == nil {
		return nil
	}
	if r.re.MatchString(name) {
		return nil
	}
	return syscall.ENOENT
}

func (r *Fs) dirOrMatches(name string) error {
	dir, err := fsutil.IsDir(r.source, name)
	if err != nil {
		return err
	}
	if dir {
		return nil
	}
	return r.matchesName(name)
}

func (r *Fs) Chtimes(name string, a, m time.Time) error {
	if err := r.dirOrMatches(name); err != nil {
		return err
	}
	return r.source.Chtimes(name, a, m)
}

func (r *Fs) Chmod(name string, mode os.FileMode) error {
	if err := r.dirOrMatches(name); err != nil {
		return err
	}
	return r.source.Chmod(name, mode)
}

func (r *Fs) Name() string {
	return "regexpfs"
}

func (r *Fs) Stat(name string) (os.FileInfo, error) {
	if err := r.dirOrMatches(name); err != nil {
		return nil, err
	}
	return r.source.Stat(name)
}

func (r *Fs) Rename(oldname, newname string) error {
	dir, err := fsutil.IsDir(r.source, oldname)
	if err != nil {
		return err
	}
	if dir {
		return nil
	}
	if err := r.matchesName(oldname); err != nil {
		return err
	}
	if err := r.matchesName(newname); err != nil {
		return err
	}
	return r.source.Rename(oldname, newname)
}

func (r *Fs) RemoveAll(p string) error {
	dir, err := fsutil.IsDir(r.source, p)
	if err != nil {
		return err
	}
	if !dir {
		if err := r.matchesName(p); err != nil {
			return err
		}
	}
	return r.source.RemoveAll(p)
}

func (r *Fs) Remove(name string) error {
	if err := r.dirOrMatches(name); err != nil {
		return err
	}
	return r.source.Remove(name)
}

func (r *Fs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	if err := r.dirOrMatches(name); err != nil {
		return nil, err
	}
	return r.source.OpenFile(name, flag, perm)
}

func (r *Fs) Open(name string) (afero.File, error) {
	dir, err := fsutil.IsDir(r.source, name)
	if err != nil {
		return nil, err
	}
	if !dir {
		if err := r.matchesName(name); err != nil {
			return nil, err
		}
	}
	f, err := r.source.Open(name)
	return &File{f: f, re: r.re}, nil
}

func (r *Fs) Mkdir(n string, p os.FileMode) error {
	return r.source.Mkdir(n, p)
}

func (r *Fs) MkdirAll(n string, p os.FileMode) error {
	return r.source.MkdirAll(n, p)
}

func (r *Fs) Create(name string) (afero.File, error) {
	if err := r.matchesName(name); err != nil {
		return nil, err
	}
	return r.source.Create(name)
}
