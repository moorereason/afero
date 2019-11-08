package rofs

import (
	"os"
	"syscall"
	"time"

	"github.com/spf13/afero"
	"github.com/spf13/afero/fsutil"
)

var _ afero.Lstater = (*Fs)(nil)

type Fs struct {
	source afero.Fs
}

func New(source afero.Fs) *Fs {
	return &Fs{source: source}
}

func (r *Fs) ReadDir(name string) ([]os.FileInfo, error) {
	return fsutil.ReadDir(r.source, name)
}

func (r *Fs) Chtimes(n string, a, m time.Time) error {
	return syscall.EPERM
}

func (r *Fs) Chmod(n string, m os.FileMode) error {
	return syscall.EPERM
}

func (r *Fs) Name() string {
	return "rofs"
}

func (r *Fs) Stat(name string) (os.FileInfo, error) {
	return r.source.Stat(name)
}

func (r *Fs) LstatIfPossible(name string) (os.FileInfo, bool, error) {
	if lsf, ok := r.source.(afero.Lstater); ok {
		return lsf.LstatIfPossible(name)
	}
	fi, err := r.Stat(name)
	return fi, false, err
}

func (r *Fs) Rename(o, n string) error {
	return syscall.EPERM
}

func (r *Fs) RemoveAll(p string) error {
	return syscall.EPERM
}

func (r *Fs) Remove(n string) error {
	return syscall.EPERM
}

func (r *Fs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	if flag&(os.O_WRONLY|syscall.O_RDWR|os.O_APPEND|os.O_CREATE|os.O_TRUNC) != 0 {
		return nil, syscall.EPERM
	}
	return r.source.OpenFile(name, flag, perm)
}

func (r *Fs) Open(n string) (afero.File, error) {
	return r.source.Open(n)
}

func (r *Fs) Mkdir(n string, p os.FileMode) error {
	return syscall.EPERM
}

func (r *Fs) MkdirAll(n string, p os.FileMode) error {
	return syscall.EPERM
}

func (r *Fs) Create(n string) (afero.File, error) {
	return nil, syscall.EPERM
}
